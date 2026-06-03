package main

import (
	"net/http"
	"store-intelligence/db"
	"store-intelligence/models"

	"github.com/gin-gonic/gin"
)

type Anomaly struct {
	Severity        string `json:"severity"`
	Type            string `json:"type"`
	Message         string `json:"message"`
	SuggestedAction string `json:"suggested_action"`
}

type AnomaliesResponse struct {
	ActiveAnomalies []Anomaly `json:"active_anomalies"`
}

// getAnomalies detects real-time operational anomalies.
func getAnomalies(c *gin.Context) {
	storeID := c.Param("id")
	var response AnomaliesResponse

	// Rule 1: Queue Spike
	// If the latest queue depth is > 5, emit a WARN. If > 10, emit CRITICAL.
	var lastQueueEvent models.Event
	db.DB.Where("store_id = ? AND event_type = ?", storeID, "BILLING_QUEUE_JOIN").
		Order("timestamp desc").
		First(&lastQueueEvent)

	if lastQueueEvent.Metadata.QueueDepth != nil {
		depth := *lastQueueEvent.Metadata.QueueDepth
		if depth > 10 {
			response.ActiveAnomalies = append(response.ActiveAnomalies, Anomaly{
				Severity:        "CRITICAL",
				Type:            "BILLING_QUEUE_SPIKE",
				Message:         "Queue depth is critical (>10)",
				SuggestedAction: "Open a new billing counter immediately.",
			})
		} else if depth > 5 {
			response.ActiveAnomalies = append(response.ActiveAnomalies, Anomaly{
				Severity:        "WARN",
				Type:            "BILLING_QUEUE_SPIKE",
				Message:         "Queue depth is building up (>5)",
				SuggestedAction: "Monitor billing area or call floor staff for assistance.",
			})
		}
	}

	// Rule 2: Dead Zone
	// No visits in a specific zone in the last 30 minutes. (Requires time processing, skipping for brevity in this mock).
	
	// Rule 3: Conversion Drop vs 7-day avg
	// (Requires POS integration, skipping for brevity).

	c.JSON(http.StatusOK, response)
}
