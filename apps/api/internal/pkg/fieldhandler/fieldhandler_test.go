package fieldhandler

import (
	"testing"
)

// Test model
type testModel struct {
	ID   int
	Name string
	Age  int
}

// Test request
type testUpdateReq struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestGetFieldHandlers(t *testing.T) {
	model := &testModel{}

	t.Run("creates handlers for all fields", func(t *testing.T) {
		handlers := GetFieldHandlers(model, nil)
		if _, ok := handlers["id"]; !ok {
			t.Error("expected handler for id")
		}
		if _, ok := handlers["name"]; !ok {
			t.Error("expected handler for name")
		}
		if _, ok := handlers["age"]; !ok {
			t.Error("expected handler for age")
		}
	})

	t.Run("skips specified fields", func(t *testing.T) {
		skipFn := func(name string) bool {
			return name == "ID"
		}
		handlers := GetFieldHandlers(model, skipFn)
		if _, ok := handlers["id"]; ok {
			t.Error("expected ID to be skipped")
		}
		if _, ok := handlers["name"]; !ok {
			t.Error("expected name handler to exist")
		}
	})

	t.Run("skips default audit fields", func(t *testing.T) {
		type modelWithAudit struct {
			ID        int    `json:"id"`
			Name      string `json:"name"`
			CreatedBy int    `json:"createdBy"`
		}
		m := &modelWithAudit{}
		handlers := GetFieldHandlers(m, func(name string) bool {
			return DefaultSkipFields[name]
		})
		if _, ok := handlers["id"]; ok {
			t.Error("expected id to be skipped")
		}
		if _, ok := handlers["createdBy"]; ok {
			t.Error("expected createdBy to be skipped")
		}
	})
}

func TestApplyUpdates(t *testing.T) {
	t.Run("applies valid updates", func(t *testing.T) {
		model := &testModel{Name: "old", Age: 0}
		handlers := GetFieldHandlers(model, nil)

		updates := map[string]interface{}{
			"name": "new",
			"age":  25,
		}

		err := ApplyUpdates(model, updates, handlers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model.Name != "new" {
			t.Errorf("expected name 'new', got %q", model.Name)
		}
		if model.Age != 25 {
			t.Errorf("expected age 25, got %d", model.Age)
		}
	})

	t.Run("returns error for type mismatch", func(t *testing.T) {
		model := &testModel{}
		handlers := GetFieldHandlers(model, nil)

		updates := map[string]interface{}{
			"age": "not a number",
		}

		err := ApplyUpdates(model, updates, handlers)
		if err == nil {
			t.Error("expected error for type mismatch, got nil")
		}
	})

	t.Run("skips unknown fields without error", func(t *testing.T) {
		model := &testModel{Name: "keep"}
		handlers := GetFieldHandlers(model, nil)

		updates := map[string]interface{}{
			"unknown": "value",
			"name":    "updated",
		}

		err := ApplyUpdates(model, updates, handlers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model.Name != "updated" {
			t.Errorf("expected name 'updated', got %q", model.Name)
		}
	})

	t.Run("empty updates does nothing", func(t *testing.T) {
		model := &testModel{Name: "original"}
		handlers := GetFieldHandlers(model, nil)

		err := ApplyUpdates(model, map[string]interface{}{}, handlers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model.Name != "original" {
			t.Errorf("name should be unchanged, got %q", model.Name)
		}
	})
}

func TestStructToMap(t *testing.T) {
	t.Run("converts struct to map", func(t *testing.T) {
		req := &testUpdateReq{Name: "test", Age: 30}
		result := StructToMap(req)

		if result["name"] != "test" {
			t.Errorf("expected name 'test', got %v", result["name"])
		}
		if result["age"] != 30 {
			t.Errorf("expected age 30, got %v", result["age"])
		}
	})

	t.Run("uses json tags as keys", func(t *testing.T) {
		type reqWithCustomTag struct {
			FirstName string `json:"first_name"`
		}
		req := &reqWithCustomTag{FirstName: "John"}
		result := StructToMap(req)

		if _, ok := result["FirstName"]; ok {
			t.Error("should use json tag as key, not field name")
		}
		if result["first_name"] != "John" {
			t.Errorf("expected first_name 'John', got %v", result["first_name"])
		}
	})

	t.Run("handles structs without json tags", func(t *testing.T) {
		type reqNoTags struct {
			Title string
		}
		req := &reqNoTags{Title: "hello"}
		result := StructToMap(req)

		if result["title"] != "hello" {
			t.Errorf("expected title 'hello', got %v", result["title"])
		}
	})
}

func TestDefaultSkipFields(t *testing.T) {
	expectedFields := []string{"ID", "CreatedAt", "CreatedBy", "UpdatedAt", "UpdatedBy"}
	for _, field := range expectedFields {
		if !DefaultSkipFields[field] {
			t.Errorf("expected DefaultSkipFields[%q] to be true", field)
		}
	}
}
