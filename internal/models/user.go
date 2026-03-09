package models

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleOwner  UserRole = "owner"
	RoleRenter UserRole = "renter"
	RoleBoth   UserRole = "both"
)

type User struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Phone          string    `json:"phone"`
	Email          *string   `json:"email,omitempty"` // pointer — nullable in DB
	PasswordHash   string    `json:"-"`
	Role           UserRole  `json:"role"`
	TelegramChatID *int64    `json:"telegram_chat_id,omitempty"`
	Verified       bool      `json:"verified"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
