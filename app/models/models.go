package models

import "time"

type Event struct {
	EventID    string    `json:"event_id" gorm:"primaryKey"`
	StoreID    string    `json:"store_id" gorm:"index"` // Useful for filtering by store
	CameraID   string    `json:"camera_id"`
	VisitorID  string    `json:"visitor_id" gorm:"index"` // Useful for tracking unique visitors
	EventType  string    `json:"event_type" gorm:"index"`
	Timestamp  time.Time `json:"timestamp"`
	ZoneID     *string   `json:"zone_id"`
	DwellMs    int       `json:"dwell_ms"`
	IsStaff    bool      `json:"is_staff"`
	Confidence float64   `json:"confidence"`
	Metadata   Metadata  `json:"metadata" gorm:"embedded"`
}

type Metadata struct {
	QueueDepth *int    `json:"queue_depth"`
	SkuZone    *string `json:"sku_zone"`
	SessionSeq int     `json:"session_seq"`
}

type PosTransaction struct {
	TransactionID string    `json:"transaction_id" gorm:"primaryKey"`
	StoreID       string    `json:"store_id" gorm:"index"`
	Timestamp     time.Time `json:"timestamp"`
	BasketValue   float64   `json:"basket_value_inr"`
}
