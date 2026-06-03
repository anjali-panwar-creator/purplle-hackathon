package main

import (
	"net/http"
	"store-intelligence/db"
	"store-intelligence/models"

	"github.com/gin-gonic/gin"
)

type HeatmapZone struct {
	ZoneID         string  `json:"zone_id"`
	VisitFrequency int64   `json:"visit_frequency"`
	AvgDwell       float64 `json:"avg_dwell"`
	IntensityScore int     `json:"intensity_score"` // 0-100 normalized
}

type HeatmapResponse struct {
	DataConfidence string        `json:"data_confidence"` // "HIGH" or "LOW"
	Zones          []HeatmapZone `json:"zones"`
}

// getHeatmap computes zone visit frequencies and normalizes them for frontend rendering.
func getHeatmap(c *gin.Context) {
	storeID := c.Param("id")
	var response HeatmapResponse

	// Check total unique sessions to set data confidence flag
	var totalSessions int64
	db.DB.Model(&models.Event{}).
		Where("store_id = ?", storeID).
		Distinct("visitor_id").
		Count(&totalSessions)

	if totalSessions < 20 {
		response.DataConfidence = "LOW"
	} else {
		response.DataConfidence = "HIGH"
	}

	// Group by zone to get frequency and dwell
	type HeatmapResult struct {
		ZoneID    string
		Frequency int64
		AvgDwell  float64
	}
	var results []HeatmapResult

	db.DB.Model(&models.Event{}).
		Select("zone_id, COUNT(DISTINCT visitor_id) as frequency, AVG(dwell_ms) as avg_dwell").
		Where("store_id = ? AND is_staff = ? AND zone_id IS NOT NULL", storeID, false).
		Group("zone_id").
		Scan(&results)

	// Normalization logic for IntensityScore (0-100)
	var maxFreq int64 = 1
	for _, r := range results {
		if r.Frequency > maxFreq {
			maxFreq = r.Frequency
		}
	}

	for _, r := range results {
		intensity := int((float64(r.Frequency) / float64(maxFreq)) * 100)
		response.Zones = append(response.Zones, HeatmapZone{
			ZoneID:         r.ZoneID,
			VisitFrequency: r.Frequency,
			AvgDwell:       r.AvgDwell,
			IntensityScore: intensity,
		})
	}

	c.JSON(http.StatusOK, response)
}
