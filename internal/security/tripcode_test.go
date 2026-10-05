package security

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

func TestProcessUsername(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// 1. Empty and whitespace
		{
			name:     "empty input defaults to Anonymous",
			input:    "",
			expected: "Anonymous",
		},
		{
			name:     "whitespace only defaults to Anonymous",
			input:    "   \t\n  ",
			expected: "Anonymous",
		},

		// 2. Plain username without tripcode
		{
			name:     "clean username preserved",
			input:    "Alice",
			expected: "Alice",
		},
		{
			name:     "username with leading and trailing whitespace trimmed",
			input:    "   Bob Smith   ",
			expected: "Bob Smith",
		},

		// 3. Exclamation mark stripping (anti-spoofing)
		{
			name:     "raw exclamation stripped",
			input:    "Alice!admin",
			expected: "Aliceadmin",
		},
		{
			name:     "multiple exclamations stripped",
			input:    "!!!Bob!hacker!!!",
			expected: "Bobhacker",
		},
		{
			name:     "spoofed tripcode signature stripped",
			input:    "Alice !12345678",
			expected: "Alice 12345678",
		},
		{
			name:     "only exclamation marks defaults to Anonymous",
			input:    "!!!!!!",
			expected: "Anonymous",
		},
		{
			name:     "exclamations and whitespace defaults to Anonymous",
			input:    "  !  !  ",
			expected: "Anonymous",
		},

		// 4. Username length capping (max 32 runes)
		{
			name:     "exact 32 characters preserved",
			input:    strings.Repeat("A", 32),
			expected: strings.Repeat("A", 32),
		},
		{
			name:     "over 32 characters truncated to 32",
			input:    strings.Repeat("A", 40),
			expected: strings.Repeat("A", 32),
		},
		{
			name:     "50 characters truncated to 32",
			input:    strings.Repeat("B", 50),
			expected: strings.Repeat("B", 32),
		},

		// 5. Valid tripcode generation
		{
			name:     "valid tripcode calculation",
			input:    "Alice#secret123",
			expected: "Alice !/PcwttlS",
		},
		{
			name:     "valid tripcode with different password",
			input:    "Bob#secretpass",
			expected: "Bob !4F95ZR1G",
		},

		// 6. Leading hash (anonymous user with tripcode)
		{
			name:     "leading hash defaults name to Anonymous",
			input:    "#secretpass",
			expected: "Anonymous !4F95ZR1G",
		},
		{
			name:     "leading whitespace before hash defaults to Anonymous",
			input:    "   #secretpass",
			expected: "Anonymous !4F95ZR1G",
		},

		// 7. Trailing hash without password
		{
			name:     "trailing hash with name returns name only",
			input:    "Alice#",
			expected: "Alice",
		},
		{
			name:     "single hash only returns Anonymous",
			input:    "#",
			expected: "Anonymous",
		},
		{
			name:     "whitespace before trailing hash returns Anonymous",
			input:    "   #",
			expected: "Anonymous",
		},

		// 8. Sanitization combined with tripcode
		{
			name:     "name with raw exclamation and tripcode",
			input:    "Alice!Admin#secret123",
			expected: "AliceAdmin !/PcwttlS",
		},
		{
			name:     "name only exclamations with tripcode defaults to Anonymous",
			input:    "!!!!#secret123",
			expected: "Anonymous !/PcwttlS",
		},
		{
			name:     "name padded with whitespace before tripcode",
			input:    "  Alice  #secret123",
			expected: "Alice !/PcwttlS",
		},
		{
			name:     "long name truncated to 32 before tripcode",
			input:    strings.Repeat("A", 40) + "#secret123",
			expected: strings.Repeat("A", 32) + " !/PcwttlS",
		},

		// 9. Unicode and Emoji support
		{
			name:     "unicode and emoji username preserved",
			input:    "🔥User🚀",
			expected: "🔥User🚀",
		},
		{
			name:     "unicode and emoji with tripcode",
			input:    "🔥User🚀#key",
			expected: "🔥User🚀 !LHDhK3oG",
		},
		{
			name:     "unicode with raw exclamation and tripcode",
			input:    "🔥!User!🚀#key",
			expected: "🔥User🚀 !LHDhK3oG",
		},

		// 10. Multiple '#' characters
		{
			name:     "multiple hashes treat first as split point",
			input:    "Alice#pass#extra",
			expected: "Alice !" + computeExpectedTrip("pass#extra"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessUsername(tt.input)
			if result != tt.expected {
				t.Errorf("ProcessUsername(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTripcodeDeterminism(t *testing.T) {
	pass := "ConsistentPassphrase123"
	u1 := ProcessUsername("IdentityOne#" + pass)
	u2 := ProcessUsername("IdentityTwo#" + pass)

	trip1 := extractTripcode(u1)
	trip2 := extractTripcode(u2)

	if trip1 == "" || trip2 == "" {
		t.Fatalf("Tripcode extraction failed: u1=%q, u2=%q", u1, u2)
	}

	if trip1 != trip2 {
		t.Errorf("Tripcode mismatch for identical passphrase: %q != %q", trip1, trip2)
	}
}

func TestAntiSpoofingTripcodePresentation(t *testing.T) {
	pass := "LegitSecret"
	legit := ProcessUsername("Alice#" + pass)
	trip := extractTripcode(legit)

	// Attacker attempts to spoof legitimate tripcode without '#'
	spoofedInput := "Alice !" + trip
	result := ProcessUsername(spoofedInput)

	if strings.Contains(result, "!") {
		t.Errorf("Spoofing vulnerability: result still contains '!': %q", result)
	}

	if result == legit {
		t.Errorf("Spoofing vulnerability: attacker matched legitimate tripcode: %q == %q", result, legit)
	}
}

func extractTripcode(username string) string {
	parts := strings.Split(username, " !")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

func computeExpectedTrip(pass string) string {
	hash := sha256.Sum256([]byte(pass))
	encoded := base64.StdEncoding.EncodeToString(hash[:])
	return encoded[:8]
}
