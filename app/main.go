package main

import (
	"log"
	"net/http"
	"time"

	"store-intelligence/db"     // Update this to your module path
	"store-intelligence/models" // Update this to your module path

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
)

func main() {
	// 1. Initialize the Database
	db.InitDB()

	// 2. Initialize the Gin Router
	router := gin.Default()

	// Enable CORS for frontend dashboard
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 3. Define the Endpoints
	router.POST("/events/ingest", ingestEvents)
	router.POST("/pos/ingest", ingestPosTransactions)
	router.GET("/stores/:id/metrics", getMetrics)
	router.GET("/stores/:id/funnel", getFunnel)
	router.GET("/stores/:id/heatmap", getHeatmap)
	router.GET("/stores/:id/anomalies", getAnomalies)
	router.GET("/health", HealthCheck)

	// 4. Start the server
	log.Println("Starting Intelligence API on :8080...")
	router.Run(":8080")
}

// ingestEvents handles the batch ingestion of detection events.
func ingestEvents(c *gin.Context) {
	// We expect an array of Events
	var events []models.Event

	// ShouldBindJSON automatically parses the raw JSON body into our Go structs.
	// If the JSON is malformed (e.g., passing a string where an int is expected), it fails here.
	if err := c.ShouldBindJSON(&events); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid JSON payload",
			"details": err.Error(),
		})
		return
	}

	// Basic Validation: Ensure the batch isn't too large
	if len(events) > 500 {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": "Batch size exceeds the maximum limit of 500 events",
		})
		return
	}

	// GORM's CreateInBatches is highly optimized.
	// The "clause.OnConflict" logic handles idempotency: if the EventID already exists, it ignores it instead of crashing.

	result := db.DB.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(&events, 100)
	if result.Error != nil {
		// Log the internal error for the backend engineer
		log.Printf("Database insertion error: %v", result.Error)

		// Return a safe 500 error to the client
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to store events",
		})
		return
	}

	// Success Response
	c.JSON(http.StatusOK, gin.H{
		"message":         "Batch ingested successfully",
		"events_stored":   result.RowsAffected,
		"events_received": len(events),
	})
}

func ingestPosTransactions(c *gin.Context) {
	var txns []models.PosTransaction
	if err := c.ShouldBindJSON(&txns); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, txn := range txns {
		// Insert into DB. We use ON CONFLICT DO NOTHING using gorm clauses to ensure idempotency.
		if err := db.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&txn).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save POS transaction"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "POS transactions successfully ingested"})
}

// HealthCheck handles the basic liveness ping.
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}
