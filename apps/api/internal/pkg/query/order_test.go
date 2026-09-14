package query

import (
	"errors"
	"testing"
)

func TestNormalizeOrder(t *testing.T) {
	allowed := AllowColumns("id", "created_at", "name")
	valid := map[string]string{
		"":                         "",
		"id":                       "id ASC",
		"id desc":                  "id DESC",
		"  CREATED_AT   Desc ":     "created_at DESC",
		"name asc, id desc":        "name ASC, id DESC",
		"created_at DESC,id,name ": "created_at DESC, id ASC, name ASC",
	}
	for input, want := range valid {
		got, err := NormalizeOrder(input, allowed)
		if err != nil || got != want {
			t.Errorf("NormalizeOrder(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
}

func TestNormalizeOrderRejectsInjection(t *testing.T) {
	allowed := AllowColumns("id", "created_at", "name")
	payloads := []string{
		"id; DROP TABLE users",
		"id desc; DROP TABLE users--",
		"(SELECT 1)",
		"id desc, (CASE WHEN (SELECT 1)=1 THEN id ELSE name END)",
		"CASE WHEN 1=1 THEN id END",
		"id desc nulls first",
		"id sideways",
		"password desc",
		"id/**/desc",
		"id--",
		"users.id desc",
		`"id" desc`,
		"id,",
		",id",
		"id desc, name asc, created_at desc, id asc",
		"pg_sleep(10)",
		"1",
		"id\tdesc; select",
	}
	for _, payload := range payloads {
		if got, err := NormalizeOrder(payload, allowed); !errors.Is(err, ErrInvalidOrder) {
			t.Errorf("NormalizeOrder(%q) = %q, %v; want ErrInvalidOrder", payload, got, err)
		}
	}
	if _, err := NormalizeOrder("id", nil); !errors.Is(err, ErrInvalidOrder) {
		t.Fatalf("nil allowlist must reject every column")
	}
}
