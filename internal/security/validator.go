package security

import (
	"errors"
	"regexp"
)

const (
	// MinTTL is the minimum allowed room time-to-live in seconds (1 second).
	MinTTL = 1
	// MaxTTL is the maximum allowed room time-to-live in seconds (7 days = 604,800 seconds).
	MaxTTL = 604800
	// DefaultTTL is the default room time-to-live in seconds (60 seconds).
	DefaultTTL = 60
	// MaxRoomIDLength is the maximum allowed length of a room identifier.
	MaxRoomIDLength = 64
	// MinRoomIDLength is the minimum allowed length of a room identifier.
	MinRoomIDLength = 1
)

var (
	// ErrEmptyRoomID indicates that the room ID is empty.
	ErrEmptyRoomID = errors.New("room ID cannot be empty")
	// ErrRoomIDTooLong indicates that the room ID exceeds the maximum allowed length of 64 characters.
	ErrRoomIDTooLong = errors.New("room ID exceeds maximum length of 64 characters")
	// ErrInvalidRoomID indicates that the room ID contains characters outside ^[a-zA-Z0-9_-]{1,64}$.
	ErrInvalidRoomID = errors.New("room ID contains invalid characters; must contain only alphanumeric characters, underscores, and hyphens (^[a-zA-Z0-9_-]{1,64}$)")
)

// roomIDRegex enforces alphanumeric characters, hyphens, and underscores, between 1 and 64 characters.
var roomIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// ValidateRoomID checks that roomID satisfies the strict regex ^[a-zA-Z0-9_-]{1,64}$.
// Returns ErrEmptyRoomID if roomID is empty.
// Returns ErrRoomIDTooLong if len(roomID) > 64.
// Returns ErrInvalidRoomID if roomID contains any illegal characters.
// Returns nil if valid.
func ValidateRoomID(roomID string) error {
	if len(roomID) == 0 {
		return ErrEmptyRoomID
	}
	if len(roomID) > MaxRoomIDLength {
		return ErrRoomIDTooLong
	}
	if !roomIDRegex.MatchString(roomID) {
		return ErrInvalidRoomID
	}
	return nil
}

// ValidateTTL clamps the provided TTL seconds between MinTTL (1) and MaxTTL (604800).
// If seconds is 0, it returns DefaultTTL (60).
// If seconds is negative, it clamps to MinTTL (1).
// If seconds exceeds MaxTTL, it clamps to MaxTTL (604800).
func ValidateTTL(seconds int) int {
	if seconds == 0 {
		return DefaultTTL
	}
	if seconds < MinTTL {
		return MinTTL
	}
	if seconds > MaxTTL {
		return MaxTTL
	}
	return seconds
}
