package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"qetero/internal/config"
	"qetero/internal/db"
	"qetero/internal/handlers"
	"qetero/internal/middleware"
	"qetero/internal/telegram"
)

func main() {
	_ = godotenv.Load() // load .env if present; ignored in production where env vars are set externally

	cfg := config.Load()

	pool, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}
	defer pool.Close()

	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Public routes
	authHandler := handlers.NewAuthHandler(pool, cfg)
	r.POST("/v1/auth/register", authHandler.Register)
	r.POST("/v1/auth/login", authHandler.Login)

	// Public listing browse (no auth required to look around)
	listingHandler := handlers.NewListingHandler(pool)
	r.GET("/v1/listings", listingHandler.Browse)
	r.GET("/v1/listings/:id", listingHandler.Get)
	r.GET("/v1/listings/:id/availability", listingHandler.Availability)

	// Protected routes
	v1 := r.Group("/v1")
	v1.Use(middleware.Auth(cfg.JWTSecret))
	{
		// Listings (write operations)
		v1.POST("/listings", listingHandler.Create)
		v1.PUT("/listings/:id", listingHandler.Update)
		v1.DELETE("/listings/:id", listingHandler.Delete)

		// Bookings
		bookingHandler := handlers.NewBookingHandler(pool)
		v1.POST("/bookings", bookingHandler.Create)
		v1.GET("/bookings/:id", bookingHandler.Get)
		v1.PUT("/bookings/:id/confirm", bookingHandler.Confirm)
		v1.PUT("/bookings/:id/cancel", bookingHandler.Cancel)

		// Users
		userHandler := handlers.NewUserHandler(pool)
		v1.GET("/users/me", userHandler.Me)
		v1.PUT("/users/me", userHandler.Update)
		v1.GET("/users/me/bookings", userHandler.MyBookings)
		v1.GET("/users/me/listings/bookings", userHandler.IncomingBookings)
	}

	// Start Telegram bot if token is configured
	if botToken := os.Getenv("TELEGRAM_BOT_TOKEN"); botToken != "" {
		bot, err := telegram.New(botToken, pool, cfg.AdminTelegramChatID, cfg.AdminTelebirr)
		if err != nil {
			log.Printf("Warning: failed to start Telegram bot: %v", err)
		} else {
			go bot.Start(context.Background())
			log.Println("Telegram bot started")
		}
	}

	log.Printf("Server starting on :%s", cfg.Port)
	log.Fatal(r.Run(":" + cfg.Port))
}
