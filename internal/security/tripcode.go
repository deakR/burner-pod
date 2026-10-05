package security

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

// sanitizeName strips raw '!', trims whitespace, enforces max 32 runes,
// and defaults to "Anonymous" if empty.
func sanitizeName(name string) string {
	name = strings.ReplaceAll(name, "!", "")
	name = strings.TrimSpace(name)
	runes := []rune(name)
	if len(runes) > 32 {
		name = string(runes[:32])
	}
	if name == "" {
		return "Anonymous"
	}
	return name
}

// ProcessUsername sanitizes a username and handles tripcode authentication.
// It strips all raw '!' characters to prevent spoofing, trims whitespace,
// enforces a maximum length of 32 characters on the username portion, and defaults
// to "Anonymous" if the username is empty or whitespace-only.
// If the input contains '#', it splits into name and password parts:
// - If password is non-empty, computes SHA-256 hash of the password, base64 encodes it,
//   and appends the first 8 characters as a verified tripcode: "Name !xxxxxxxx".
// - If password is empty, returns the sanitized name without a tripcode.
func ProcessUsername(rawName string) string {
	if !strings.Contains(rawName, "#") {
		return sanitizeName(rawName)
	}

	parts := strings.SplitN(rawName, "#", 2)
	namePart := sanitizeName(parts[0])
	passPart := parts[1]

	if passPart == "" {
		return namePart
	}

	hash := sha256.Sum256([]byte(passPart))
	encoded := base64.StdEncoding.EncodeToString(hash[:])
	trip := encoded[:8]

	return namePart + " !" + trip
}

