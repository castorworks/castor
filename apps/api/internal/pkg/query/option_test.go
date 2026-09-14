package query

import (
	"testing"
)

func TestNewOption(t *testing.T) {
	t.Run("with no args", func(t *testing.T) {
		opt := NewOption("status = 'active'")
		if opt.Condition != "status = 'active'" {
			t.Errorf("expected condition 'status = active', got %q", opt.Condition)
		}
		if len(opt.Args) != 0 {
			t.Errorf("expected no args, got %d", len(opt.Args))
		}
	})

	t.Run("with single arg", func(t *testing.T) {
		opt := NewOption("id = ?", 42)
		if opt.Condition != "id = ?" {
			t.Errorf("expected condition 'id = ?', got %q", opt.Condition)
		}
		if len(opt.Args) != 1 || opt.Args[0] != 42 {
			t.Errorf("expected args [42], got %v", opt.Args)
		}
	})

	t.Run("with multiple args", func(t *testing.T) {
		opt := NewOption("id = ? AND name = ?", 42, "test")
		if opt.Condition != "id = ? AND name = ?" {
			t.Errorf("unexpected condition: %q", opt.Condition)
		}
		if len(opt.Args) != 2 {
			t.Errorf("expected 2 args, got %d", len(opt.Args))
		}
		if opt.Args[0] != 42 || opt.Args[1] != "test" {
			t.Errorf("unexpected args: %v", opt.Args)
		}
	})

	t.Run("with nil arg", func(t *testing.T) {
		opt := NewOption("status IS NULL", nil)
		if len(opt.Args) != 1 || opt.Args[0] != nil {
			t.Errorf("expected [nil], got %v", opt.Args)
		}
	})
}
