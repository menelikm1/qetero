package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"qetero/internal/models"
	"qetero/internal/repository"
)

type BookingHandler struct {
	bookings *repository.BookingRepo
	listings *repository.ListingRepo
}

func NewBookingHandler(db *pgxpool.Pool) *BookingHandler {
	return &BookingHandler{
		bookings: repository.NewBookingRepo(db),
		listings: repository.NewListingRepo(db),
	}
}

type createBookingRequest struct {
	ListingID string `json:"listing_id" binding:"required"`
	StartDate string `json:"start_date" binding:"required"` // YYYY-MM-DD
	EndDate   string `json:"end_date"   binding:"required"` // YYYY-MM-DD
}

func (h *BookingHandler) Create(c *gin.Context) {
	renterID := c.MustGet("user_id").(uuid.UUID)

	var req createBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	listingID, err := uuid.Parse(req.ListingID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid listing_id"})
		return
	}

	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date, expected YYYY-MM-DD"})
		return
	}
	end, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date, expected YYYY-MM-DD"})
		return
	}

	if !end.After(start) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_date must be after start_date"})
		return
	}
	if start.Before(time.Now().Truncate(24 * time.Hour)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date cannot be in the past"})
		return
	}

	listing, err := h.listings.GetByID(c.Request.Context(), listingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}
	if !listing.IsAvailable {
		c.JSON(http.StatusConflict, gin.H{"error": "listing is not available"})
		return
	}
	if listing.OwnerID == renterID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot book your own listing"})
		return
	}

	days := int(end.Sub(start).Hours()/24) + 1
	if days < listing.MinimumDays {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":        "does not meet minimum rental days",
			"minimum_days": listing.MinimumDays,
		})
		return
	}

	conflict, err := h.bookings.HasConflict(c.Request.Context(), listingID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check availability"})
		return
	}
	if conflict {
		c.JSON(http.StatusConflict, gin.H{"error": "listing is already booked for those dates"})
		return
	}

	booking := &models.Booking{
		ID:         uuid.New(),
		ListingID:  listingID,
		RenterID:   renterID,
		OwnerID:    listing.OwnerID,
		StartDate:  start,
		EndDate:    end,
		TotalDays:  days,
		TotalPrice: float64(days) * listing.PricePerDay,
		Status:     models.StatusPending,
	}

	if err := h.bookings.Create(c.Request.Context(), booking); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create booking"})
		return
	}

	c.JSON(http.StatusCreated, booking)
}

func (h *BookingHandler) Get(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}

	booking, err := h.bookings.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
		return
	}

	if booking.RenterID != userID && booking.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, booking)
}

func (h *BookingHandler) Confirm(c *gin.Context) {
	ownerID := c.MustGet("user_id").(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}

	booking, err := h.bookings.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
		return
	}
	if booking.OwnerID != ownerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the owner can confirm bookings"})
		return
	}
	if booking.Status != models.StatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only pending bookings can be confirmed"})
		return
	}

	if err := h.bookings.UpdateStatus(c.Request.Context(), id, models.StatusConfirmed, ""); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm booking"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "booking confirmed"})
}

type cancelRequest struct {
	Reason string `json:"reason"`
}

func (h *BookingHandler) Cancel(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}

	booking, err := h.bookings.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
		return
	}
	if booking.RenterID != userID && booking.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	if booking.Status == models.StatusCompleted || booking.Status == models.StatusCancelled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "booking cannot be cancelled in its current state"})
		return
	}

	var req cancelRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.bookings.UpdateStatus(c.Request.Context(), id, models.StatusCancelled, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel booking"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "booking cancelled"})
}
