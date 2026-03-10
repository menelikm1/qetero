package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"qetero/internal/models"
	"qetero/internal/repository"
)

// ---------------------------------------------------------------------------
// testListingHandler method implementations
// ---------------------------------------------------------------------------

func (h *testListingHandler) create(c *gin.Context) {
	ownerID := c.MustGet("user_id").(uuid.UUID)

	var req createListingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.MinimumDays < 1 {
		req.MinimumDays = 1
	}

	listing := &models.Listing{
		ID:          uuid.New(),
		OwnerID:     ownerID,
		Title:       req.Title,
		Category:    req.Category,
		Description: req.Description,
		Location:    req.Location,
		PricePerDay: req.PricePerDay,
		MinimumDays: req.MinimumDays,
		Images:      []string{},
	}

	if err := h.listings.Create(c.Request.Context(), listing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create listing"})
		return
	}

	c.JSON(http.StatusCreated, listing)
}

func (h *testListingHandler) get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid listing id"})
		return
	}

	listing, err := h.listings.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	c.JSON(http.StatusOK, listing)
}

func (h *testListingHandler) browse(c *gin.Context) {
	f := repository.ListingFilter{
		Category: c.Query("category"),
		Location: c.Query("location"),
	}
	f.Limit = 20

	listings, err := h.listings.Browse(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch listings"})
		return
	}
	if listings == nil {
		listings = []models.Listing{}
	}

	c.JSON(http.StatusOK, gin.H{"listings": listings, "page": f.Page, "limit": f.Limit})
}

func (h *testListingHandler) update(c *gin.Context) {
	ownerID := c.MustGet("user_id").(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid listing id"})
		return
	}

	listing, err := h.listings.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}
	if listing.OwnerID != ownerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your listing"})
		return
	}

	var req createListingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	listing.Title = req.Title
	listing.Category = req.Category
	listing.Description = req.Description
	listing.Location = req.Location
	listing.PricePerDay = req.PricePerDay
	listing.MinimumDays = req.MinimumDays

	if err := h.listings.Update(c.Request.Context(), listing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update listing"})
		return
	}

	c.JSON(http.StatusOK, listing)
}

func (h *testListingHandler) delete(c *gin.Context) {
	ownerID := c.MustGet("user_id").(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid listing id"})
		return
	}

	if err := h.listings.Delete(c.Request.Context(), id, ownerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete listing"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "listing deleted"})
}

func (h *testListingHandler) availability(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid listing id"})
		return
	}

	dates, err := h.bookings.GetBookedDates(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch availability"})
		return
	}
	if dates == nil {
		dates = []repository.DateRange{}
	}

	c.JSON(http.StatusOK, gin.H{"booked_ranges": dates})
}

// ---------------------------------------------------------------------------
// Router helpers
// ---------------------------------------------------------------------------

// injectUserID is a middleware that sets user_id in the gin context.
func injectUserID(id uuid.UUID) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", id)
		c.Next()
	}
}

func newListingRouter(h *testListingHandler, ownerID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.GET("/v1/listings", h.browse)
	r.GET("/v1/listings/:id", h.get)
	r.GET("/v1/listings/:id/availability", h.availability)
	auth := r.Group("/v1/listings", injectUserID(ownerID))
	auth.POST("", h.create)
	auth.PUT("/:id", h.update)
	auth.DELETE("/:id", h.delete)
	return r
}

func sampleListing(ownerID uuid.UUID) *models.Listing {
	return &models.Listing{
		ID:          uuid.New(),
		OwnerID:     ownerID,
		Title:       "CAT 320 Excavator",
		Category:    models.CategoryExcavator,
		Description: "Well-maintained excavator available for rent",
		Location:    "Addis Ababa",
		PricePerDay: 5000,
		MinimumDays: 3,
		IsAvailable: true,
		Status:      models.ListingStatusActive,
		Images:      []string{},
	}
}

// ---------------------------------------------------------------------------
// Get listing tests
// ---------------------------------------------------------------------------

func TestGetListing_HappyPath(t *testing.T) {
	ownerID := uuid.New()
	listing := sampleListing(ownerID)

	store := &mockListingStore{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*models.Listing, error) {
			if id == listing.ID {
				return listing, nil
			}
			return nil, errors.New("not found")
		},
	}
	h := &testListingHandler{listings: store, bookings: &mockBookingStore{}}
	r := newListingRouter(h, ownerID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/listings/"+listing.ID.String(), nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
}

