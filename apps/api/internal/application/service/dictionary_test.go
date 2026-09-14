package service

import (
	"context"
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/dictionary"
	"github.com/castorworks/castor/internal/domain/shared"
)

// ============================================
// Mock repositories for dictionary service
// ============================================

type mockDictTypeRepo struct {
	types   map[uint]*dictionary.DictType
	codeMap map[string]*dictionary.DictType
	nextID  uint
}

func newMockDictTypeRepo() *mockDictTypeRepo {
	return &mockDictTypeRepo{
		types:   make(map[uint]*dictionary.DictType),
		codeMap: make(map[string]*dictionary.DictType),
		nextID:  1,
	}
}

func (m *mockDictTypeRepo) Gets(_ context.Context) ([]dictionary.DictType, error) {
	var result []dictionary.DictType
	for _, t := range m.types {
		result = append(result, *t)
	}
	return result, nil
}

func (m *mockDictTypeRepo) Get(_ context.Context, id uint) (*dictionary.DictType, error) {
	if t, ok := m.types[id]; ok {
		return t, nil
	}
	return nil, shared.ErrNotFound
}

func (m *mockDictTypeRepo) GetByCode(_ context.Context, code string) (*dictionary.DictType, error) {
	if t, ok := m.codeMap[code]; ok {
		return t, nil
	}
	return nil, shared.ErrNotFound
}

func (m *mockDictTypeRepo) Create(_ context.Context, t *dictionary.DictType) error {
	t.ID = m.nextID
	m.nextID++
	m.types[t.ID] = t
	m.codeMap[t.Code] = t
	return nil
}

func (m *mockDictTypeRepo) Update(_ context.Context, t *dictionary.DictType) error {
	m.types[t.ID] = t
	m.codeMap[t.Code] = t
	return nil
}

func (m *mockDictTypeRepo) Delete(_ context.Context, id uint) error {
	if t, ok := m.types[id]; ok {
		delete(m.codeMap, t.Code)
		delete(m.types, id)
		return nil
	}
	return shared.ErrNotFound
}

func (m *mockDictTypeRepo) ExistsByCode(_ context.Context, code string) (bool, error) {
	_, ok := m.codeMap[code]
	return ok, nil
}

type mockDictItemRepo struct {
	items  map[uint]*dictionary.DictItem
	nextID uint
}

func newMockDictItemRepo() *mockDictItemRepo {
	return &mockDictItemRepo{
		items:  make(map[uint]*dictionary.DictItem),
		nextID: 1,
	}
}

