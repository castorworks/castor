package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/setting"
)

func TestPasswordPolicyValidate(t *testing.T) {
	t.Parallel()
	simple := PasswordPolicy{MinLength: 8}
	complex := PasswordPolicy{MinLength: 8, RequireComplexity: true}
	tests := []struct {
		name     string
		policy   PasswordPolicy
		password string
		want     error
	}{
		{"meets length", simple, "abcdefgh", nil},
		{"too short", simple, "abcdefg", apperror.ErrPasswordTooShort},
		{"empty with zero minimum", PasswordPolicy{}, "", apperror.ErrPasswordTooShort},
		// 长度按字符而不是字节计：8 个汉字是 24 字节，但只有 8 个字符。
		{"counts characters not bytes", simple, "密码密码密码密码", nil},
		{"seven multibyte characters are too short", simple, "密码密码密码密", apperror.ErrPasswordTooShort},
		{"bcrypt limit", simple, strings.Repeat("a", MaxPasswordBytes), nil},
		{"over bcrypt limit", simple, strings.Repeat("a", MaxPasswordBytes+1), apperror.ErrPasswordTooLong},
		{"two classes are too weak", complex, "password123", apperror.ErrPasswordTooWeak},
		{"three classes", complex, "Password123", nil},
		{"symbols count as a class", complex, "password-123", nil},
		{"spaces are not a class", complex, "pass word 123", apperror.ErrPasswordTooWeak},
		{"length checked before complexity", complex, "Ab1", apperror.ErrPasswordTooShort},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.policy.Validate(tt.password); !errors.Is(err, tt.want) || (tt.want == nil && err != nil) {
				t.Fatalf("Validate(%q) = %v, want %v", tt.password, err, tt.want)
			}
		})
	}
}

func TestPasswordPolicyCredentialExpiry(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)
	if got := (PasswordPolicy{}).CredentialExpiry(now); !got.IsZero() {
		t.Fatalf("no max age must mean never expires, got %v", got)
	}
	if got, want := (PasswordPolicy{MaxAgeDays: 90}).CredentialExpiry(now), now.AddDate(0, 0, 90); !got.Equal(want) {
		t.Fatalf("CredentialExpiry = %v, want %v", got, want)
	}
}

func TestSettingHelperPasswordPolicy(t *testing.T) {
	t.Parallel()
	set := func(repo *mockSettingRepo, key, value string) {
		repo.settings[key] = &setting.Setting{Key: key, Value: value}
	}

	t.Run("missing settings fall back to secure defaults", func(t *testing.T) {
		t.Parallel()
		got := NewSettingHelper(&mockSettingRepo{settings: map[string]*setting.Setting{}}).PasswordPolicy(context.Background())
		want := PasswordPolicy{MinLength: DefaultPasswordMinLength, RequireComplexity: true, MaxAgeDays: 0}
		if got != want {
			t.Fatalf("PasswordPolicy() = %+v, want %+v", got, want)
		}
	})
	t.Run("configured values", func(t *testing.T) {
		t.Parallel()
		repo := newMockSettingRepo()
		set(repo, setting.KeySecurityPasswordMinLength, "12")
		set(repo, setting.KeySecurityPasswordComplexity, "false")
		set(repo, setting.KeySecurityPasswordMaxAgeDays, "90")
		got := NewSettingHelper(repo).PasswordPolicy(context.Background())
		want := PasswordPolicy{MinLength: 12, RequireComplexity: false, MaxAgeDays: 90}
		if got != want {
			t.Fatalf("PasswordPolicy() = %+v, want %+v", got, want)
		}
	})
	t.Run("invalid values fall back", func(t *testing.T) {
		t.Parallel()
		repo := newMockSettingRepo()
		set(repo, setting.KeySecurityPasswordMinLength, "0")
		set(repo, setting.KeySecurityPasswordComplexity, "maybe")
		set(repo, setting.KeySecurityPasswordMaxAgeDays, "-5")
		got := NewSettingHelper(repo).PasswordPolicy(context.Background())
		want := PasswordPolicy{MinLength: DefaultPasswordMinLength, RequireComplexity: true, MaxAgeDays: 0}
		if got != want {
			t.Fatalf("PasswordPolicy() = %+v, want %+v", got, want)
		}
	})
}
