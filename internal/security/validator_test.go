package security_test

import (
	"errors"
	"strings"
	"testing"

	"burner-pod/internal/security"
)

func TestValidateRoomID_Valid(t *testing.T) {
	validIDs := []string{
		"a",
		"Z",
		"0",
		"_",
		"-",
		"room",
		"room-123",
		"valid_Room-123_",
		"ROOM_ID",
		"123456",
		"a-b_c-d_1-2",
		strings.Repeat("a", 64),
		strings.Repeat("X", 60) + "-123",
	}

	for _, id := range validIDs {
		t.Run(id, func(t *testing.T) {
			if err := security.ValidateRoomID(id); err != nil {
				t.Errorf("Expected valid room ID for '%s', got error: %v", id, err)
			}
		})
	}
}

func TestValidateRoomID_Empty(t *testing.T) {
	err := security.ValidateRoomID("")
	if err == nil {
		t.Fatal("Expected error for empty room ID, got nil")
	}
	if !errors.Is(err, security.ErrEmptyRoomID) {
		t.Errorf("Expected ErrEmptyRoomID, got: %v", err)
	}
}

func TestValidateRoomID_LengthBoundaries(t *testing.T) {
	// Length 1: valid
	if err := security.ValidateRoomID("x"); err != nil {
		t.Errorf("Expected length 1 to be valid, got: %v", err)
	}

	// Length 64: valid
	len64 := strings.Repeat("x", 64)
	if err := security.ValidateRoomID(len64); err != nil {
		t.Errorf("Expected length 64 to be valid, got: %v", err)
	}

	// Length 65: invalid (too long)
	len65 := strings.Repeat("x", 65)
	err := security.ValidateRoomID(len65)
	if err == nil {
		t.Fatal("Expected error for length 65 room ID, got nil")
	}
	if !errors.Is(err, security.ErrRoomIDTooLong) {
		t.Errorf("Expected ErrRoomIDTooLong, got: %v", err)
	}

	// Extreme length 500: invalid (too long)
	len500 := strings.Repeat("x", 500)
	err = security.ValidateRoomID(len500)
	if err == nil {
		t.Fatal("Expected error for length 500 room ID, got nil")
	}
	if !errors.Is(err, security.ErrRoomIDTooLong) {
		t.Errorf("Expected ErrRoomIDTooLong, got: %v", err)
	}
}

func TestValidateRoomID_InvalidCharacters(t *testing.T) {
	invalidCases := []struct {
		name string
		id   string
	}{
		{"space in middle", "room with space"},
		{"leading space", " room"},
		{"trailing space", "room "},
		{"only spaces", "   "},
		{"dollar sign", "room$special"},
		{"at and exclamation", "room@bad!"},
		{"html tag", "room<script>"},
		{"parent path traversal", "../traversal"},
		{"deep path traversal", "../../admin"},
		{"current dir traversal", "./room"},
		{"hash symbol", "room#tag"},
		{"plus symbol", "room+plus"},
		{"equals symbol", "room=value"},
		{"question mark", "room?query"},
		{"percent encoded", "room%20id"},
		{"asterisk", "room*star"},
		{"pipe", "room|pipe"},
		{"quotes", "room\"quote\""},
		{"single quote", "room'quote"},
		{"backtick", "room`tick`"},
		{"curly braces", "room{123}"},
		{"brackets", "room[123]"},
		{"colon", "room:1"},
		{"semicolon", "room;1"},
		{"dot", "room.test"},
		{"emoji", "room🔥"},
		{"unicode", "café"},
		{"newline", "room\n"},
		{"tab", "room\t"},
		{"null byte", "room\x00"},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			err := security.ValidateRoomID(tc.id)
			if err == nil {
				t.Errorf("Expected error for invalid room ID '%s', got nil", tc.id)
			}
			// When length <= 64, error must be ErrInvalidRoomID
			if len(tc.id) <= 64 && !errors.Is(err, security.ErrInvalidRoomID) {
				t.Errorf("Expected ErrInvalidRoomID for '%s', got: %v", tc.id, err)
			}
		})
	}
}

func TestValidateRoomID_GeneratedIDsAlwaysValid(t *testing.T) {
	for i := 0; i < 100; i++ {
		id, err := security.GenerateRoomID()
		if err != nil {
			t.Fatalf("GenerateRoomID failed at iteration %d: %v", i, err)
		}
		if err := security.ValidateRoomID(id); err != nil {
			t.Errorf("Generated room ID '%s' failed validation: %v", id, err)
		}
	}
}

func TestValidateTTL_ClampingAndDefaults(t *testing.T) {
	testCases := []struct {
		name     string
		input    int
		expected int
	}{
		{"zero defaults to 60", 0, 60},
		{"negative 10 clamps to min 1", -10, 1},
		{"negative 1 clamps to min 1", -1, 1},
		{"negative extreme clamps to min 1", -999999, 1},
		{"min boundary 1", 1, 1},
		{"normal 2 seconds", 2, 2},
		{"normal 60 seconds", 60, 60},
		{"normal 300 seconds", 300, 300},
		{"normal 86400 (1 day)", 86400, 86400},
		{"max boundary 604800 (7 days)", 604800, 604800},
		{"exceeding max by 1 clamps to max", 604801, 604800},
		{"extreme high clamps to max", 999999999, 604800},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := security.ValidateTTL(tc.input)
			if actual != tc.expected {
				t.Errorf("ValidateTTL(%d) = %d; expected %d", tc.input, actual, tc.expected)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	if security.MinTTL != 1 {
		t.Errorf("Expected MinTTL=1, got %d", security.MinTTL)
	}
	if security.MaxTTL != 604800 {
		t.Errorf("Expected MaxTTL=604800, got %d", security.MaxTTL)
	}
	if security.DefaultTTL != 60 {
		t.Errorf("Expected DefaultTTL=60, got %d", security.DefaultTTL)
	}
	if security.MaxRoomIDLength != 64 {
		t.Errorf("Expected MaxRoomIDLength=64, got %d", security.MaxRoomIDLength)
	}
	if security.MinRoomIDLength != 1 {
		t.Errorf("Expected MinRoomIDLength=1, got %d", security.MinRoomIDLength)
	}
}
