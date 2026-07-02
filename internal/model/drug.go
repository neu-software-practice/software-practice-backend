package model

import "time"

// Drug describes a pharmacy catalog item used to price and fulfill medication cards.
type Drug struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Aliases       []string  `json:"aliases"`
	Spec          string    `json:"spec"`
	DefaultDosage string    `json:"defaultDosage"`
	DefaultDays   int       `json:"defaultDays"`
	UnitPrice     float64   `json:"unitPrice"`
	StockQuantity int       `json:"stockQuantity"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
