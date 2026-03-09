package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"qetero/internal/repository"
)

type UserHandler struct {
	users    *repository.UserRepo
	bookings *repository.BookingRepo
}

func NewUserHandler(db *pgxpool.Pool) *UserHandler {
	return &UserHandler{
		users:    repository.NewUserRepo(db),
		bookings: repository.NewBookingRepo(db),
	}
}

func (h *UserHandler) Me(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

type updateUserRequest struct {
	Name  string `json:"name"  binding:"required"`
	Phone string `json:"phone"`
}

func (h *UserHandler) Update(c *gin.Context) {
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

func (h *UserHandler) MyBookings(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	bookings, err := h.bookings.GetByRenter(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch bookings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bookings": bookings})
}

func (h *UserHandler) IncomingBookings(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	bookings, err := h.bookings.GetByOwner(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch bookings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bookings": bookings})
}
