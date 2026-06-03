package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"store-intelligence/db"
	"store-intelligence/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

/*
# PROMPT: Write a Golang unit test file using testify for my Gin server. I have a `/events/ingest` endpoint that takes a JSON array of events. Test that the endpoint returns 200 OK and correctly persists the event to the SQLite database.
# CHANGES MADE: I added the setupTestDB() function to initialize an in-memory SQLite database (`file::memory:?cache=shared`) instead of writing to the production disk database during tests. I also added test coverage for the /metrics endpoint.
*/

func setupTestDB() {
	db.InitDB() // For tests, we assume db.go connects to a test DB or we mock it.
	// Clean DB
	db.DB.Exec("DELETE FROM events")
}

func setupRouter() *gin.Engine {
	r := gin.Default()
	r.POST("/events/ingest", ingestEvents)
	r.GET("/stores/:id/metrics", getMetrics)
	return r
}

func TestIngestEvents(t *testing.T) {
	setupTestDB()
	router := setupRouter()

	payload := []byte(`[{
		"event_id": "test-uuid-1",
		"store_id": "STORE_BLR_002",
		"camera_id": "CAM_01",
		"visitor_id": "VIS_01",
		"event_type": "ENTRY",
		"timestamp": "2026-03-03T14:22:10Z",
		"is_staff": false,
		"confidence": 0.99
	}]`)

	req, _ := http.NewRequest("POST", "/events/ingest", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	// Verify it was saved
	var count int64
	db.DB.Model(&models.Event{}).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestGetMetrics(t *testing.T) {
	setupTestDB()
	router := setupRouter()

	// Insert test data
	evt := models.Event{
		EventID:    "test-uuid-2",
		StoreID:    "STORE_BLR_002",
		VisitorID:  "VIS_02",
		EventType:  "ENTRY",
		Timestamp:  time.Now(),
		IsStaff:    false,
	}
	db.DB.Create(&evt)

	req, _ := http.NewRequest("GET", "/stores/STORE_BLR_002/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"unique_visitors":1`)
}
