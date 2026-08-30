package models

import "time"

const (
	TelegramRoleOwner  = "owner"
	TelegramRoleAdmin  = "admin"
	TelegramRoleUser   = "user"
	TelegramRoleViewer = "viewer"
)

// TelegramConfiguration controls Telegram access to Couchness.
// The bot token is intentionally not persisted; it is read from the environment.
type TelegramConfiguration struct {
	Enabled bool                       `json:"enabled"`
	Users   map[string]*TelegramUser   `json:"users"`
	Invites map[string]*TelegramInvite `json:"invites,omitempty"`
}

// TelegramUser is an authorized Telegram account.
type TelegramUser struct {
	ID      int64     `json:"id"`
	Role    string    `json:"role"`
	AddedAt time.Time `json:"added_at"`
}

// TelegramInvite is a single-use, expiring bot invitation.
type TelegramInvite struct {
	Code      string    `json:"code"`
	Role      string    `json:"role"`
	CreatedBy int64     `json:"created_by"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ValidTelegramRole reports whether role is supported.
func ValidTelegramRole(role string) bool {
	switch role {
	case TelegramRoleOwner, TelegramRoleAdmin, TelegramRoleUser, TelegramRoleViewer:
		return true
	default:
		return false
	}
}
