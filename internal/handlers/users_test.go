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
)

// ---------------------------------------------------------------------------
// testUserHandler method implementations
// ---------------------------------------------------------------------------

func (h *testUserHandler) me(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *testUserHandler) update(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.users.Update(c.Request.Context(), userID, req.Name, req.Phone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}

func (h *testUserHandler) myBookings(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	bookings, err := h.bookings.GetByRenter(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch bookings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bookings": bookings})
}

func (h *testUserHandler) incomingBookings(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	bookings, err := h.bookings.GetByOwner(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch bookings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bookings": bookings})
}

// ---------------------------------------------------------------------------
// Router helper
// ---------------------------------------------------------------------------

func newUserRouter(h *testUserHandler, userID uuid.UUID) *gin.Engine {
	r := gin.New()
	protected := r.Group("", injectUserID(userID))
	protected.GET("/v1/users/me", h.me)
	protected.PUT("/v1/users/me", h.update)
	protected.GET("/v1/users/me/bookings", h.myBookings)
	protected.GET("/v1/users/me/listings/bookings", h.incomingBookings)
	return r
}

func sampleUser(id uuid.UUID) *models.User {
	return &models.User{
		ID:    id,
		Name:  "Abebe Girma",
		Phone: "+251911000001",
		Role:  models.RoleRenter,
	}
}

// ---------------------------------------------------------------------------
// Me (GET /v1/users/me) tests
// ---------------------------------------------------------------------------

func TestMe_HappyPath(t *testing.T) {
	userID := uuid.New()
	store := &mockUserStore{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*models.User, error) {
			return sampleUser(id), nil
		},
	}
	h := &testUserHandler{users: store, bookings: &mockBookingStore{}}
	r := newUserRouter(h, userID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
	resp := decodeBody(t, w)
	if resp["name"] != "Abebe Girma" {
		t.Errorf("unexpected name: %v", resp["name"])
	}
}

func TestMe_UserNotFound(t *testing.T) {
	store := &mockUserStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.User, error) {
			return nil, errors.New("no rows in result set")
		},
	}
	h := &testUserHandler{users: store, bookings: &mockBookingStore{}}
	r := newUserRouter(h, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusNotFound)
	resp := decodeBody(t, w)
	if resp["error"] != "user not found" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

// ---------------------------------------------------------------------------
// Update (PUT /v1/users/me) tests
// ---------------------------------------------------------------------------

func TestUpdateUser_HappyPath(t *testing.T) {
	userID := uuid.New()
	store := &mockUserStore{
		updateFn: func(_ context.Context, id uuid.UUID, name, phone string) error { return nil },
	}
	h := &testUserHandler{users: store, bookings: &mockBookingStore{}}
	r := newUserRouter(h, userID)

	body := toJSON(t, map[string]any{
		"name":  "Bekele Molla",
		"phone": "+251922000002",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/users/me", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
	resp := decodeBody(t, w)
	if resp["message"] != "profile updated" {
		t.Errorf("unexpected message: %v", resp["message"])
	}
}

func TestUpdateUser_MissingName(t *testing.T) {
	h := &testUserHandler{users: &mockUserStore{}, bookings: &mockBookingStore{}}
	r := newUserRouter(h, uuid.New())

	body := toJSON(t, map[string]any{"phone": "+251922000002"}) // name is required
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/users/me", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

func TestUpdateUser_StoreError(t *testing.T) {
	store := &mockUserStore{
		updateFn: func(_ context.Context, _ uuid.UUID, _, _ string) error {
			return errors.New("db connection lost")
		},
	}
	h := &testUserHandler{users: store, bookings: &mockBookingStore{}}
	r := newUserRouter(h, uuid.New())

	body := toJSON(t, map[string]any{"name": "Abebe", "phone": "+251911000001"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/users/me", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusInternalServerError)
}

// ---------------------------------------------------------------------------
// MyBookings (GET /v1/users/me/bookings) tests
// ---------------------------------------------------------------------------

func TestMyBookings_HappyPath(t *testing.T) {
	userID := uuid.New()
	ownerID := uuid.New()
	listingID := uuid.New()
	booking := sampleBooking(userID, ownerID, listingID)

	bookingStore := &mockBookingStore{
		getByRenterFn: func(_ context.Context, _ uuid.UUID) ([]models.Booking, error) {
			return []models.Booking{*booking}, nil
		},
	}
	h := &testUserHandler{users: &mockUserStore{}, bookings: bookingStore}
	r := newUserRouter(h, userID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/users/me/bookings", nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
	resp := decodeBody(t, w)
	bookings, ok := resp["bookings"].([]any)
	if !ok || len(bookings) != 1 {
		t.Errorf("expected 1 booking in response, got %v", resp["bookings"])
	}
}

func TestMyBookings_EmptyResult(t *testing.T) {
	bookingStore := &mockBookingStore{
		getByRenterFn: func(_ context.Context, _ uuid.UUID) ([]models.Booking, error) {
			return nil, nil
		},
	}
	h := &testUserHandler{users: &mockUserStore{}, bookings: bookingStore}
	r := newUserRouter(h, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/users/me/bookings", nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
}

func TestMyBookings_StoreError(t *testing.T) {
	bookingStore := &mockBookingStore{
		getByRenterFn: func(_ context.Context, _ uuid.UUID) ([]models.Booking, error) {
			return nil, errors.New("db error")
		},
	}
	h := &testUserHandler{users: &mockUserStore{}, bookings: bookingStore}
	r := newUserRouter(h, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/users/me/bookings", nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusInternalServerError)
}

// ---------------------------------------------------------------------------
// IncomingBookings (GET /v1/users/me/listings/bookings) tests
// ---------------------------------------------------------------------------

func TestIncomingBookings_HappyPath(t *testing.T) {
	ownerID := uuid.New()
	renterID := uuid.New()
	listingID := uuid.New()
	booking := sampleBooking(renterID, ownerID, listingID)

	bookingStore := &mockBookingStore{
		getByOwnerFn: func(_ context.Context, _ uuid.UUID) ([]models.Booking, error) {
			return []models.Booking{*booking}, nil
		},
	}
	h := &testUserHandler{users: &mockUserStore{}, bookings: bookingStore}
	r := newUserRouter(h, ownerID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/users/me/listings/bookings", nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
	resp := decodeBody(t, w)
	bookings, ok := resp["bookings"].([]any)
	if !ok || len(bookings) != 1 {
		t.Errorf("expected 1 incoming booking in response, got %v", resp["bookings"])
	}
}

func TestIncomingBookings_StoreError(t *testing.T) {
	bookingStore := &mockBookingStore{
		getByOwnerFn: func(_ context.Context, _ uuid.UUID) ([]models.Booking, error) {
			return nil, errors.New("db error")
		},
	}
	h := &testUserHandler{users: &mockUserStore{}, bookings: bookingStore}
	r := newUserRouter(h, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/users/me/listings/bookings", nil)
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusInternalServerError)
}
