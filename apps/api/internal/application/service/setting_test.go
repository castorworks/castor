package service

import (
	"context"
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/setting"
	"github.com/castorworks/castor/internal/domain/shared"
)

// ============================================
// Mock setting repository
// ============================================

type mockSettingRepoForSetting struct {
	settings map[uint]*setting.Setting
	keyMap   map[string]*setting.Setting
	nextID   uint
}

func newMockSettingRepoForSetting() *mockSettingRepoForSetting {
	return &mockSettingRepoForSetting{
		settings: make(map[uint]*setting.Setting),
		keyMap:   make(map[string]*setting.Setting),
		nextID:   1,
	}
}

func (m *mockSettingRepoForSetting) Gets(_ context.Context, category string) ([]setting.Setting, error) {
	var result []setting.Setting
	for _, s := range m.settings {
		if category == "" || string(s.Category) == category {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockSettingRepoForSetting) Get(_ context.Context, id uint) (*setting.Setting, error) {
	if s, ok := m.settings[id]; ok {
		return s, nil
	}
	return nil, shared.ErrNotFound
}

func (m *mockSettingRepoForSetting) GetByKey(_ context.Context, key string) (*setting.Setting, error) {
	if s, ok := m.keyMap[key]; ok {
		return s, nil
	}
	return nil, shared.ErrNotFound
}

func (m *mockSettingRepoForSetting) Create(_ context.Context, s *setting.Setting) error {
	if _, ok := m.keyMap[s.Key]; ok {
		return ErrSettingKeyExists
	}
	s.ID = m.nextID
	m.nextID++
	m.settings[s.ID] = s
	m.keyMap[s.Key] = s
	return nil
}

func (m *mockSettingRepoForSetting) Update(_ context.Context, s *setting.Setting) error {
	m.settings[s.ID] = s
	m.keyMap[s.Key] = s
	return nil
}

func (m *mockSettingRepoForSetting) Delete(_ context.Context, id uint) error {
	if s, ok := m.settings[id]; ok {
		delete(m.keyMap, s.Key)
		delete(m.settings, id)
		return nil
	}
	return shared.ErrNotFound
}

func (m *mockSettingRepoForSetting) GetByKeys(_ context.Context, keys []string) ([]setting.Setting, error) {
	var result []setting.Setting
	for _, key := range keys {
		if s, ok := m.keyMap[key]; ok {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockSettingRepoForSetting) BatchUpdate(_ context.Context, items []setting.Setting) error {
	for _, item := range items {
		if existing, ok := m.keyMap[item.Key]; ok {
			existing.Value = item.Value
		}
	}
	return nil
}

func (m *mockSettingRepoForSetting) GetPublicSettings(_ context.Context) ([]setting.Setting, error) {
	var result []setting.Setting
	for _, s := range m.settings {
		if s.IsPublic {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockSettingRepoForSetting) GetByCategory(_ context.Context, cat setting.SettingCategory) ([]setting.Setting, error) {
	return m.Gets(context.Background(), string(cat))
}

func (m *mockSettingRepoForSetting) ExistsByKey(_ context.Context, key string) (bool, error) {
	_, ok := m.keyMap[key]
	return ok, nil
}

// ============================================
// Tests
// ============================================

func TestSettingService_Post(t *testing.T) {
	t.Run("creates new setting", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		svc := NewSettingService(repo)

		req := &dto.SettingPostReq{
			Key:      "test.key",
			Value:    "test-value",
			Type:     "STRING",
			Category: "GENERAL",
			Name:     "Test Key",
		}

		resp, err := svc.Post(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Key != "test.key" {
			t.Errorf("expected key 'test.key', got %q", resp.Key)
		}
		if resp.Value != "test-value" {
			t.Errorf("expected value 'test-value', got %q", resp.Value)
		}
	})

	t.Run("duplicate key returns error", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.keyMap["dup.key"] = &setting.Setting{Key: "dup.key", Value: "old"}
		svc := NewSettingService(repo)

		req := &dto.SettingPostReq{
			Key:   "dup.key",
			Value: "new",
		}

		_, err := svc.Post(context.Background(), req)
		if err != ErrSettingKeyExists {
			t.Errorf("expected ErrSettingKeyExists, got %v", err)
		}
	})

	t.Run("invalid value for declared type returns error", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		svc := NewSettingService(repo)

		req := &dto.SettingPostReq{
			Key:   "num.setting",
			Value: "not-a-number",
			Type:  "NUMBER",
			Name:  "Num Setting",
		}

		_, err := svc.Post(context.Background(), req)
		if err != ErrInvalidSettingValue {
			t.Errorf("expected ErrInvalidSettingValue, got %v", err)
		}
	})

	t.Run("defaults to STRING type when empty", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		svc := NewSettingService(repo)

		req := &dto.SettingPostReq{
			Key:   "no.type",
			Value: "anything",
			Name:  "No Type",
		}

		resp, err := svc.Post(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Type != string(setting.SettingTypeString) {
			t.Errorf("expected type STRING, got %q", resp.Type)
		}
	})
}

func TestSettingService_Put(t *testing.T) {
	t.Run("updates existing setting", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.settings[1] = &setting.Setting{
			ID:    1,
			Key:   "existing.key",
			Value: "old-value",
			Name:  "Old Name",
			Type:  setting.SettingTypeString,
		}
		repo.keyMap["existing.key"] = repo.settings[1]
		svc := NewSettingService(repo)

		newVal := "new-value"
		req := &dto.SettingPutReq{Value: &newVal}

		resp, err := svc.Put(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Value != "new-value" {
			t.Errorf("expected value 'new-value', got %q", resp.Value)
		}
	})

	t.Run("allows setting value to empty string", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.settings[1] = &setting.Setting{
			ID:    1,
			Key:   "site.logo",
			Value: "/logo.png",
			Type:  setting.SettingTypeString,
		}
		repo.keyMap["site.logo"] = repo.settings[1]
		svc := NewSettingService(repo)

		emptyVal := ""
		req := &dto.SettingPutReq{Value: &emptyVal}

		resp, err := svc.Put(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Value != "" {
			t.Errorf("expected empty value, got %q", resp.Value)
		}
	})

	t.Run("not found returns error", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		svc := NewSettingService(repo)

		val := "new"
		req := &dto.SettingPutReq{Value: &val}
		_, err := svc.Put(context.Background(), 999, req)
		if err == nil {
			t.Error("expected error for non-existent setting, got nil")
		}
	})

	t.Run("nil value does not overwrite", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.settings[1] = &setting.Setting{
			ID:    1,
			Key:   "test.key",
			Value: "keep-me",
			Type:  setting.SettingTypeString,
		}
		repo.keyMap["test.key"] = repo.settings[1]
		svc := NewSettingService(repo)

		req := &dto.SettingPutReq{} // nil Value
		resp, err := svc.Put(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Value != "keep-me" {
			t.Errorf("expected value unchanged, got %q", resp.Value)
		}
	})

	t.Run("invalid value for number type returns error", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.settings[1] = &setting.Setting{
			ID:   1,
			Key:  "num.key",
			Type: setting.SettingTypeNumber,
		}
		repo.keyMap["num.key"] = repo.settings[1]
		svc := NewSettingService(repo)

		badVal := "not-a-number"
		req := &dto.SettingPutReq{Value: &badVal}
		_, err := svc.Put(context.Background(), 1, req)
		if err != ErrInvalidSettingValue {
			t.Errorf("expected ErrInvalidSettingValue, got %v", err)
		}
	})

	t.Run("login methods setting rejects empty array", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.settings[1] = &setting.Setting{
			ID:    1,
			Key:   setting.KeySecurityLoginAllowedMethods,
			Value: `["password"]`,
			Type:  setting.SettingTypeArray,
		}
		repo.keyMap[setting.KeySecurityLoginAllowedMethods] = repo.settings[1]
		svc := NewSettingService(repo)

		emptyMethods := `[]`
		_, err := svc.Put(context.Background(), 1, &dto.SettingPutReq{Value: &emptyMethods})
		if !errors.Is(err, ErrInvalidLoginMethodSetting) {
			t.Fatalf("expected ErrInvalidLoginMethodSetting, got %v", err)
		}
	})

	t.Run("login methods setting rejects unsupported method", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.settings[1] = &setting.Setting{
			ID:    1,
			Key:   setting.KeySecurityLoginAllowedMethods,
			Value: `["password"]`,
			Type:  setting.SettingTypeArray,
		}
		repo.keyMap[setting.KeySecurityLoginAllowedMethods] = repo.settings[1]
		svc := NewSettingService(repo)

		unsupportedMethods := `["password","oauth"]`
		_, err := svc.Put(context.Background(), 1, &dto.SettingPutReq{Value: &unsupportedMethods})
		if !errors.Is(err, ErrInvalidLoginMethodSetting) {
			t.Fatalf("expected ErrInvalidLoginMethodSetting, got %v", err)
		}
	})

	t.Run("login methods setting normalizes duplicates in fixed order", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.settings[1] = &setting.Setting{
			ID:    1,
			Key:   setting.KeySecurityLoginAllowedMethods,
			Value: `["password"]`,
			Type:  setting.SettingTypeArray,
		}
		repo.keyMap[setting.KeySecurityLoginAllowedMethods] = repo.settings[1]
		svc := NewSettingService(repo)

		methods := `["mobile","password","mobile"]`
		resp, err := svc.Put(context.Background(), 1, &dto.SettingPutReq{Value: &methods})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Value != `["password","mobile"]` {
			t.Fatalf("expected normalized methods, got %s", resp.Value)
		}
		if repo.keyMap[setting.KeySecurityLoginAllowedMethods].Value != `["password","mobile"]` {
			t.Fatalf("expected repository value to be normalized, got %s", repo.keyMap[setting.KeySecurityLoginAllowedMethods].Value)
		}
	})
}

