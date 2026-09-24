package permission

import (
	"testing"
)

func TestStringSlice_Contains(t *testing.T) {
	tests := []struct {
		name     string
		slice    StringSlice
		item     string
		expected bool
	}{
		{name: "contains item", slice: StringSlice{"GET", "POST"}, item: "GET", expected: true},
		{name: "does not contain", slice: StringSlice{"GET", "POST"}, item: "DELETE", expected: false},
		{name: "empty slice", slice: StringSlice{}, item: "GET", expected: false},
		{name: "nil slice", slice: nil, item: "GET", expected: false},
		{name: "single match", slice: StringSlice{"POST"}, item: "POST", expected: true},
		{name: "case sensitive", slice: StringSlice{"GET"}, item: "get", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.slice.Contains(tt.item)
			if result != tt.expected {
				t.Errorf("Contains(%q) = %v, want %v", tt.item, result, tt.expected)
			}
		})
	}
}

func TestStringSlice_Scan(t *testing.T) {
	t.Run("scans from []byte", func(t *testing.T) {
		var s StringSlice
		err := s.Scan([]byte(`["a","b","c"]`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(s) != 3 || s[0] != "a" || s[1] != "b" || s[2] != "c" {
			t.Errorf("unexpected result: %v", s)
		}
	})

	t.Run("scans from string", func(t *testing.T) {
		var s StringSlice
		err := s.Scan(`["x","y"]`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(s) != 2 {
			t.Errorf("expected length 2, got %d", len(s))
		}
	})

	t.Run("nil value", func(t *testing.T) {
		var s StringSlice
		err := s.Scan(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s != nil {
			t.Errorf("expected nil, got %v", s)
		}
	})

	t.Run("empty bytes", func(t *testing.T) {
		var s StringSlice
		err := s.Scan([]byte{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s != nil {
			t.Errorf("expected nil, got %v", s)
		}
	})

	t.Run("invalid type returns error", func(t *testing.T) {
		var s StringSlice
		err := s.Scan(42)
		if err == nil {
			t.Error("expected error for invalid type, got nil")
		}
	})
}

func TestStringSlice_Value(t *testing.T) {
	t.Run("returns JSON for populated slice", func(t *testing.T) {
		s := StringSlice{"GET", "POST"}
		val, err := s.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		b, ok := val.([]byte)
		if !ok {
			t.Fatalf("expected []byte, got %T", val)
		}
		if string(b) != `["GET","POST"]` {
			t.Errorf("unexpected JSON: %s", string(b))
		}
	})

	t.Run("returns nil for nil slice", func(t *testing.T) {
		var s StringSlice
		val, err := s.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != nil {
			t.Errorf("expected nil, got %v", val)
		}
	})
}

func TestActionConstants(t *testing.T) {
	if ActionGET != "GET" {
		t.Errorf("ActionGET = %q, want %q", ActionGET, "GET")
	}
	if ActionPOST != "POST" {
		t.Errorf("ActionPOST = %q, want %q", ActionPOST, "POST")
	}
	if ActionPUT != "PUT" {
		t.Errorf("ActionPUT = %q, want %q", ActionPUT, "PUT")
	}
	if ActionPATCH != "PATCH" {
		t.Errorf("ActionPATCH = %q, want %q", ActionPATCH, "PATCH")
	}
	if ActionDELETE != "DELETE" {
		t.Errorf("ActionDELETE = %q, want %q", ActionDELETE, "DELETE")
	}

	if len(AllActions) != 5 {
		t.Errorf("AllActions should have 5 elements, got %d", len(AllActions))
	}
}

func TestRoleConstants(t *testing.T) {
	if RoleAdmin != "admin" {
		t.Errorf("RoleAdmin = %q, want %q", RoleAdmin, "admin")
	}
	if RoleAuditor != "auditor" {
		t.Errorf("RoleAuditor = %q, want %q", RoleAuditor, "auditor")
	}
	if RoleUser != "user" {
		t.Errorf("RoleUser = %q, want %q", RoleUser, "user")
	}
}

func TestResourceCategoryConstants(t *testing.T) {
	if CategoryAdmin != "admin" {
		t.Errorf("CategoryAdmin = %q, want %q", CategoryAdmin, "admin")
	}
	if CategoryUser != "user" {
		t.Errorf("CategoryUser = %q, want %q", CategoryUser, "user")
	}
}
