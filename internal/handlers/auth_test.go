package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"qetero/internal/auth"
	"qetero/internal/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// testAuthHandler method implementations
// ---------------------------------------------------------------------------

func (h *testAuthHandler) register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 4) // cost 4 for speed in tests
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process password"})
		return
	}

	user := &models.User{
		ID:           uuid.New(),
		Name:         req.Name,
		Phone:        req.Phone,
		PasswordHash: string(hash),
		Role:         req.Role,
	}
	if req.Email != "" {
		user.Email = &req.Email
	}

	if err := h.users.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "registration failed", "detail": err.Error()})
		return
	}

	token, err := auth.GenerateToken(user.ID, string(user.Role), h.cfg.jwtSecret, h.cfg.expiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": token, "user": user})
}

func (h *testAuthHandler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Phone == "" && req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone or email required"})
		return
	}

	var user *models.User
	var err error
	if req.Phone != "" {
		user, err = h.users.GetByPhoneForAuth(c.Request.Context(), req.Phone)
	} else {
		user, err = h.users.GetByEmail(c.Request.Context(), req.Email)
	}
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := auth.GenerateToken(user.ID, string(user.Role), h.cfg.jwtSecret, h.cfg.expiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestCfg() testCfg {
	return testCfg{jwtSecret: "test-secret-key", expiry: time.Hour}
}

func newAuthRouter(h *testAuthHandler) *gin.Engine {
	r := gin.New()
	r.POST("/v1/auth/register", h.register)
	r.POST("/v1/auth/login", h.login)
	return r
}

func toJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("toJSON: %v", err)
	}
	return bytes.NewBuffer(b)
}

func assertStatus(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

// validPasswordHash returns a bcrypt hash of "password123" at cost 4 (fast for tests).
func validPasswordHash(t *testing.T) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte("password123"), 4)
	if err != nil {
		t.Fatal(err)
	}
	return string(h)
}

// ---------------------------------------------------------------------------
// Register tests
// ---------------------------------------------------------------------------

func TestRegister_HappyPath(t *testing.T) {
	store := &mockUserStore{
		createFn: func(_ context.Context, u *models.User) error { return nil },
	}
	h := &testAuthHandler{users: store, cfg: newTestCfg()}
	r := newAuthRouter(h)

	body := toJSON(t, map[string]any{
		"name":     "Abebe Girma",
		"phone":    "+251911000001",
		"password": "securepass",
		"role":     "renter",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusCreated)
	resp := decodeBody(t, w)
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("expected non-empty token in response")
	}
	if resp["user"] == nil {
		t.Error("expected user object in response")
	}
}

