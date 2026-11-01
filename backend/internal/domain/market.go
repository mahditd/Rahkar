package domain

import "time"

// MarketPrice is a national asking-price estimate, never an individual listing.
type MarketPrice struct {
	Name       string `json:"name"`
	ModelYear  int    `json:"model_year"`
	PriceToman int64  `json:"price_toman"`
	MileageKM  int    `json:"mileage_km"`
	DataPoints int    `json:"data_points"`
	SourceURL  string `json:"source_url"`
}

type MarketSnapshot struct {
	ComputedAt time.Time     `json:"computed_at"`
	Items      []MarketPrice `json:"items"`
}
