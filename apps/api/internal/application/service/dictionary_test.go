package service

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/dictionary"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/redis/go-redis/v9"
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
		svc := NewDictionaryService(typeRepo, itemRepo, nil, "castor")

		req := &dto.DictTypePostReq{
			Code:        "color",
			Name:        dto.I18nTextReq{En: "Color", Zh: "Color", Ja: "Color", Ko: "Color"},
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
		svc := NewDictionaryService(typeRepo, itemRepo, nil, "castor")

		req := &dto.DictTypePostReq{Code: "existing", Name: dto.I18nTextReq{En: "Dup", Zh: "Dup", Ja: "Dup", Ko: "Dup"}}
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
			ID: 1, Code: "status", Name: shared.Text("Status", "Status", "Status", "Status"), Description: "Old desc",
		}
		typeRepo.codeMap["status"] = typeRepo.types[1]
		itemRepo := newMockDictItemRepo()
		svc := NewDictionaryService(typeRepo, itemRepo, nil, "castor")

		newName := dto.I18nTextReq{En: "Updated Status", Zh: "Updated Status", Ja: "Updated Status", Ko: "Updated Status"}
		req := &dto.DictTypePutReq{Name: &newName}
		resp, err := svc.PutType(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Name != newName.ToI18nText() {
			t.Errorf("expected name %v, got %v", newName, resp.Name)
		}
	})

	t.Run("empty fields do not overwrite", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		typeRepo.types[1] = &dictionary.DictType{
			ID: 1, Code: "test", Name: shared.Text("Keep Me", "Keep Me", "Keep Me", "Keep Me"),
		}
		typeRepo.codeMap["test"] = typeRepo.types[1]
		itemRepo := newMockDictItemRepo()
		svc := NewDictionaryService(typeRepo, itemRepo, nil, "castor")

		req := &dto.DictTypePutReq{}
		resp, err := svc.PutType(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Name != shared.Text("Keep Me", "Keep Me", "Keep Me", "Keep Me") {
			t.Errorf("expected name unchanged, got %v", resp.Name)
		}
	})

	t.Run("can clear description", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		typeRepo.types[1] = &dictionary.DictType{
			ID: 1, Code: "test", Name: shared.Text("Test", "Test", "Test", "Test"), Description: "Has desc",
		}
		typeRepo.codeMap["test"] = typeRepo.types[1]
		itemRepo := newMockDictItemRepo()
		svc := NewDictionaryService(typeRepo, itemRepo, nil, "castor")

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
			ID: 1, Code: "temp", Name: shared.Text("Temporary", "Temporary", "Temporary", "Temporary"),
		}
		typeRepo.codeMap["temp"] = typeRepo.types[1]
		itemRepo := newMockDictItemRepo()
		itemRepo.items[1] = &dictionary.DictItem{ID: 1, TypeCode: "temp", Label: shared.Text("Item1", "Item1", "Item1", "Item1")}
		svc := NewDictionaryService(typeRepo, itemRepo, nil, "castor")

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
		itemRepo.items[1] = &dictionary.DictItem{ID: 1, TypeCode: "gender", Label: shared.Text("Male", "Male", "Male", "Male")}
		svc := NewDictionaryService(typeRepo, itemRepo, nil, "castor")

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
		svc := NewDictionaryService(typeRepo, itemRepo, nil, "castor")

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
		typeRepo.codeMap["gender"] = &dictionary.DictType{Code: "gender", Name: shared.Text("Gender", "Gender", "Gender", "Gender")}
		itemRepo := newMockDictItemRepo()
		svc := NewDictionaryService(typeRepo, itemRepo, nil, "castor")

		resp, err := svc.GetTypeByCode(context.Background(), "gender")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Name != shared.Text("Gender", "Gender", "Gender", "Gender") {
			t.Errorf("expected 'Gender', got %v", resp.Name)
		}
	})
}

