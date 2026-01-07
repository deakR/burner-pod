package utils

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateRoomID creates a cryptographically secure, random, URL-safe string for room identifiers.
// Returns a base64-encoded string from 12 random bytes, suitable for use in URLs.
// Returns an error if the system's random number generator fails.
func GenerateRoomID() (string, error) {
	b := make([]byte, 12)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
