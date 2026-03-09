package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"qetero/internal/auth"
	"qetero/internal/config"
	"qetero/internal/models"
	"qetero/internal/repository"
)

type AuthHandler struct {
	users *repository.UserRepo
	cfg   *config.Config
}

func NewAuthHandler(db *pgxpool.Pool, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		users: repository.NewUserRepo(db),
		cfg:   cfg,
	}
}

type registerRequest struct {
	Name     string          `json:"name"     binding:"required"`
	Phone    string          `json:"phone"    binding:"required"`
	Email    string          `json:"email"    binding:"omitempty,email"`
	Password string          `json:"password" binding:"required,min=8"`
	Role     models.UserRole `json:"role"     binding:"required,oneof=owner renter both"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
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

	token, err := auth.GenerateToken(user.ID, string(user.Role), h.cfg.JWTSecret, h.cfg.JWTAccessExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": token, "user": user})
}

type loginRequest struct {
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
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
		// Return same error for both "not found" and "wrong password" to prevent user enumeration
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := auth.GenerateToken(user.ID, string(user.Role), h.cfg.JWTSecret, h.cfg.JWTAccessExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}