func TestRegister_MissingRequiredFields(t *testing.T) {
	store := &mockUserStore{}
	h := &testAuthHandler{users: store, cfg: newTestCfg()}
	r := newAuthRouter(h)

	// Missing "name"
	body := toJSON(t, map[string]any{
		"phone":    "+251911000001",
		"password": "securepass",
		"role":     "renter",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

func TestRegister_InvalidRole(t *testing.T) {
	store := &mockUserStore{}
	h := &testAuthHandler{users: store, cfg: newTestCfg()}
	r := newAuthRouter(h)

	body := toJSON(t, map[string]any{
		"name":     "Abebe",
		"phone":    "+251911000001",
		"password": "securepass",
		"role":     "admin", // not a valid enum value
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

func TestRegister_PasswordTooShort(t *testing.T) {
	store := &mockUserStore{}
	h := &testAuthHandler{users: store, cfg: newTestCfg()}
	r := newAuthRouter(h)

	body := toJSON(t, map[string]any{
		"name":     "Abebe",
		"phone":    "+251911000001",
		"password": "short", // < 8 chars
		"role":     "renter",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

func TestRegister_DuplicatePhone(t *testing.T) {
	store := &mockUserStore{
		createFn: func(_ context.Context, _ *models.User) error {
			return errors.New("duplicate key value violates unique constraint")
		},
	}
	h := &testAuthHandler{users: store, cfg: newTestCfg()}
	r := newAuthRouter(h)

	body := toJSON(t, map[string]any{
		"name":     "Abebe",
		"phone":    "+251911000001",
		"password": "securepass",
		"role":     "renter",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusConflict)
	resp := decodeBody(t, w)
	if resp["error"] == nil {
		t.Error("expected error in response body")
	}
}

func TestRegister_WithOptionalEmail(t *testing.T) {
	var capturedUser *models.User
	store := &mockUserStore{
		createFn: func(_ context.Context, u *models.User) error {
			capturedUser = u
			return nil
		},
	}
	h := &testAuthHandler{users: store, cfg: newTestCfg()}
	r := newAuthRouter(h)

	body := toJSON(t, map[string]any{
		"name":     "Abebe",
		"phone":    "+251911000001",
		"email":    "abebe@example.com",
		"password": "securepass",
		"role":     "both",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusCreated)
	if capturedUser == nil || capturedUser.Email == nil || *capturedUser.Email != "abebe@example.com" {
		t.Errorf("expected email to be set on user, got %v", capturedUser)
	}
}

// ---------------------------------------------------------------------------
// Login tests
// ---------------------------------------------------------------------------

func TestLogin_HappyPath_ByPhone(t *testing.T) {
	hash := validPasswordHash(t)
	store := &mockUserStore{
		getByPhoneAuthFn: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{
				ID:           uuid.New(),
				Name:         "Abebe",
				Phone:        "+251911000001",
				PasswordHash: hash,
				Role:         models.RoleRenter,
			}, nil
		},
	}
	h := &testAuthHandler{users: store, cfg: newTestCfg()}
	r := newAuthRouter(h)

	body := toJSON(t, map[string]any{
		"phone":    "+251911000001",
		"password": "password123",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
	resp := decodeBody(t, w)
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("expected non-empty token")
	}
}

func TestLogin_HappyPath_ByEmail(t *testing.T) {
	hash := validPasswordHash(t)
	email := "abebe@example.com"
	store := &mockUserStore{
		getByEmailFn: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{
				ID:           uuid.New(),
				Name:         "Abebe",
				Phone:        "+251911000001",
				Email:        &email,
				PasswordHash: hash,
				Role:         models.RoleOwner,
			}, nil
		},
	}
	h := &testAuthHandler{users: store, cfg: newTestCfg()}
	r := newAuthRouter(h)

	body := toJSON(t, map[string]any{
		"email":    "abebe@example.com",
		"password": "password123",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusOK)
}

func TestLogin_MissingPassword(t *testing.T) {
	h := &testAuthHandler{users: &mockUserStore{}, cfg: newTestCfg()}
	r := newAuthRouter(h)

	body := toJSON(t, map[string]any{"phone": "+251911000001"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
}

func TestLogin_NeitherPhoneNorEmail(t *testing.T) {
	h := &testAuthHandler{users: &mockUserStore{}, cfg: newTestCfg()}
	r := newAuthRouter(h)

	body := toJSON(t, map[string]any{"password": "password123"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusBadRequest)
	resp := decodeBody(t, w)
	if resp["error"] != "phone or email required" {
		t.Errorf("unexpected error message: %v", resp["error"])
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	store := &mockUserStore{
		getByPhoneAuthFn: func(_ context.Context, _ string) (*models.User, error) {
			return nil, errors.New("no rows in result set")
		},
	}
	h := &testAuthHandler{users: store, cfg: newTestCfg()}
	r := newAuthRouter(h)

	body := toJSON(t, map[string]any{
		"phone":    "+251999999999",
		"password": "password123",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusUnauthorized)
	resp := decodeBody(t, w)
	if resp["error"] != "invalid credentials" {
		t.Errorf("unexpected error message: %v", resp["error"])
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	hash := validPasswordHash(t)
	store := &mockUserStore{
		getByPhoneAuthFn: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{
				ID:           uuid.New(),
				PasswordHash: hash,
				Role:         models.RoleRenter,
			}, nil
		},
	}
	h := &testAuthHandler{users: store, cfg: newTestCfg()}
	r := newAuthRouter(h)

	body := toJSON(t, map[string]any{
		"phone":    "+251911000001",
		"password": "wrongpassword",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w.Code, http.StatusUnauthorized)
}
