package main

import (
	"net/http"
	"store-intelligence/db"
	"store-intelligence/models"

	"github.com/gin-gonic/gin"
)

type FunnelResponse struct {
	EntryCount   int64   `json:"entry_count"`
	ZoneVisit    int64   `json:"zone_visit"`
	BillingQueue int64   `json:"billing_queue"`
	Purchase     int64   `json:"purchase"`
	DropOffPct   float64 `json:"drop_off_pct"`
}

// getFunnel computes the conversion funnel.
// Session is the unit, not raw events. Re-entries must not double-count a visitor.
func getFunnel(c *gin.Context) {
	storeID := c.Param("id")
	var response FunnelResponse

	// 1. Entry Count: Unique visitors who had an ENTRY event
	db.DB.Model(&models.Event{}).
		Where("store_id = ? AND event_type = ? AND is_staff = ?", storeID, "ENTRY", false).
		Distinct("visitor_id").
		Count(&response.EntryCount)

	// 2. Zone Visit: Unique visitors who had a ZONE_ENTER or ZONE_DWELL event
	db.DB.Model(&models.Event{}).
		Where("store_id = ? AND event_type IN ? AND is_staff = ?", storeID, []string{"ZONE_ENTER", "ZONE_DWELL"}, false).
		Distinct("visitor_id").
		Count(&response.ZoneVisit)

	// 3. Billing Queue: Unique visitors who had a BILLING_QUEUE_JOIN event
	db.DB.Model(&models.Event{}).
		Where("store_id = ? AND event_type = ? AND is_staff = ?", storeID, "BILLING_QUEUE_JOIN", false).
		Distinct("visitor_id").
		Count(&response.BillingQueue)

	// 4. Purchase: (Stubbed until POS integration)
	response.Purchase = 0

	// Calculate Drop-off from Entry to Purchase (if Entry > 0)
	if response.EntryCount > 0 {
		response.DropOffPct = 100.0 * float64(response.EntryCount-response.Purchase) / float64(response.EntryCount)
	}

	c.JSON(http.StatusOK, response)
}