func TestDictionaryService_PostItem(t *testing.T) {
	t.Run("creates dict item", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		seedDictType(typeRepo, &dictionary.DictType{ID: 1, Code: "gender", IsEnabled: true})
		itemRepo := newMockDictItemRepo()
		svc := NewDictionaryService(typeRepo, itemRepo, nil, "castor")

		req := &dto.DictItemPostReq{
			TypeCode: "gender",
			Label:    dto.I18nTextReq{En: "Male", Zh: "Male", Ja: "Male", Ko: "Male"},
			Value:    "MALE",
		}

		resp, err := svc.PostItem(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Label != shared.Text("Male", "Male", "Male", "Male") {
			t.Errorf("expected label 'Male', got %v", resp.Label)
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
			ID: 1, TypeCode: "status", Label: shared.Text("Old Label", "Old Label", "Old Label", "Old Label"), Value: "OLD",
		}
		svc := NewDictionaryService(typeRepo, itemRepo, nil, "castor")

		newLabel := dto.I18nTextReq{En: "New Label", Zh: "New Label", Ja: "New Label", Ko: "New Label"}
		req := &dto.DictItemPutReq{Label: &newLabel}
		resp, err := svc.PutItem(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Label != shared.Text("New Label", "New Label", "New Label", "New Label") {
			t.Errorf("expected label 'New Label', got %v", resp.Label)
		}
	})

	t.Run("duplicate value returns error", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		itemRepo := newMockDictItemRepo()
		itemRepo.items[1] = &dictionary.DictItem{
			ID: 1, TypeCode: "status", Label: shared.Text("Active", "Active", "Active", "Active"), Value: "ACTIVE",
		}
		itemRepo.items[2] = &dictionary.DictItem{
			ID: 2, TypeCode: "status", Label: shared.Text("Pending", "Pending", "Pending", "Pending"), Value: "PENDING",
		}
		svc := NewDictionaryService(typeRepo, itemRepo, nil, "castor")

		dupValue := "ACTIVE"
		req := &dto.DictItemPutReq{Value: &dupValue}
		_, err := svc.PutItem(context.Background(), 2, req)
		if err != ErrDictItemValueExists {
			t.Errorf("expected ErrDictItemValueExists, got %v", err)
		}
	})
}

// seedDictType 同时登记到 mock 的两个索引里。
func seedDictType(repo *mockDictTypeRepo, t *dictionary.DictType) {
	repo.types[t.ID] = t
	repo.codeMap[t.Code] = t
}

func TestDictionaryService_PostItemRejectsInvalidInput(t *testing.T) {
	label := dto.I18nTextReq{En: "Draft", Zh: "草稿", Ja: "下書き", Ko: "임시"}

	t.Run("unknown type leaves no orphan item", func(t *testing.T) {
		itemRepo := newMockDictItemRepo()
		svc := NewDictionaryService(newMockDictTypeRepo(), itemRepo, nil, "castor")

		_, err := svc.PostItem(context.Background(), &dto.DictItemPostReq{TypeCode: "missing", Label: label, Value: "DRAFT"})
		if !errors.Is(err, shared.ErrNotFound) {
			t.Fatalf("error = %v, want ErrNotFound", err)
		}
		if len(itemRepo.items) != 0 {
			t.Fatalf("orphan item was created: %+v", itemRepo.items)
		}
	})

	t.Run("free-form color is rejected", func(t *testing.T) {
		typeRepo := newMockDictTypeRepo()
		seedDictType(typeRepo, &dictionary.DictType{ID: 1, Code: "order_status", IsEnabled: true})
		svc := NewDictionaryService(typeRepo, newMockDictItemRepo(), nil, "castor")

		_, err := svc.PostItem(context.Background(), &dto.DictItemPostReq{
			TypeCode: "order_status", Label: label, Value: "DRAFT", Color: "#ff0000",
		})
		if !errors.Is(err, apperror.ErrDictItemColorInvalid) {
			t.Fatalf("error = %v, want ErrDictItemColorInvalid", err)
		}
	})
}

