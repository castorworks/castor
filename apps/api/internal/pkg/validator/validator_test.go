package validator

import (
	"testing"
)

func TestIsAlphaNumeric(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "letters only", input: "hello", expected: true},
		{name: "numbers only", input: "12345", expected: true},
		{name: "alphanumeric", input: "hello123", expected: true},
		{name: "with underscore", input: "hello_world", expected: true},
		{name: "with dot", input: "hello.world", expected: true},
		{name: "with hyphen", input: "hello-world", expected: true},
		{name: "mixed valid chars", input: "h.ello-123_world", expected: true},
		{name: "empty string", input: "", expected: true},
		{name: "space not allowed", input: "hello world", expected: false},
		{name: "slash not allowed", input: "hello/world", expected: false},
		{name: "at sign not allowed", input: "hello@world", expected: false},
		{name: "exclamation not allowed", input: "hello!", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAlphaNumeric(tt.input)
			if result != tt.expected {
				t.Errorf("IsAlphaNumeric(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "simple filename", input: "report.pdf", expected: true},
		{name: "with spaces in middle", input: "my report.pdf", expected: true},
		{name: "chinese characters", input: "报告.pdf", expected: true},
		{name: "with parentheses", input: "report (1).pdf", expected: true},
		{name: "with underscores", input: "my_report.pdf", expected: true},
		{name: "with hyphens", input: "my-report.pdf", expected: true},
		{name: "empty string", input: "", expected: false},
		{name: "starts with dot", input: ".hidden", expected: false},
		{name: "starts with space", input: " report.pdf", expected: false},
		{name: "ends with space", input: "report.pdf ", expected: false},
		{name: "contains slash", input: "path/report.pdf", expected: false},
		{name: "contains backslash", input: "path\\report.pdf", expected: false},
		{name: "contains colon", input: "report:file.pdf", expected: false},
		{name: "contains asterisk", input: "report*.pdf", expected: false},
		{name: "contains question mark", input: "report?.pdf", expected: false},
		{name: "contains pipe", input: "report|file.pdf", expected: false},
		{name: "CON reserved", input: "CON.txt", expected: false},
		{name: "PRN reserved", input: "PRN", expected: false},
		{name: "AUX reserved", input: "aux.log", expected: false},
		{name: "NUL reserved", input: "NUL", expected: false},
		{name: "COM1 reserved", input: "COM1", expected: false},
		{name: "LPT1 reserved", input: "LPT1.txt", expected: false},
		{name: "lowercase reserved", input: "con.pdf", expected: false},
		{name: "case insensitive reserved", input: "Con.txt", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidFilename(tt.input)
			if result != tt.expected {
				t.Errorf("IsValidFilename(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsMobile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "valid 11-digit starting with 13", input: "13812345678", expected: true},
		{name: "valid 11-digit starting with 15", input: "15812345678", expected: true},
		{name: "valid 11-digit starting with 18", input: "18812345678", expected: true},
		{name: "valid 11-digit starting with 19", input: "19912345678", expected: true},
		{name: "too short", input: "1381234567", expected: false},
		{name: "too long", input: "138123456789", expected: false},
		{name: "starts with 10 invalid", input: "10812345678", expected: false},
		{name: "starts with 12 invalid", input: "12812345678", expected: false},
		{name: "starts with 2 invalid", input: "23812345678", expected: false},
		{name: "empty", input: "", expected: false},
		{name: "contains letters", input: "13812345abc", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMobile(tt.input)
			if result != tt.expected {
				t.Errorf("IsMobile(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "valid simple", input: "user@example.com", expected: true},
		{name: "valid with dots", input: "user.name@example.com", expected: true},
		{name: "valid with plus", input: "user+tag@example.com", expected: true},
		{name: "valid subdomain", input: "user@mail.example.com", expected: true},
		{name: "valid with numbers", input: "user123@example123.com", expected: true},
		{name: "valid underscore", input: "user_name@example.com", expected: true},
		{name: "valid hyphen", input: "user-name@example-domain.com", expected: true},
		{name: "valid long TLD", input: "user@example.museum", expected: true},
		{name: "empty", input: "", expected: false},
		{name: "no at sign", input: "userexample.com", expected: false},
		{name: "no domain", input: "user@", expected: false},
		{name: "no TLD", input: "user@example", expected: false},
		{name: "single char TLD invalid", input: "user@example.c", expected: false},
		{name: "spaces", input: "user @example.com", expected: false},
		{name: "double at", input: "user@@example.com", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEmail(tt.input)
			if result != tt.expected {
				t.Errorf("IsEmail(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