func TestSettingService_Get(t *testing.T) {
	t.Run("returns setting by ID", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.settings[1] = &setting.Setting{ID: 1, Key: "my.key", Value: "v"}
		repo.keyMap["my.key"] = repo.settings[1]
		svc := NewSettingService(repo)

		resp, err := svc.Get(context.Background(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Key != "my.key" {
			t.Errorf("expected key 'my.key', got %q", resp.Key)
		}
	})
}

func TestSettingService_GetByKey(t *testing.T) {
	t.Run("returns setting by key", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.keyMap["look.up"] = &setting.Setting{Key: "look.up", Value: "found"}
		svc := NewSettingService(repo)

		resp, err := svc.GetByKey(context.Background(), "look.up")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Value != "found" {
			t.Errorf("expected 'found', got %q", resp.Value)
		}
	})
}

func TestSettingService_Delete(t *testing.T) {
	t.Run("deletes existing setting", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.settings[1] = &setting.Setting{ID: 1, Key: "del.key", Value: "x", IsSystem: false}
		repo.keyMap["del.key"] = repo.settings[1]
		svc := NewSettingService(repo)

		err := svc.Delete(context.Background(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_, err = repo.Get(context.Background(), 1)
		if err == nil {
			t.Error("expected setting to be deleted")
		}
	})

	t.Run("system setting cannot be deleted", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.settings[1] = &setting.Setting{ID: 1, Key: "site.name", Value: "Castor", IsSystem: true}
		repo.keyMap["site.name"] = repo.settings[1]
		svc := NewSettingService(repo)

		err := svc.Delete(context.Background(), 1)
		if err != ErrSystemSettingDelete {
			t.Errorf("expected ErrSystemSettingDelete, got %v", err)
		}
		// 确认未被删除
		_, err = repo.Get(context.Background(), 1)
		if err != nil {
			t.Error("system setting should not have been deleted")
		}
	})
}

