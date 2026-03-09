package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ListingStatus string

const (
	ListingStatusPendingReview ListingStatus = "pending_review"
	ListingStatusActive        ListingStatus = "active"
	ListingStatusRejected      ListingStatus = "rejected"
)

type ListingCategory string

const (
	CategoryExcavator    ListingCategory = "excavator"
	CategoryCrane        ListingCategory = "crane"
	CategoryScaffold     ListingCategory = "scaffold"
	CategoryCompactor    ListingCategory = "compactor"
	CategoryLoader       ListingCategory = "loader"
	CategoryForklift     ListingCategory = "forklift"
	CategoryGenerator    ListingCategory = "generator"
	CategoryWaterTruck   ListingCategory = "water_truck"
	CategoryConcreteMixer ListingCategory = "concrete_mixer"
	CategoryDumpTruck    ListingCategory = "dump_truck"
	CategoryDozer        ListingCategory = "dozer"
	CategoryRoller       ListingCategory = "roller"
	CategoryOther        ListingCategory = "other"
)

type Listing struct {
	ID          uuid.UUID       `json:"id"`
	OwnerID     uuid.UUID       `json:"owner_id"`
	Title       string          `json:"title"`
	Category    ListingCategory `json:"category"`
	Description string          `json:"description"`
	Location    string          `json:"location"`
	PricePerDay float64         `json:"price_per_day"`
	MinimumDays int             `json:"minimum_days"`
	Images      []string        `json:"images"`
	// Specs is stored as JSONB — flexible key/value pairs (e.g. weight, capacity, fuel_type)
	Specs       json.RawMessage `json:"specs"`
	IsAvailable bool          `json:"is_available"`
	Status      ListingStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}