func TestGetListing_InvalidID(t *testing.T) {
	h := &testListingHandler{listings: &mockListingStore{}, bookings: &mockBookingStore{}}
	r := newListingRouter(h, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/listings/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

func TestGetListing_NotFound(t *testing.T) {
	store := &mockListingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Listing, error) {
			return nil, errors.New("no rows in result set")
		},
	}
	h := &testListingHandler{listings: store, bookings: &mockBookingStore{}}
	r := newListingRouter(h, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/listings/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusNotFound)
}

// ---------------------------------------------------------------------------
// Browse listings tests
// ---------------------------------------------------------------------------

func TestBrowseListings_HappyPath(t *testing.T) {
	ownerID := uuid.New()
	store := &mockListingStore{
		browseFn: func(_ context.Context, _ repository.ListingFilter) ([]models.Listing, error) {
			return []models.Listing{*sampleListing(ownerID)}, nil
		},
	}
	h := &testListingHandler{listings: store, bookings: &mockBookingStore{}}
	r := newListingRouter(h, ownerID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/listings", nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
	resp := decodeBody(t, w)
	listings, ok := resp["listings"].([]any)
	if !ok || len(listings) != 1 {
		t.Errorf("expected 1 listing in response, got %v", resp["listings"])
	}
}

func TestBrowseListings_EmptyResult(t *testing.T) {
	store := &mockListingStore{
		browseFn: func(_ context.Context, _ repository.ListingFilter) ([]models.Listing, error) {
			return nil, nil // repo returns nil slice
		},
	}
	h := &testListingHandler{listings: store, bookings: &mockBookingStore{}}
	r := newListingRouter(h, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/listings", nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
	resp := decodeBody(t, w)
	listings, ok := resp["listings"].([]any)
	if !ok {
		t.Fatalf("listings key should be an array, got %T", resp["listings"])
	}
	if len(listings) != 0 {
		t.Errorf("expected empty listings array, got %d items", len(listings))
	}
}

func TestBrowseListings_StoreError(t *testing.T) {
	store := &mockListingStore{
		browseFn: func(_ context.Context, _ repository.ListingFilter) ([]models.Listing, error) {
			return nil, errors.New("db error")
		},
	}
	h := &testListingHandler{listings: store, bookings: &mockBookingStore{}}
	r := newListingRouter(h, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/listings", nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusInternalServerError)
}

// ---------------------------------------------------------------------------
// Create listing tests
// ---------------------------------------------------------------------------

func TestCreateListing_HappyPath(t *testing.T) {
	ownerID := uuid.New()
	store := &mockListingStore{
		createFn: func(_ context.Context, _ *models.Listing) error { return nil },
	}
	h := &testListingHandler{listings: store, bookings: &mockBookingStore{}}
	r := newListingRouter(h, ownerID)

	body := toJSON(t, map[string]any{
		"title":        "CAT 320 Excavator",
		"category":     "excavator",
		"description":  "Well-maintained",
		"location":     "Addis Ababa",
		"price_per_day": 5000,
		"minimum_days": 3,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/listings", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusCreated)
}

func TestCreateListing_MissingTitle(t *testing.T) {
	ownerID := uuid.New()
	h := &testListingHandler{listings: &mockListingStore{}, bookings: &mockBookingStore{}}
	r := newListingRouter(h, ownerID)

	body := toJSON(t, map[string]any{
		"category":     "excavator",
		"description":  "desc",
		"location":     "Addis",
		"price_per_day": 5000,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/listings", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

func TestCreateListing_InvalidCategory(t *testing.T) {
	ownerID := uuid.New()
	h := &testListingHandler{listings: &mockListingStore{}, bookings: &mockBookingStore{}}
	r := newListingRouter(h, ownerID)

	body := toJSON(t, map[string]any{
		"title":        "My Machine",
		"category":     "helicopter", // invalid
		"description":  "desc",
		"location":     "Addis",
		"price_per_day": 5000,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/listings", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

func TestCreateListing_ZeroPrice(t *testing.T) {
	ownerID := uuid.New()
	h := &testListingHandler{listings: &mockListingStore{}, bookings: &mockBookingStore{}}
	r := newListingRouter(h, ownerID)

	body := toJSON(t, map[string]any{
		"title":        "My Machine",
		"category":     "crane",
		"description":  "desc",
		"location":     "Addis",
		"price_per_day": 0, // must be > 0
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/listings", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// Update listing tests
// ---------------------------------------------------------------------------

func TestUpdateListing_HappyPath(t *testing.T) {
	ownerID := uuid.New()
	listing := sampleListing(ownerID)

	store := &mockListingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Listing, error) { return listing, nil },
		updateFn:  func(_ context.Context, _ *models.Listing) error { return nil },
	}
	h := &testListingHandler{listings: store, bookings: &mockBookingStore{}}
	r := newListingRouter(h, ownerID)

	body := toJSON(t, map[string]any{
		"title":        "Updated Excavator",
		"category":     "excavator",
		"description":  "Updated description",
		"location":     "Dire Dawa",
		"price_per_day": 6000,
		"minimum_days": 2,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/listings/"+listing.ID.String(), body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
}

func TestUpdateListing_NotOwner(t *testing.T) {
	ownerID := uuid.New()
	otherOwnerID := uuid.New()
	listing := sampleListing(otherOwnerID) // owned by someone else

	store := &mockListingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Listing, error) { return listing, nil },
	}
	// Request authenticated as ownerID, but listing belongs to otherOwnerID
	h := &testListingHandler{listings: store, bookings: &mockBookingStore{}}
	r := newListingRouter(h, ownerID)

	body := toJSON(t, map[string]any{
		"title":        "Hijack",
		"category":     "excavator",
		"description":  "desc",
		"location":     "Addis",
		"price_per_day": 1000,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/listings/"+listing.ID.String(), body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusForbidden)
}

func TestUpdateListing_NotFound(t *testing.T) {
	ownerID := uuid.New()
	store := &mockListingStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.Listing, error) {
			return nil, errors.New("not found")
		},
	}
	h := &testListingHandler{listings: store, bookings: &mockBookingStore{}}
	r := newListingRouter(h, ownerID)

	body := toJSON(t, map[string]any{
		"title":        "X",
		"category":     "crane",
		"description":  "d",
		"location":     "Addis",
		"price_per_day": 1000,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/listings/"+uuid.New().String(), body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusNotFound)
}

// ---------------------------------------------------------------------------
// Delete listing tests
// ---------------------------------------------------------------------------

func TestDeleteListing_HappyPath(t *testing.T) {
	ownerID := uuid.New()
	listing := sampleListing(ownerID)

	store := &mockListingStore{
		deleteFn: func(_ context.Context, id, oid uuid.UUID) error { return nil },
	}
	h := &testListingHandler{listings: store, bookings: &mockBookingStore{}}
	r := newListingRouter(h, ownerID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/v1/listings/"+listing.ID.String(), nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
	resp := decodeBody(t, w)
	if resp["message"] != "listing deleted" {
		t.Errorf("unexpected message: %v", resp["message"])
	}
}

func TestDeleteListing_InvalidID(t *testing.T) {
	h := &testListingHandler{listings: &mockListingStore{}, bookings: &mockBookingStore{}}
	r := newListingRouter(h, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/v1/listings/bad-uuid", nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// Availability tests
// ---------------------------------------------------------------------------

func TestAvailability_HappyPath(t *testing.T) {
	listingID := uuid.New()
	bookingStore := &mockBookingStore{
		getBookedDatesFn: func(_ context.Context, id uuid.UUID) ([]repository.DateRange, error) {
			return []repository.DateRange{
				{Start: "2026-04-01", End: "2026-04-05"},
			}, nil
		},
	}
	h := &testListingHandler{listings: &mockListingStore{}, bookings: bookingStore}
	r := newListingRouter(h, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/listings/"+listingID.String()+"/availability", nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
	resp := decodeBody(t, w)
	ranges, ok := resp["booked_ranges"].([]any)
	if !ok || len(ranges) != 1 {
		t.Errorf("expected 1 booked range, got %v", resp["booked_ranges"])
	}
}

func TestAvailability_InvalidID(t *testing.T) {
	h := &testListingHandler{listings: &mockListingStore{}, bookings: &mockBookingStore{}}
	r := newListingRouter(h, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/listings/bad-id/availability", nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}