func TestSettingService_Gets(t *testing.T) {
	t.Run("returns all settings when no category filter", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.keyMap["a"] = &setting.Setting{ID: 1, Key: "a", Value: "1", Category: setting.CategoryGeneral}
		repo.settings[1] = repo.keyMap["a"]
		repo.keyMap["b"] = &setting.Setting{ID: 2, Key: "b", Value: "2", Category: setting.CategorySecurity}
		repo.settings[2] = repo.keyMap["b"]
		svc := NewSettingService(repo)

		resp, err := svc.Gets(context.Background(), "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp) != 2 {
			t.Errorf("expected 2 settings, got %d", len(resp))
		}
	})
}

func TestSettingService_BatchUpdate(t *testing.T) {
	t.Run("updates multiple settings", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.keyMap["k1"] = &setting.Setting{Key: "k1", Value: "old1"}
		repo.keyMap["k2"] = &setting.Setting{Key: "k2", Value: "old2"}
		svc := NewSettingService(repo)

		req := &dto.SettingBatchUpdateReq{
			Settings: []dto.SettingUpdateItem{
				{Key: "k1", Value: "new1"},
				{Key: "k2", Value: "new2"},
			},
		}

		err := svc.BatchUpdate(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.keyMap["k1"].Value != "new1" {
			t.Errorf("expected k1='new1', got %q", repo.keyMap["k1"].Value)
		}
		if repo.keyMap["k2"].Value != "new2" {
			t.Errorf("expected k2='new2', got %q", repo.keyMap["k2"].Value)
		}
	})

	t.Run("rejects invalid login methods setting", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.keyMap[setting.KeySecurityLoginAllowedMethods] = &setting.Setting{
			Key:   setting.KeySecurityLoginAllowedMethods,
			Value: `["password"]`,
			Type:  setting.SettingTypeArray,
		}
		svc := NewSettingService(repo)

		err := svc.BatchUpdate(context.Background(), &dto.SettingBatchUpdateReq{
			Settings: []dto.SettingUpdateItem{
				{Key: setting.KeySecurityLoginAllowedMethods, Value: `["email","sso"]`},
			},
		})
		if !errors.Is(err, ErrInvalidLoginMethodSetting) {
			t.Fatalf("expected ErrInvalidLoginMethodSetting, got %v", err)
		}
	})

	t.Run("normalizes login methods setting", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.keyMap[setting.KeySecurityLoginAllowedMethods] = &setting.Setting{
			Key:   setting.KeySecurityLoginAllowedMethods,
			Value: `["password"]`,
			Type:  setting.SettingTypeArray,
		}
		svc := NewSettingService(repo)

		err := svc.BatchUpdate(context.Background(), &dto.SettingBatchUpdateReq{
			Settings: []dto.SettingUpdateItem{
				{Key: setting.KeySecurityLoginAllowedMethods, Value: `["mobile","email","email"]`},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.keyMap[setting.KeySecurityLoginAllowedMethods].Value != `["email","mobile"]` {
			t.Fatalf("expected normalized value, got %s", repo.keyMap[setting.KeySecurityLoginAllowedMethods].Value)
		}
	})
}

func TestSettingService_GetPublicSettings(t *testing.T) {
	t.Run("returns only public settings with type-aware parsing", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.keyMap["site.name"] = &setting.Setting{
			Key: "site.name", Value: "MySite", Type: setting.SettingTypeString, IsPublic: true,
		}
		repo.settings[1] = repo.keyMap["site.name"]
		repo.keyMap["feature.on"] = &setting.Setting{
			Key: "feature.on", Value: "true", Type: setting.SettingTypeBool, IsPublic: true,
		}
		repo.settings[2] = repo.keyMap["feature.on"]
		repo.keyMap["count"] = &setting.Setting{
			Key: "count", Value: "42", Type: setting.SettingTypeNumber, IsPublic: true,
		}
		repo.settings[3] = repo.keyMap["count"]
		repo.keyMap["secret"] = &setting.Setting{
			Key: "secret", Value: "hidden", Type: setting.SettingTypeString, IsPublic: false,
		}
		repo.settings[4] = repo.keyMap["secret"]
		svc := NewSettingService(repo)

		result, err := svc.GetPublicSettings(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result["site.name"] != "MySite" {
			t.Errorf("expected site.name='MySite', got %v", result["site.name"])
		}
		if result["feature.on"] != true {
			t.Errorf("expected feature.on=true, got %v", result["feature.on"])
		}
		if result["count"] != 42 {
			t.Errorf("expected count=42, got %v", result["count"])
		}
		if _, ok := result["secret"]; ok {
			t.Error("expected secret key to be excluded")
		}
	})
}

func TestSettingService_GetValue(t *testing.T) {
	t.Run("returns value by key", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.keyMap["my.key"] = &setting.Setting{Key: "my.key", Value: "the-value"}
		svc := NewSettingService(repo)

		val, err := svc.GetValue(context.Background(), "my.key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "the-value" {
			t.Errorf("expected 'the-value', got %q", val)
		}
	})
}

