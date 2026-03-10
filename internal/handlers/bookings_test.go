package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"qetero/internal/models"
)

// ---------------------------------------------------------------------------
// testBookingHandler method implementations
// ---------------------------------------------------------------------------

func (h *testBookingHandler) create(c *gin.Context) {
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

func (h *testBookingHandler) get(c *gin.Context) {
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

func (h *testBookingHandler) confirm(c *gin.Context) {
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

func (h *testBookingHandler) cancel(c *gin.Context) {
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

// ---------------------------------------------------------------------------
// Router helper
// ---------------------------------------------------------------------------

func newBookingRouter(h *testBookingHandler, userID uuid.UUID) *gin.Engine {
	r := gin.New()
	protected := r.Group("", injectUserID(userID))
	protected.POST("/v1/bookings", h.create)
	protected.GET("/v1/bookings/:id", h.get)
	protected.PUT("/v1/bookings/:id/confirm", h.confirm)
	protected.PUT("/v1/bookings/:id/cancel", h.cancel)
	return r
}

// futureDate returns a date N days from now formatted as YYYY-MM-DD.
func futureDate(n int) string {
	return time.Now().AddDate(0, 0, n).Format("2006-01-02")
}

func sampleBooking(renterID, ownerID, listingID uuid.UUID) *models.Booking {
	return &models.Booking{
		ID:        uuid.New(),
		ListingID: listingID,
		RenterID:  renterID,
		OwnerID:   ownerID,
		StartDate: time.Now().AddDate(0, 0, 5),
		EndDate:   time.Now().AddDate(0, 0, 8),
		TotalDays: 4,
		TotalPrice: 20000,
		Status:    models.StatusPending,
	}
}

// ---------------------------------------------------------------------------
// Create booking tests
// ---------------------------------------------------------------------------

func TestCreateBooking_HappyPath(t *testing.T) {
	ownerID := uuid.New()
	renterID := uuid.New()
	listing := sampleListing(ownerID)
	listing.MinimumDays = 1

	listingStore := &mockListingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Listing, error) { return listing, nil },
	}
	bookingStore := &mockBookingStore{
		hasConflictFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time) (bool, error) { return false, nil },
		createFn:      func(_ context.Context, _ *models.Booking) error { return nil },
	}
	h := &testBookingHandler{bookings: bookingStore, listings: listingStore}
	r := newBookingRouter(h, renterID)

	body := toJSON(t, map[string]any{
		"listing_id": listing.ID.String(),
		"start_date": futureDate(5),
		"end_date":   futureDate(8),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/bookings", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusCreated)
}

func TestCreateBooking_MissingFields(t *testing.T) {
	h := &testBookingHandler{bookings: &mockBookingStore{}, listings: &mockListingStore{}}
	r := newBookingRouter(h, uuid.New())

	body := toJSON(t, map[string]any{"listing_id": uuid.New().String()}) // missing dates
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/bookings", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

func TestCreateBooking_InvalidListingID(t *testing.T) {
	h := &testBookingHandler{bookings: &mockBookingStore{}, listings: &mockListingStore{}}
	r := newBookingRouter(h, uuid.New())

	body := toJSON(t, map[string]any{
		"listing_id": "not-a-uuid",
		"start_date": futureDate(5),
		"end_date":   futureDate(8),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/bookings", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

func TestCreateBooking_EndBeforeStart(t *testing.T) {
	h := &testBookingHandler{bookings: &mockBookingStore{}, listings: &mockListingStore{}}
	r := newBookingRouter(h, uuid.New())

	body := toJSON(t, map[string]any{
		"listing_id": uuid.New().String(),
		"start_date": futureDate(8),
		"end_date":   futureDate(5), // end before start
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/bookings", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

func TestCreateBooking_PastStartDate(t *testing.T) {
	h := &testBookingHandler{bookings: &mockBookingStore{}, listings: &mockListingStore{}}
	r := newBookingRouter(h, uuid.New())

	body := toJSON(t, map[string]any{
		"listing_id": uuid.New().String(),
		"start_date": "2020-01-01",
		"end_date":   "2020-01-05",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/bookings", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

func TestCreateBooking_ListingNotFound(t *testing.T) {
	listingStore := &mockListingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Listing, error) {
			return nil, errors.New("not found")
		},
	}
	h := &testBookingHandler{bookings: &mockBookingStore{}, listings: listingStore}
	r := newBookingRouter(h, uuid.New())

	body := toJSON(t, map[string]any{
		"listing_id": uuid.New().String(),
		"start_date": futureDate(5),
		"end_date":   futureDate(8),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/bookings", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusNotFound)
}

func TestCreateBooking_ListingUnavailable(t *testing.T) {
	ownerID := uuid.New()
	renterID := uuid.New()
	listing := sampleListing(ownerID)
	listing.IsAvailable = false

	listingStore := &mockListingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Listing, error) { return listing, nil },
	}
	h := &testBookingHandler{bookings: &mockBookingStore{}, listings: listingStore}
	r := newBookingRouter(h, renterID)

	body := toJSON(t, map[string]any{
		"listing_id": listing.ID.String(),
		"start_date": futureDate(5),
		"end_date":   futureDate(8),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/bookings", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusConflict)
}

func TestCreateBooking_CannotBookOwnListing(t *testing.T) {
	ownerID := uuid.New()
	listing := sampleListing(ownerID)

	listingStore := &mockListingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Listing, error) { return listing, nil },
	}
	// renterID == ownerID
	h := &testBookingHandler{bookings: &mockBookingStore{}, listings: listingStore}
	r := newBookingRouter(h, ownerID)

	body := toJSON(t, map[string]any{
		"listing_id": listing.ID.String(),
		"start_date": futureDate(5),
		"end_date":   futureDate(8),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/bookings", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
	resp := decodeBody(t, w)
	if resp["error"] != "cannot book your own listing" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

func TestCreateBooking_BelowMinimumDays(t *testing.T) {
	ownerID := uuid.New()
	renterID := uuid.New()
	listing := sampleListing(ownerID)
	listing.MinimumDays = 5 // require at least 5 days

	listingStore := &mockListingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Listing, error) { return listing, nil },
	}
	h := &testBookingHandler{bookings: &mockBookingStore{}, listings: listingStore}
	r := newBookingRouter(h, renterID)

	body := toJSON(t, map[string]any{
		"listing_id": listing.ID.String(),
		"start_date": futureDate(5),
		"end_date":   futureDate(6), // only 2 days
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/bookings", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

func TestCreateBooking_DateConflict(t *testing.T) {
	ownerID := uuid.New()
	renterID := uuid.New()
	listing := sampleListing(ownerID)
	listing.MinimumDays = 1

	listingStore := &mockListingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Listing, error) { return listing, nil },
	}
	bookingStore := &mockBookingStore{
		hasConflictFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time) (bool, error) {
			return true, nil // conflict!
		},
	}
	h := &testBookingHandler{bookings: bookingStore, listings: listingStore}
	r := newBookingRouter(h, renterID)

	body := toJSON(t, map[string]any{
		"listing_id": listing.ID.String(),
		"start_date": futureDate(5),
		"end_date":   futureDate(8),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/bookings", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusConflict)
}

// ---------------------------------------------------------------------------
// Get booking tests
// ---------------------------------------------------------------------------

func TestGetBooking_HappyPath_AsRenter(t *testing.T) {
	ownerID := uuid.New()
	renterID := uuid.New()
	listingID := uuid.New()
	booking := sampleBooking(renterID, ownerID, listingID)

	bookingStore := &mockBookingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Booking, error) { return booking, nil },
	}
	h := &testBookingHandler{bookings: bookingStore, listings: &mockListingStore{}}
	r := newBookingRouter(h, renterID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/bookings/"+booking.ID.String(), nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
}

func TestGetBooking_HappyPath_AsOwner(t *testing.T) {
	ownerID := uuid.New()
	renterID := uuid.New()
	listingID := uuid.New()
	booking := sampleBooking(renterID, ownerID, listingID)

	bookingStore := &mockBookingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Booking, error) { return booking, nil },
	}
	// authenticated as owner
	h := &testBookingHandler{bookings: bookingStore, listings: &mockListingStore{}}
	r := newBookingRouter(h, ownerID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/bookings/"+booking.ID.String(), nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
}

func TestGetBooking_AccessDenied(t *testing.T) {
	ownerID := uuid.New()
	renterID := uuid.New()
	thirdParty := uuid.New()
	listingID := uuid.New()
	booking := sampleBooking(renterID, ownerID, listingID)

	bookingStore := &mockBookingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Booking, error) { return booking, nil },
	}
	// authenticated as unrelated third party
	h := &testBookingHandler{bookings: bookingStore, listings: &mockListingStore{}}
	r := newBookingRouter(h, thirdParty)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/bookings/"+booking.ID.String(), nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusForbidden)
}

func TestGetBooking_NotFound(t *testing.T) {
	bookingStore := &mockBookingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Booking, error) {
			return nil, errors.New("not found")
		},
	}
	h := &testBookingHandler{bookings: bookingStore, listings: &mockListingStore{}}
	r := newBookingRouter(h, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/bookings/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusNotFound)
}

func TestGetBooking_InvalidID(t *testing.T) {
	h := &testBookingHandler{bookings: &mockBookingStore{}, listings: &mockListingStore{}}
	r := newBookingRouter(h, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/bookings/bad-id", nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// Confirm booking tests
// ---------------------------------------------------------------------------

func TestConfirmBooking_HappyPath(t *testing.T) {
	ownerID := uuid.New()
	renterID := uuid.New()
	listingID := uuid.New()
	booking := sampleBooking(renterID, ownerID, listingID)

	bookingStore := &mockBookingStore{
		getByIDFn:      func(_ context.Context, _ uuid.UUID) (*models.Booking, error) { return booking, nil },
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ models.BookingStatus, _ string) error { return nil },
	}
	// authenticated as owner
	h := &testBookingHandler{bookings: bookingStore, listings: &mockListingStore{}}
	r := newBookingRouter(h, ownerID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/bookings/%s/confirm", booking.ID), nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
	resp := decodeBody(t, w)
	if resp["message"] != "booking confirmed" {
		t.Errorf("unexpected message: %v", resp["message"])
	}
}

func TestConfirmBooking_NotOwner(t *testing.T) {
	ownerID := uuid.New()
	renterID := uuid.New()
	listingID := uuid.New()
	booking := sampleBooking(renterID, ownerID, listingID)

	bookingStore := &mockBookingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Booking, error) { return booking, nil },
	}
	// renter tries to confirm — not allowed
	h := &testBookingHandler{bookings: bookingStore, listings: &mockListingStore{}}
	r := newBookingRouter(h, renterID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/bookings/%s/confirm", booking.ID), nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusForbidden)
}

func TestConfirmBooking_AlreadyConfirmed(t *testing.T) {
	ownerID := uuid.New()
	renterID := uuid.New()
	listingID := uuid.New()
	booking := sampleBooking(renterID, ownerID, listingID)
	booking.Status = models.StatusConfirmed // already confirmed

	bookingStore := &mockBookingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Booking, error) { return booking, nil },
	}
	h := &testBookingHandler{bookings: bookingStore, listings: &mockListingStore{}}
	r := newBookingRouter(h, ownerID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/bookings/%s/confirm", booking.ID), nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

func TestConfirmBooking_NotFound(t *testing.T) {
	bookingStore := &mockBookingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Booking, error) {
			return nil, errors.New("not found")
		},
	}
	h := &testBookingHandler{bookings: bookingStore, listings: &mockListingStore{}}
	r := newBookingRouter(h, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/bookings/%s/confirm", uuid.New()), nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusNotFound)
}

// ---------------------------------------------------------------------------
// Cancel booking tests
// ---------------------------------------------------------------------------

func TestCancelBooking_HappyPath_ByRenter(t *testing.T) {
	ownerID := uuid.New()
	renterID := uuid.New()
	listingID := uuid.New()
	booking := sampleBooking(renterID, ownerID, listingID)

	bookingStore := &mockBookingStore{
		getByIDFn:      func(_ context.Context, _ uuid.UUID) (*models.Booking, error) { return booking, nil },
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ models.BookingStatus, _ string) error { return nil },
	}
	h := &testBookingHandler{bookings: bookingStore, listings: &mockListingStore{}}
	r := newBookingRouter(h, renterID)

	body := toJSON(t, map[string]any{"reason": "changed plans"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/bookings/%s/cancel", booking.ID), body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
	resp := decodeBody(t, w)
	if resp["message"] != "booking cancelled" {
		t.Errorf("unexpected message: %v", resp["message"])
	}
}

func TestCancelBooking_HappyPath_ByOwner(t *testing.T) {
	ownerID := uuid.New()
	renterID := uuid.New()
	listingID := uuid.New()
	booking := sampleBooking(renterID, ownerID, listingID)
	booking.Status = models.StatusConfirmed

	bookingStore := &mockBookingStore{
		getByIDFn:      func(_ context.Context, _ uuid.UUID) (*models.Booking, error) { return booking, nil },
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ models.BookingStatus, _ string) error { return nil },
	}
	// owner cancels
	h := &testBookingHandler{bookings: bookingStore, listings: &mockListingStore{}}
	r := newBookingRouter(h, ownerID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/bookings/%s/cancel", booking.ID), nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
}

func TestCancelBooking_AccessDenied(t *testing.T) {
	ownerID := uuid.New()
	renterID := uuid.New()
	thirdParty := uuid.New()
	listingID := uuid.New()
	booking := sampleBooking(renterID, ownerID, listingID)

	bookingStore := &mockBookingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Booking, error) { return booking, nil },
	}
	h := &testBookingHandler{bookings: bookingStore, listings: &mockListingStore{}}
	r := newBookingRouter(h, thirdParty)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/bookings/%s/cancel", booking.ID), nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusForbidden)
}

func TestCancelBooking_AlreadyCancelled(t *testing.T) {
	ownerID := uuid.New()
	renterID := uuid.New()
	listingID := uuid.New()
	booking := sampleBooking(renterID, ownerID, listingID)
	booking.Status = models.StatusCancelled

	bookingStore := &mockBookingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Booking, error) { return booking, nil },
	}
	h := &testBookingHandler{bookings: bookingStore, listings: &mockListingStore{}}
	r := newBookingRouter(h, renterID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/bookings/%s/cancel", booking.ID), nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

func TestCancelBooking_AlreadyCompleted(t *testing.T) {
	ownerID := uuid.New()
	renterID := uuid.New()
	listingID := uuid.New()
	booking := sampleBooking(renterID, ownerID, listingID)
	booking.Status = models.StatusCompleted

	bookingStore := &mockBookingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Booking, error) { return booking, nil },
	}
	h := &testBookingHandler{bookings: bookingStore, listings: &mockListingStore{}}
	r := newBookingRouter(h, renterID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/bookings/%s/cancel", booking.ID), nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}