// 系统字典项的 value 是与代码里枚举常量的连接键：改掉或删掉之后徽章退回原始值、
// 筛选下拉查不到数据，而且不报错。标签、颜色、启停等展示属性仍须可改。
func TestDictionaryService_SystemItemIsProtected(t *testing.T) {
	newService := func() (DictionaryService, *mockDictItemRepo) {
		itemRepo := newMockDictItemRepo()
		itemRepo.items[1] = &dictionary.DictItem{
			ID: 1, TypeCode: "asset_status", Value: "ACTIVE", IsSystem: true, IsEnabled: true,
			Label: shared.Text("Active", "生效", "有効", "활성"),
		}
		itemRepo.items[2] = &dictionary.DictItem{
			ID: 2, TypeCode: "asset_status", Value: "CUSTOM", IsEnabled: true,
			Label: shared.Text("Custom", "自定义", "カスタム", "사용자 정의"),
		}
		return NewDictionaryService(newMockDictTypeRepo(), itemRepo, nil, "castor"), itemRepo
	}

	t.Run("value cannot change", func(t *testing.T) {
		svc, itemRepo := newService()
		renamed := "Active"
		_, err := svc.PutItem(context.Background(), 1, &dto.DictItemPutReq{Value: &renamed})
		if !errors.Is(err, apperror.ErrSystemDictItemValueLocked) {
			t.Fatalf("error = %v, want ErrSystemDictItemValueLocked", err)
		}
		if itemRepo.items[1].Value != "ACTIVE" {
			t.Fatalf("value = %q, want it untouched", itemRepo.items[1].Value)
		}
	})

	t.Run("resubmitting the same value is not a change", func(t *testing.T) {
		svc, _ := newService()
		same := "ACTIVE"
		if _, err := svc.PutItem(context.Background(), 1, &dto.DictItemPutReq{Value: &same}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("presentation stays editable", func(t *testing.T) {
		svc, itemRepo := newService()
		label := dto.I18nTextReq{En: "Live", Zh: "上线", Ja: "公開中", Ko: "게시됨"}
		color, disabled := "blue", false
		_, err := svc.PutItem(context.Background(), 1, &dto.DictItemPutReq{Label: &label, Color: &color, IsEnabled: &disabled})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := itemRepo.items[1]
		if got.Label.Zh != "上线" || got.Color != "blue" || got.IsEnabled {
			t.Fatalf("presentation not updated: %+v", got)
		}
		if !got.IsSystem {
			t.Fatal("IsSystem must survive an update")
		}
	})

	t.Run("cannot be deleted", func(t *testing.T) {
		svc, itemRepo := newService()
		if err := svc.DeleteItem(context.Background(), 1); !errors.Is(err, apperror.ErrSystemDictItemDelete) {
			t.Fatalf("error = %v, want ErrSystemDictItemDelete", err)
		}
		if _, ok := itemRepo.items[1]; !ok {
			t.Fatal("system item was deleted")
		}
	})

	t.Run("operator-created items are unrestricted", func(t *testing.T) {
		svc, itemRepo := newService()
		renamed := "CUSTOM_V2"
		if _, err := svc.PutItem(context.Background(), 2, &dto.DictItemPutReq{Value: &renamed}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := svc.DeleteItem(context.Background(), 2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := itemRepo.items[2]; ok {
			t.Fatal("custom item should have been deleted")
		}
	})
}

func TestDictionaryService_PutItemRejectsUnknownColor(t *testing.T) {
	itemRepo := newMockDictItemRepo()
	itemRepo.items[1] = &dictionary.DictItem{ID: 1, TypeCode: "status", Value: "ENABLED", Color: "green"}
	svc := NewDictionaryService(newMockDictTypeRepo(), itemRepo, nil, "castor")

	bad := "rgb(0,0,0)"
	if _, err := svc.PutItem(context.Background(), 1, &dto.DictItemPutReq{Color: &bad}); !errors.Is(err, apperror.ErrDictItemColorInvalid) {
		t.Fatalf("error = %v, want ErrDictItemColorInvalid", err)
	}
	cleared := ""
	if _, err := svc.PutItem(context.Background(), 1, &dto.DictItemPutReq{Color: &cleared}); err != nil {
		t.Fatalf("clearing the color must be allowed: %v", err)
	}
	if itemRepo.items[1].Color != "" {
		t.Fatalf("color = %q, want cleared", itemRepo.items[1].Color)
	}
}

// 读取接口的可见范围：已登录拿全部启用类型，免认证只拿公开类型。
func TestDictionaryService_ReadScopes(t *testing.T) {
	typeRepo := newMockDictTypeRepo()
	seedDictType(typeRepo, &dictionary.DictType{ID: 1, Code: "gender", IsEnabled: true, IsPublic: true})
	seedDictType(typeRepo, &dictionary.DictType{ID: 2, Code: "audit_log_type", IsEnabled: true})
	seedDictType(typeRepo, &dictionary.DictType{ID: 3, Code: "retired", IsEnabled: false, IsPublic: true})
	seedDictType(typeRepo, &dictionary.DictType{ID: 4, Code: "empty_public", IsEnabled: true, IsPublic: true})

	itemRepo := newMockDictItemRepo()
	itemRepo.items[1] = &dictionary.DictItem{ID: 1, TypeCode: "gender", Value: "MALE", IsEnabled: true}
	itemRepo.items[2] = &dictionary.DictItem{ID: 2, TypeCode: "gender", Value: "HIDDEN", IsEnabled: false}
	itemRepo.items[3] = &dictionary.DictItem{ID: 3, TypeCode: "audit_log_type", Value: "LOGIN", IsEnabled: true}
	itemRepo.items[4] = &dictionary.DictItem{ID: 4, TypeCode: "retired", Value: "X", IsEnabled: true}

	svc := NewDictionaryService(typeRepo, itemRepo, nil, "castor")
	ctx := context.Background()

	enabled, err := svc.GetEnabledDicts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(enabled["gender"]) != 1 || enabled["gender"][0].Value != "MALE" {
		t.Errorf("gender = %+v, want only the enabled item", enabled["gender"])
	}
	if len(enabled["audit_log_type"]) != 1 {
		t.Errorf("signed-in callers must see non-public types, got %+v", enabled["audit_log_type"])
	}
	if _, ok := enabled["retired"]; ok {
		t.Error("disabled types must be excluded")
	}
	if items, ok := enabled["empty_public"]; !ok || items == nil {
		t.Error("an enabled type without items must still appear as an empty list")
	}

	public, err := svc.GetAllPublicDicts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := public["gender"]; !ok {
		t.Error("public type missing from the public endpoint")
	}
	for _, code := range []string{"audit_log_type", "retired"} {
		if _, ok := public[code]; ok {
			t.Errorf("%s must not be exposed without authentication", code)
		}
	}

	if items, err := svc.GetPublicDictItems(ctx, "gender"); err != nil || len(items) != 1 {
		t.Errorf("GetPublicDictItems(gender) = %+v, %v", items, err)
	}
	// 未公开与不存在对外不作区分
	for _, code := range []string{"audit_log_type", "retired", "nope"} {
		if _, err := svc.GetPublicDictItems(ctx, code); !errors.Is(err, shared.ErrNotFound) {
			t.Errorf("GetPublicDictItems(%s) error = %v, want ErrNotFound", code, err)
		}
	}
}

// 快照缓存：读走缓存，任何写入都必须让它失效，否则运营改完标签界面上看不到变化。
func TestDictionaryService_SnapshotCache(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	typeRepo := newMockDictTypeRepo()
	seedDictType(typeRepo, &dictionary.DictType{ID: 1, Code: "status", IsEnabled: true})
	typeRepo.nextID = 2
	itemRepo := newMockDictItemRepo()
	itemRepo.items[1] = &dictionary.DictItem{ID: 1, TypeCode: "status", Value: "ENABLED", IsEnabled: true}
	itemRepo.nextID = 2

	svc := NewDictionaryService(typeRepo, itemRepo, client, "castor")
	ctx := context.Background()
	label := dto.I18nTextReq{En: "Pending", Zh: "待处理", Ja: "保留中", Ko: "대기 중"}

	if _, err := svc.GetEnabledDicts(ctx); err != nil {
		t.Fatal(err)
	}
	if !redisServer.Exists("castor:" + constant.DICT_SNAPSHOT_CACHE_KEY) {
		t.Fatal("first read must populate the cache")
	}

	// 绕过服务直接改仓储：读到的仍是缓存，证明读路径确实走缓存
	itemRepo.items[9] = &dictionary.DictItem{ID: 9, TypeCode: "status", Value: "BYPASS", IsEnabled: true}
	cached, _ := svc.GetEnabledDicts(ctx)
	if len(cached["status"]) != 1 {
		t.Fatalf("expected the cached snapshot, got %+v", cached["status"])
	}
	delete(itemRepo.items, 9)

	writes := map[string]func() error{
		"PostItem": func() error {
			_, err := svc.PostItem(ctx, &dto.DictItemPostReq{TypeCode: "status", Label: label, Value: "PENDING"})
			return err
		},
		"PutItem": func() error {
			color := "red"
			_, err := svc.PutItem(ctx, 1, &dto.DictItemPutReq{Color: &color})
			return err
		},
		"DeleteItem": func() error { return svc.DeleteItem(ctx, 1) },
		"PostType": func() error {
			_, err := svc.PostType(ctx, &dto.DictTypePostReq{Code: "extra", Name: label})
			return err
		},
		"PutType": func() error {
			public := true
			_, err := svc.PutType(ctx, 1, &dto.DictTypePutReq{IsPublic: &public})
			return err
		},
		"DeleteType": func() error { return svc.DeleteType(ctx, 1) },
	}
	for _, name := range []string{"PostItem", "PutItem", "DeleteItem", "PostType", "PutType", "DeleteType"} {
		if _, err := svc.GetEnabledDicts(ctx); err != nil {
			t.Fatal(err)
		}
		if err := writes[name](); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if redisServer.Exists("castor:" + constant.DICT_SNAPSHOT_CACHE_KEY) {
			t.Errorf("%s must invalidate the snapshot cache", name)
		}
	}

	// Redis 不可用时退回数据库，界面标签不能因为缓存故障整体失效
	redisServer.Close()
	if _, err := svc.GetEnabledDicts(ctx); err != nil {
		t.Fatalf("reads must survive a Redis outage: %v", err)
	}
}
