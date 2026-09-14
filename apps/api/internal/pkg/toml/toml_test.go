package toml

import (
	"testing"
)

func TestUnmarshal(t *testing.T) {
	t.Run("parses simple TOML", func(t *testing.T) {
		input := `
title = "Test Config"
port = 8080
`
		var config struct {
			Title string
			Port  int
		}
		err := Unmarshal([]byte(input), &config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.Title != "Test Config" {
			t.Errorf("expected Title 'Test Config', got %q", config.Title)
		}
		if config.Port != 8080 {
			t.Errorf("expected Port 8080, got %d", config.Port)
		}
	})

	t.Run("parses nested TOML", func(t *testing.T) {
		input := `
[database]
host = "localhost"
port = 5432
`
		var config struct {
			Database struct {
				Host string
				Port int
			}
		}
		err := Unmarshal([]byte(input), &config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.Database.Host != "localhost" {
			t.Errorf("expected Host 'localhost', got %q", config.Database.Host)
		}
		if config.Database.Port != 5432 {
			t.Errorf("expected Port 5432, got %d", config.Database.Port)
		}
	})

	t.Run("returns error for invalid TOML", func(t *testing.T) {
		input := `invalid = `
		var config struct{}
		err := Unmarshal([]byte(input), &config)
		if err == nil {
			t.Error("expected error for invalid TOML, got nil")
		}
	})
}

func TestMarshal(t *testing.T) {
	t.Run("marshals struct to TOML", func(t *testing.T) {
		config := struct {
			Name string
			Age  int
		}{
			Name: "Alice",
			Age:  30,
		}

		result, err := Marshal(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected non-empty output")
		}
	})
}

func TestMarshalToString(t *testing.T) {
	t.Run("returns string representation", func(t *testing.T) {
		config := struct {
			Title string
		}{
			Title: "Hello",
		}

		result, err := MarshalToString(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == "" {
			t.Error("expected non-empty string")
		}
	})
}

func TestDecode(t *testing.T) {
	t.Run("decodes TOML string", func(t *testing.T) {
		input := `
key = "value"
count = 42
`
		var result struct {
			Key   string
			Count int
		}
		_, err := Decode(input, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Key != "value" {
			t.Errorf("expected 'value', got %q", result.Key)
		}
	})
}
