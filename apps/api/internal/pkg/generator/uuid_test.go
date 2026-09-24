package generator

import (
	"strings"
	"testing"
)

func TestGenerateShortUUIDString(t *testing.T) {
	// Generate multiple short UUIDs to verify format and uniqueness
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		uuid := GenerateShortUUIDString()
		if len(uuid) != 8 {
			t.Errorf("expected length 8, got %d", len(uuid))
		}
		if strings.Contains(uuid, "-") {
			t.Errorf("short UUID should not contain hyphens: %s", uuid)
		}
		if seen[uuid] {
			t.Errorf("duplicate short UUID generated: %s", uuid)
		}
		seen[uuid] = true
	}
}

func TestGenerateCustomUUIDNoConfusingChars(t *testing.T) {
	confusingChars := "01oOiIlL"

	tests := []struct {
		name             string
		length           int
		expectedLen      int
		checkNoConfusing bool
	}{
		{name: "default length for zero", length: 0, expectedLen: 8},
		{name: "default length for negative", length: -5, expectedLen: 8},
		{name: "length 16", length: 16, expectedLen: 16},
		{name: "length 32", length: 32, expectedLen: 32},
		{name: "capped at 32", length: 50, expectedLen: 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateCustomUUIDNoConfusingChars(tt.length)
			if len(result) != tt.expectedLen {
				t.Errorf("expected length %d, got %d", tt.expectedLen, len(result))
			}
			for _, char := range confusingChars {
				if strings.Contains(result, string(char)) {
					t.Errorf("result contains confusing character %q: %s", char, result)
				}
			}
		})
	}
}

func TestGenerateCustomUUIDNoConfusingChars_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		uuid := GenerateCustomUUIDNoConfusingChars(16)
		if seen[uuid] {
			t.Errorf("duplicate UUID generated: %s", uuid)
		}
		seen[uuid] = true
	}
}