func (m *mockDictItemRepo) Gets(_ context.Context, typeCode string) ([]dictionary.DictItem, error) {
	var result []dictionary.DictItem
	for _, item := range m.items {
		if typeCode == "" || item.TypeCode == typeCode {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (m *mockDictItemRepo) Get(_ context.Context, id uint) (*dictionary.DictItem, error) {
	if item, ok := m.items[id]; ok {
		return item, nil
	}
	return nil, shared.ErrNotFound
}

func (m *mockDictItemRepo) GetByTypeCode(_ context.Context, typeCode string) ([]dictionary.DictItem, error) {
	return m.Gets(context.Background(), typeCode)
}

func (m *mockDictItemRepo) GetEnabledByTypeCode(_ context.Context, typeCode string) ([]dictionary.DictItem, error) {
	var result []dictionary.DictItem
	for _, item := range m.items {
		if item.TypeCode == typeCode && item.IsEnabled {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (m *mockDictItemRepo) Create(_ context.Context, item *dictionary.DictItem) error {
	item.ID = m.nextID
	m.nextID++
	m.items[item.ID] = item
	return nil
}

func (m *mockDictItemRepo) Update(_ context.Context, item *dictionary.DictItem) error {
	m.items[item.ID] = item
	return nil
}

func (m *mockDictItemRepo) Delete(_ context.Context, id uint) error {
	delete(m.items, id)
	return nil
}

func (m *mockDictItemRepo) DeleteByTypeCode(_ context.Context, typeCode string) error {
	for id, item := range m.items {
		if item.TypeCode == typeCode {
			delete(m.items, id)
		}
	}
	return nil
}

func (m *mockDictItemRepo) BatchCreate(_ context.Context, items []dictionary.DictItem) error {
	for i := range items {
		items[i].ID = m.nextID
		m.nextID++
		m.items[items[i].ID] = &items[i]
	}
	return nil
}

func (m *mockDictItemRepo) ExistsByTypeCodeAndValue(_ context.Context, typeCode, value string, excludeID uint) (bool, error) {
	for _, item := range m.items {
		if item.TypeCode == typeCode && item.Value == value && item.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockDictItemRepo) GetAllEnabled(_ context.Context) ([]dictionary.DictItem, error) {
	var result []dictionary.DictItem
	for _, item := range m.items {
		if item.IsEnabled {
			result = append(result, *item)
		}
	}
	return result, nil
}

// ============================================
// Tests
// ============================================

func TestDictionaryService_PostType(t *testing.T) {
	t.Run("creates new dict type", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		itemRepo := newMockDictItemRepo()
		svc := NewDictionaryService(typeRepo, itemRepo)

		req := &dto.DictTypePostReq{
			Code:        "color",
			Name:        "Color",
			Description: "Color options",
		}

		resp, err := svc.PostType(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Code != "color" {
			t.Errorf("expected code 'color', got %q", resp.Code)
		}
		if resp.IsEnabled != true {
			t.Error("expected IsEnabled to be true by default")
		}
	})

	t.Run("duplicate code returns error", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		typeRepo.codeMap["existing"] = &dictionary.DictType{Code: "existing"}
		itemRepo := newMockDictItemRepo()
		svc := NewDictionaryService(typeRepo, itemRepo)

		req := &dto.DictTypePostReq{Code: "existing", Name: "Dup"}
		_, err := svc.PostType(context.Background(), req)
		if err != ErrDictTypeCodeExists {
			t.Errorf("expected ErrDictTypeCodeExists, got %v", err)
		}
	})
}

func TestDictionaryService_PutType(t *testing.T) {
	t.Run("updates existing type", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		typeRepo.types[1] = &dictionary.DictType{
			ID: 1, Code: "status", Name: "Status", Description: "Old desc",
		}
		typeRepo.codeMap["status"] = typeRepo.types[1]
		itemRepo := newMockDictItemRepo()
		svc := NewDictionaryService(typeRepo, itemRepo)

		newName := "Updated Status"
		req := &dto.DictTypePutReq{Name: &newName}
		resp, err := svc.PutType(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Name != newName {
			t.Errorf("expected name %q, got %q", newName, resp.Name)
		}
	})

	t.Run("empty fields do not overwrite", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		typeRepo.types[1] = &dictionary.DictType{
			ID: 1, Code: "test", Name: "Keep Me",
		}
		typeRepo.codeMap["test"] = typeRepo.types[1]
		itemRepo := newMockDictItemRepo()
		svc := NewDictionaryService(typeRepo, itemRepo)

		req := &dto.DictTypePutReq{}
		resp, err := svc.PutType(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Name != "Keep Me" {
			t.Errorf("expected name unchanged, got %q", resp.Name)
		}
	})

	t.Run("can clear description", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		typeRepo.types[1] = &dictionary.DictType{
			ID: 1, Code: "test", Name: "Test", Description: "Has desc",
		}
		typeRepo.codeMap["test"] = typeRepo.types[1]
		itemRepo := newMockDictItemRepo()
		svc := NewDictionaryService(typeRepo, itemRepo)

		emptyDesc := ""
		req := &dto.DictTypePutReq{Description: &emptyDesc}
		resp, err := svc.PutType(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Description != "" {
			t.Errorf("expected description cleared, got %q", resp.Description)
		}
	})
}

func TestDictionaryService_DeleteType(t *testing.T) {
	t.Run("deletes type and its items", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		typeRepo.types[1] = &dictionary.DictType{
			ID: 1, Code: "temp", Name: "Temporary",
		}
		typeRepo.codeMap["temp"] = typeRepo.types[1]
		itemRepo := newMockDictItemRepo()
		itemRepo.items[1] = &dictionary.DictItem{ID: 1, TypeCode: "temp", Label: "Item1"}
		svc := NewDictionaryService(typeRepo, itemRepo)

		err := svc.DeleteType(context.Background(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Verify type is deleted
		_, err = typeRepo.Get(context.Background(), 1)
		if err == nil {
			t.Error("expected type to be deleted")
		}
		// Verify items are deleted
		if len(itemRepo.items) != 0 {
			t.Error("expected items to be deleted with type")
		}
	})

	t.Run("rejects system type without touching its items", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		typeRepo.types[1] = &dictionary.DictType{ID: 1, Code: "gender", IsSystem: true}
		typeRepo.codeMap["gender"] = typeRepo.types[1]
		itemRepo := newMockDictItemRepo()
		itemRepo.items[1] = &dictionary.DictItem{ID: 1, TypeCode: "gender", Label: "Male"}
		svc := NewDictionaryService(typeRepo, itemRepo)

		err := svc.DeleteType(context.Background(), 1)
		if !errors.Is(err, apperror.ErrSystemDictTypeDelete) {
			t.Fatalf("expected ErrSystemDictTypeDelete, got %v", err)
		}
		if _, ok := typeRepo.types[1]; !ok {
			t.Error("system type must not be deleted")
		}
		if len(itemRepo.items) != 1 {
			t.Error("items of a system type must not be deleted")
		}
	})
}

func TestDictionaryService_GetTypes(t *testing.T) {
	t.Run("returns all types", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		typeRepo.types[1] = &dictionary.DictType{ID: 1, Code: "a"}
		typeRepo.types[2] = &dictionary.DictType{ID: 2, Code: "b"}
		typeRepo.codeMap["a"] = typeRepo.types[1]
		typeRepo.codeMap["b"] = typeRepo.types[2]
		itemRepo := newMockDictItemRepo()
		svc := NewDictionaryService(typeRepo, itemRepo)

		resp, err := svc.GetTypes(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp) != 2 {
			t.Errorf("expected 2 types, got %d", len(resp))
		}
	})
}

func TestDictionaryService_GetTypeByCode(t *testing.T) {
	t.Run("returns type by code", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		typeRepo.codeMap["gender"] = &dictionary.DictType{Code: "gender", Name: "Gender"}
		itemRepo := newMockDictItemRepo()
		svc := NewDictionaryService(typeRepo, itemRepo)

		resp, err := svc.GetTypeByCode(context.Background(), "gender")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Name != "Gender" {
			t.Errorf("expected 'Gender', got %q", resp.Name)
		}
	})
}

func TestDictionaryService_PostItem(t *testing.T) {
	t.Run("creates dict item", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		itemRepo := newMockDictItemRepo()
		svc := NewDictionaryService(typeRepo, itemRepo)

		req := &dto.DictItemPostReq{
			TypeCode: "gender",
			Label:    "Male",
			Value:    "MALE",
		}

		resp, err := svc.PostItem(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Label != "Male" {
			t.Errorf("expected label 'Male', got %q", resp.Label)
		}
		if resp.IsEnabled != true {
			t.Error("expected IsEnabled to be true by default")
		}
	})
}

func TestDictionaryService_PutItem(t *testing.T) {
	t.Run("updates existing item", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		itemRepo := newMockDictItemRepo()
		itemRepo.items[1] = &dictionary.DictItem{
			ID: 1, TypeCode: "status", Label: "Old Label", Value: "OLD",
		}
		svc := NewDictionaryService(typeRepo, itemRepo)

		newLabel := "New Label"
		req := &dto.DictItemPutReq{Label: &newLabel}
		resp, err := svc.PutItem(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Label != "New Label" {
			t.Errorf("expected label 'New Label', got %q", resp.Label)
		}
	})

	t.Run("duplicate value returns error", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		itemRepo := newMockDictItemRepo()
		itemRepo.items[1] = &dictionary.DictItem{
			ID: 1, TypeCode: "status", Label: "Active", Value: "ACTIVE",
		}
		itemRepo.items[2] = &dictionary.DictItem{
			ID: 2, TypeCode: "status", Label: "Pending", Value: "PENDING",
		}
		svc := NewDictionaryService(typeRepo, itemRepo)

		dupValue := "ACTIVE"
		req := &dto.DictItemPutReq{Value: &dupValue}
		_, err := svc.PutItem(context.Background(), 2, req)
		if err != ErrDictItemValueExists {
			t.Errorf("expected ErrDictItemValueExists, got %v", err)
		}
	})
}

func TestDictionaryService_GetAllPublicDicts(t *testing.T) {
	t.Run("returns map of enabled type codes to items", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		typeRepo.types[1] = &dictionary.DictType{ID: 1, Code: "gender", IsEnabled: true}
		typeRepo.types[2] = &dictionary.DictType{ID: 2, Code: "disabled_type", IsEnabled: false}
		typeRepo.codeMap["gender"] = typeRepo.types[1]
		typeRepo.codeMap["disabled_type"] = typeRepo.types[2]

		itemRepo := newMockDictItemRepo()
		itemRepo.items[1] = &dictionary.DictItem{ID: 1, TypeCode: "gender", Label: "Male", IsEnabled: true}

		svc := NewDictionaryService(typeRepo, itemRepo)

		result, err := svc.GetAllPublicDicts(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := result["gender"]; !ok {
			t.Error("expected 'gender' key in result")
		}
		if _, ok := result["disabled_type"]; ok {
			t.Error("expected disabled_type to be excluded")
		}
	})
}
