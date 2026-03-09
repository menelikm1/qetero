package models

import (
	"time"

	"github.com/google/uuid"
)

type DepositStatus string

const (
	DepositStatusPending  DepositStatus = "pending"
	DepositStatusVerified DepositStatus = "verified"
	DepositStatusRejected DepositStatus = "rejected"
)

type BookingStatus string

const (
	StatusPending   BookingStatus = "pending"
	StatusConfirmed BookingStatus = "confirmed"
	StatusActive    BookingStatus = "active"
	StatusCompleted BookingStatus = "completed"
	StatusCancelled BookingStatus = "cancelled"
)

type Booking struct {
	ID                 uuid.UUID     `json:"id"`
	ListingID          uuid.UUID     `json:"listing_id"`
	RenterID           uuid.UUID     `json:"renter_id"`
	OwnerID            uuid.UUID     `json:"owner_id"`
	StartDate          time.Time     `json:"start_date"`
	EndDate            time.Time     `json:"end_date"`
	TotalDays          int           `json:"total_days"`
	TotalPrice         float64       `json:"total_price"`
	Status             BookingStatus `json:"status"`
	DepositRef         string        `json:"deposit_ref,omitempty"`
	DepositStatus      DepositStatus `json:"deposit_status"`
	CancellationReason string        `json:"cancellation_reason,omitempty"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}
