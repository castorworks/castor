package ucontext

import (
	"context"
	"testing"
)

func TestAuditContext(t *testing.T) {
	t.Run("GetAuditUserID from AuditContext", func(t *testing.T) {
		ac := NewAuditContext(context.Background(), 42)
		result := GetAuditUserID(ac)
		if result != 42 {
			t.Errorf("expected 42, got %d", result)
		}
	})

	t.Run("GetAuditUserID from non-AuditContext returns 0", func(t *testing.T) {
		result := GetAuditUserID(context.Background())
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})

	t.Run("GetAuditUserID from nil context returns 0", func(t *testing.T) {
		var ctx context.Context
		result := GetAuditUserID(ctx)
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})
}

func TestWithAuditUserID(t *testing.T) {
	t.Run("stores and retrieves user ID via AuditUserIDFromContext", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithAuditUserID(ctx, 123)
		result := AuditUserIDFromContext(ctx)
		if result != 123 {
			t.Errorf("expected 123, got %d", result)
		}
	})

	t.Run("overwrites previous value", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithAuditUserID(ctx, 1)
		ctx = WithAuditUserID(ctx, 2)
		result := AuditUserIDFromContext(ctx)
		if result != 2 {
			t.Errorf("expected 2, got %d", result)
		}
	})

	t.Run("AuditContext takes precedence over context.Value", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithAuditUserID(ctx, 1)
		ac := NewAuditContext(ctx, 99)
		result := AuditUserIDFromContext(ac)
		if result != 99 {
			t.Errorf("expected 99 (from AuditContext), got %d", result)
		}
	})
}

func TestAuditUserIDFromContext_ZeroCases(t *testing.T) {
	t.Run("empty context returns 0", func(t *testing.T) {
		result := AuditUserIDFromContext(context.Background())
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})

	t.Run("nil context returns 0", func(t *testing.T) {
		var ctx context.Context
		result := AuditUserIDFromContext(ctx)
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})
}
