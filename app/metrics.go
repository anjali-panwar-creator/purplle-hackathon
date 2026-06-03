package main

import (
	"net/http"
	"store-intelligence/db"
	"store-intelligence/models"

	"github.com/gin-gonic/gin"
)

type MetricsResponse struct {
	UniqueVisitors  int64              `json:"unique_visitors"`
	CurrentVisitors int64              `json:"current_visitors"`
	ConversionRate  float64            `json:"conversion_rate"`
	AvgDwellPerZone map[string]float64 `json:"avg_dwell_per_zone"`
	QueueDepth      int                `json:"queue_depth"`
	AbandonmentRate float64            `json:"abandonment_rate"`
}

// getMetrics computes the real-time business metrics for a given store for today.
func getMetrics(c *gin.Context) {
	storeID := c.Param("id")

	var response MetricsResponse

	// 1. Unique Visitors (excluding staff) for today.
	// Since this is a hackathon, we assume "today" means all data in the DB for this store.
	db.DB.Model(&models.Event{}).
		Where("store_id = ? AND is_staff = ?", storeID, false).
		Distinct("visitor_id").
		Count(&response.UniqueVisitors)

	// 1b. Current Visitors (Active in the store right now)
	// We count people who have entered (ENTRY or REENTRY) but have not exited (EXIT).
	var entryCount int64
	var reentryCount int64
	var exitCount int64
	db.DB.Model(&models.Event{}).Where("store_id = ? AND event_type = ?", storeID, "ENTRY").Count(&entryCount)
	db.DB.Model(&models.Event{}).Where("store_id = ? AND event_type = ?", storeID, "REENTRY").Count(&reentryCount)
	db.DB.Model(&models.Event{}).Where("store_id = ? AND event_type = ?", storeID, "EXIT").Count(&exitCount)
	response.CurrentVisitors = (entryCount + reentryCount) - exitCount

	// 2. Queue Depth
	// The current queue depth is the number of visitors who have a BILLING_QUEUE_JOIN event
	// but DO NOT have a subsequent BILLING_QUEUE_ABANDON or EXIT event within the same session.
	// For simplicity in this demo, let's just count the raw latest queue depth from the last BILLING event.
	var lastQueueEvent models.Event
	db.DB.Where("store_id = ? AND event_type = ?", storeID, "BILLING_QUEUE_JOIN").
		Order("timestamp desc").
		First(&lastQueueEvent)
	if lastQueueEvent.Metadata.QueueDepth != nil {
		response.QueueDepth = *lastQueueEvent.Metadata.QueueDepth
	}

	// 3. Average Dwell Per Zone
	// Query to aggregate dwell_ms by zone_id
	type DwellResult struct {
		ZoneID   string  `gorm:"column:zone_id"`
		AvgDwell float64 `gorm:"column:avg_dwell"`
	}
	var dwellResults []DwellResult
	db.DB.Model(&models.Event{}).
		Select("zone_id, AVG(dwell_ms) as avg_dwell").
		Where("store_id = ? AND is_staff = ? AND event_type = ? AND zone_id IS NOT NULL", storeID, false, "ZONE_EXIT").
		Group("zone_id").
		Scan(&dwellResults)

	response.AvgDwellPerZone = make(map[string]float64)
	for _, dr := range dwellResults {
		response.AvgDwellPerZone[dr.ZoneID] = dr.AvgDwell
	}

	// 4. Conversion Rate & Abandonment Rate
	// 5. Conversion Rate (POS Integration)
	if response.UniqueVisitors > 0 {
		var convertedVisitors int64
		query := `
			SELECT COUNT(DISTINCT e.visitor_id)
			FROM events e
			INNER JOIN pos_transactions p ON p.store_id = e.store_id
			WHERE e.store_id = ?
			  AND e.is_staff = 0
			  AND e.zone_id = 'BILLING_ZONE'
			  AND p.timestamp >= e.timestamp 
			  AND p.timestamp <= datetime(e.timestamp, '+5 minutes')
		`
		db.DB.Raw(query, storeID).Scan(&convertedVisitors)
		response.ConversionRate = (float64(convertedVisitors) / float64(response.UniqueVisitors)) * 100
	} else {
		response.ConversionRate = 0.0
	}
	response.AbandonmentRate = 0.0

	c.JSON(http.StatusOK, response)
}