func TestSettingService_GetBool(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"true string", "true", true},
		{"1 string", "1", true},
		{"false string", "false", false},
		{"0 string", "0", false},
		{"other string", "anything", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockSettingRepoForSetting()
			repo.keyMap["bool.key"] = &setting.Setting{Key: "bool.key", Value: tt.value}
			svc := NewSettingService(repo)

			result, err := svc.GetBool(context.Background(), "bool.key")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSettingService_GetInt(t *testing.T) {
	t.Run("returns integer value", func(t *testing.T) {
		repo := newMockSettingRepoForSetting()
		repo.keyMap["num.key"] = &setting.Setting{Key: "num.key", Value: "123"}
		svc := NewSettingService(repo)

		result, err := svc.GetInt(context.Background(), "num.key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != 123 {
			t.Errorf("expected 123, got %d", result)
		}
	})
}

func TestParseSettingValue(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		settingType setting.SettingType
		expected    interface{}
	}{
		{name: "bool true", value: "true", settingType: setting.SettingTypeBool, expected: true},
		{name: "bool 1", value: "1", settingType: setting.SettingTypeBool, expected: true},
		{name: "bool false", value: "false", settingType: setting.SettingTypeBool, expected: false},
		{name: "number int", value: "42", settingType: setting.SettingTypeNumber, expected: 42},
		{name: "number float", value: "3.14", settingType: setting.SettingTypeNumber, expected: 3.14},
		{name: "string type", value: "hello", settingType: setting.SettingTypeString, expected: "hello"},
		{name: "json type", value: `{"k":"v"}`, settingType: setting.SettingTypeJSON, expected: `{"k":"v"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSettingValue(tt.value, tt.settingType)
			if result != tt.expected {
				t.Errorf("parseSettingValue(%q, %q) = %v (type %T), want %v (type %T)",
					tt.value, tt.settingType, result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestValidateSettingValue(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		settingType setting.SettingType
		wantErr     bool
	}{
		// BOOL
		{name: "bool true", value: "true", settingType: setting.SettingTypeBool, wantErr: false},
		{name: "bool false", value: "false", settingType: setting.SettingTypeBool, wantErr: false},
		{name: "bool 1", value: "1", settingType: setting.SettingTypeBool, wantErr: false},
		{name: "bool 0", value: "0", settingType: setting.SettingTypeBool, wantErr: false},
		{name: "bool invalid", value: "yes", settingType: setting.SettingTypeBool, wantErr: true},
		// NUMBER
		{name: "number int", value: "42", settingType: setting.SettingTypeNumber, wantErr: false},
		{name: "number float", value: "3.14", settingType: setting.SettingTypeNumber, wantErr: false},
		{name: "number negative", value: "-10", settingType: setting.SettingTypeNumber, wantErr: false},
		{name: "number invalid", value: "abc", settingType: setting.SettingTypeNumber, wantErr: true},
		// JSON
		{name: "json object", value: `{"key":"val"}`, settingType: setting.SettingTypeJSON, wantErr: false},
		{name: "json array", value: `[1,2,3]`, settingType: setting.SettingTypeJSON, wantErr: false},
		{name: "json invalid", value: `{bad`, settingType: setting.SettingTypeJSON, wantErr: true},
		// ARRAY
		{name: "array valid", value: `["a","b"]`, settingType: setting.SettingTypeArray, wantErr: false},
		{name: "array not array", value: `{"k":"v"}`, settingType: setting.SettingTypeArray, wantErr: true},
		{name: "array invalid json", value: `[bad`, settingType: setting.SettingTypeArray, wantErr: true},
		// STRING
		{name: "string anything", value: "anything goes", settingType: setting.SettingTypeString, wantErr: false},
		// SECRET
		{name: "secret anything", value: "s3cr3t", settingType: setting.SettingTypeSecret, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSettingValue(tt.value, tt.settingType)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSettingValue(%q, %q) error = %v, wantErr %v",
					tt.value, tt.settingType, err, tt.wantErr)
			}
			if err != nil && err != ErrInvalidSettingValue {
				t.Errorf("expected ErrInvalidSettingValue, got %v", err)
			}
		})
	}
}
