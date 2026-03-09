package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"qetero/internal/models"
	"qetero/internal/repository"
)

type ListingHandler struct {
	listings *repository.ListingRepo
	bookings *repository.BookingRepo
}

func NewListingHandler(db *pgxpool.Pool) *ListingHandler {
	return &ListingHandler{
		listings: repository.NewListingRepo(db),
		bookings: repository.NewBookingRepo(db),
	}
}

type createListingRequest struct {
	Title       string                 `json:"title"        binding:"required"`
	Category    models.ListingCategory `json:"category"     binding:"required,oneof=excavator crane scaffold compactor loader forklift generator water_truck concrete_mixer dump_truck dozer roller other"`
	Description string                 `json:"description"  binding:"required"`
	Location    string                 `json:"location"     binding:"required"`
	PricePerDay float64                `json:"price_per_day" binding:"required,gt=0"`
	MinimumDays int                    `json:"minimum_days"`
	Specs       map[string]any         `json:"specs"`
}

func (h *ListingHandler) Create(c *gin.Context) {
	ownerID := c.MustGet("user_id").(uuid.UUID)

	var req createListingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.MinimumDays < 1 {
		req.MinimumDays = 1
	}

	specsJSON, err := json.Marshal(req.Specs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid specs"})
		return
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
		Specs:       specsJSON,
	}

	if err := h.listings.Create(c.Request.Context(), listing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create listing"})
		return
	}

	c.JSON(http.StatusCreated, listing)
}

func (h *ListingHandler) Get(c *gin.Context) {
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

func (h *ListingHandler) Browse(c *gin.Context) {
	f := repository.ListingFilter{
		Category: c.Query("category"),
		Location: c.Query("location"),
	}
	if v := c.Query("min_price"); v != "" {
		if p, err := strconv.ParseFloat(v, 64); err == nil {
			f.MinPrice = &p
		}
	}
	if v := c.Query("max_price"); v != "" {
		if p, err := strconv.ParseFloat(v, 64); err == nil {
			f.MaxPrice = &p
		}
	}
	if v := c.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			f.Page = p
		}
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

func (h *ListingHandler) Update(c *gin.Context) {
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

	specsJSON, err := json.Marshal(req.Specs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid specs"})
		return
	}

	listing.Title = req.Title
	listing.Category = req.Category
	listing.Description = req.Description
	listing.Location = req.Location
	listing.PricePerDay = req.PricePerDay
	listing.MinimumDays = req.MinimumDays
	listing.Specs = specsJSON

	if err := h.listings.Update(c.Request.Context(), listing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update listing"})
		return
	}

	c.JSON(http.StatusOK, listing)
}

func (h *ListingHandler) Delete(c *gin.Context) {
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

func (h *ListingHandler) Availability(c *gin.Context) {
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
