package db

import (
	"log"

	"store-intelligence/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB initializes the SQLite database and runs auto-migrations.
func InitDB() {
	var err error

	// Open a connection to a local SQLite file named "store_intelligence.db"
	// We also configure the logger to show us the SQL queries in the terminal (great for learning!)
	DB, err = gorm.Open(sqlite.Open("store_intelligence.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	log.Println("Database connection established.")

	// AutoMigrate creates the table based on your Event struct.
	// It is idempotent: it won't drop existing columns or data.
	err = DB.AutoMigrate(&models.Event{}, &models.PosTransaction{})
	if err != nil {
		log.Fatalf("Failed to run auto-migrations: %v", err)
	}

	log.Println("Database migrated successfully.")
}
