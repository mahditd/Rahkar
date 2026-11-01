package domain

import "time"

type SourceStatus struct {
	SourceName    string    `json:"source_name"`
	Status        string    `json:"status"`
	LastRunAt     time.Time `json:"last_run_at"`
	ImportedCount int       `json:"imported_count"`
}

type SavedAlert struct {
	Token         string    `json:"token"`
	Query         string    `json:"query"`
	City          string    `json:"city"`
	Make          string    `json:"make"`
	MaxPriceToman int64     `json:"max_price_toman"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}
