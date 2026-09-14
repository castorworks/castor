package service

import (
	"context"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/dictionary"
	"github.com/jinzhu/copier"
)

// DictionaryService 字典服务接口
type DictionaryService interface {
	// 字典类型
	GetTypes(ctx context.Context) ([]dto.DictTypeResp, error)
	GetType(ctx context.Context, id uint) (*dto.DictTypeResp, error)
	GetTypeByCode(ctx context.Context, code string) (*dto.DictTypeResp, error)
	PostType(ctx context.Context, req *dto.DictTypePostReq) (*dto.DictTypeResp, error)
	PutType(ctx context.Context, id uint, req *dto.DictTypePutReq) (*dto.DictTypeResp, error)
	DeleteType(ctx context.Context, id uint) error

	// 字典项
	GetItems(ctx context.Context, typeCode string) ([]dto.DictItemResp, error)
	GetItem(ctx context.Context, id uint) (*dto.DictItemResp, error)
	GetItemsByTypeCode(ctx context.Context, typeCode string) ([]dto.DictItemResp, error)
	PostItem(ctx context.Context, req *dto.DictItemPostReq) (*dto.DictItemResp, error)
	PutItem(ctx context.Context, id uint, req *dto.DictItemPutReq) (*dto.DictItemResp, error)
	DeleteItem(ctx context.Context, id uint) error

	// 公开接口
	GetPublicDictItems(ctx context.Context, typeCode string) ([]dto.DictItemResp, error)
	GetAllPublicDicts(ctx context.Context) (map[string][]dto.DictItemResp, error)
}

type dictionaryService struct {
	typeRepo dictionary.DictTypeRepository
	itemRepo dictionary.DictItemRepository
}

// NewDictionaryService 创建字典服务
func NewDictionaryService(
	typeRepo dictionary.DictTypeRepository,
	itemRepo dictionary.DictItemRepository,
) DictionaryService {
	return &dictionaryService{
		typeRepo: typeRepo,
		itemRepo: itemRepo,
	}
}

// ========== 字典类型 ==========

func (s *dictionaryService) GetTypes(ctx context.Context) ([]dto.DictTypeResp, error) {
	items, err := s.typeRepo.Gets(ctx)
	if err != nil {
		return nil, err
	}

	var resp []dto.DictTypeResp
	for _, item := range items {
		var r dto.DictTypeResp
		copier.Copy(&r, &item)
		resp = append(resp, r)
	}
	return resp, nil
}

func (s *dictionaryService) GetType(ctx context.Context, id uint) (*dto.DictTypeResp, error) {
	item, err := s.typeRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	var resp dto.DictTypeResp
	copier.Copy(&resp, item)
	return &resp, nil
}

func (s *dictionaryService) GetTypeByCode(ctx context.Context, code string) (*dto.DictTypeResp, error) {
	item, err := s.typeRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	var resp dto.DictTypeResp
	copier.Copy(&resp, item)
	return &resp, nil
}

func (s *dictionaryService) PostType(ctx context.Context, req *dto.DictTypePostReq) (*dto.DictTypeResp, error) {
	// 检查 code 是否已存在
	exists, err := s.typeRepo.ExistsByCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDictTypeCodeExists
	}

	item := &dictionary.DictType{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		IsEnabled:   true,
		SortOrder:   req.SortOrder,
	}

	if err := s.typeRepo.Create(ctx, item); err != nil {
		return nil, err
	}

	var resp dto.DictTypeResp
	copier.Copy(&resp, item)
	return &resp, nil
}

func (s *dictionaryService) PutType(ctx context.Context, id uint, req *dto.DictTypePutReq) (*dto.DictTypeResp, error) {
	item, err := s.typeRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		item.Name = *req.Name
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.IsEnabled != nil {
		item.IsEnabled = *req.IsEnabled
	}
	if req.SortOrder != nil {
		item.SortOrder = *req.SortOrder
	}

	if err := s.typeRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	var resp dto.DictTypeResp
	copier.Copy(&resp, item)
	return &resp, nil
}

func (s *dictionaryService) DeleteType(ctx context.Context, id uint) error {
	// 删除字典类型时同时删除其下的所有字典项
	item, err := s.typeRepo.Get(ctx, id)
	if err != nil {
		return err
	}
	// 系统内置类型不可删除，必须在删除字典项之前校验
	if item.IsSystem {
		return apperror.ErrSystemDictTypeDelete
	}

	// 先删除字典项
	if err := s.itemRepo.DeleteByTypeCode(ctx, item.Code); err != nil {
		return err
	}

	// 再删除字典类型
	return s.typeRepo.Delete(ctx, id)
}

// ========== 字典项 ==========

func (s *dictionaryService) GetItems(ctx context.Context, typeCode string) ([]dto.DictItemResp, error) {
	items, err := s.itemRepo.Gets(ctx, typeCode)
	if err != nil {
		return nil, err
	}

	var resp []dto.DictItemResp
	for _, item := range items {
		var r dto.DictItemResp
		copier.Copy(&r, &item)
		resp = append(resp, r)
	}
	return resp, nil
}

func (s *dictionaryService) GetItem(ctx context.Context, id uint) (*dto.DictItemResp, error) {
	item, err := s.itemRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	var resp dto.DictItemResp
	copier.Copy(&resp, item)
	return &resp, nil
}

func (s *dictionaryService) GetItemsByTypeCode(ctx context.Context, typeCode string) ([]dto.DictItemResp, error) {
	items, err := s.itemRepo.GetByTypeCode(ctx, typeCode)
	if err != nil {
		return nil, err
	}

	var resp []dto.DictItemResp
	for _, item := range items {
		var r dto.DictItemResp
		copier.Copy(&r, &item)
		resp = append(resp, r)
	}
	return resp, nil
}

func (s *dictionaryService) PostItem(ctx context.Context, req *dto.DictItemPostReq) (*dto.DictItemResp, error) {
	// 检查同类型下是否已存在相同 value
	exists, err := s.itemRepo.ExistsByTypeCodeAndValue(ctx, req.TypeCode, req.Value, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDictItemValueExists
	}

	item := &dictionary.DictItem{
		TypeCode:    req.TypeCode,
		Label:       req.Label,
		Value:       req.Value,
		Description: req.Description,
		Extra:       req.Extra,
		Color:       req.Color,
		Icon:        req.Icon,
		ParentID:    req.ParentID,
		IsDefault:   req.IsDefault,
		IsEnabled:   true,
		SortOrder:   req.SortOrder,
	}

	if err := s.itemRepo.Create(ctx, item); err != nil {
		return nil, err
	}

	var resp dto.DictItemResp
	copier.Copy(&resp, item)
	return &resp, nil
}

func (s *dictionaryService) PutItem(ctx context.Context, id uint, req *dto.DictItemPutReq) (*dto.DictItemResp, error) {
	item, err := s.itemRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// 如果更新了 value，检查唯一性
	if req.Value != nil && *req.Value != "" && *req.Value != item.Value {
		exists, err := s.itemRepo.ExistsByTypeCodeAndValue(ctx, item.TypeCode, *req.Value, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrDictItemValueExists
		}
		item.Value = *req.Value
	}

	if req.Label != nil {
		item.Label = *req.Label
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.Extra != nil {
		item.Extra = *req.Extra
	}
	if req.Color != nil {
		item.Color = *req.Color
	}
	if req.Icon != nil {
		item.Icon = *req.Icon
	}
	if req.ParentID != nil {
		item.ParentID = req.ParentID
	}
	if req.IsDefault != nil {
		item.IsDefault = *req.IsDefault
	}
	if req.IsEnabled != nil {
		item.IsEnabled = *req.IsEnabled
	}
	if req.SortOrder != nil {
		item.SortOrder = *req.SortOrder
	}

	if err := s.itemRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	var resp dto.DictItemResp
	copier.Copy(&resp, item)
	return &resp, nil
}

func (s *dictionaryService) DeleteItem(ctx context.Context, id uint) error {
	return s.itemRepo.Delete(ctx, id)
}

// ========== 公开接口 ==========

func (s *dictionaryService) GetPublicDictItems(ctx context.Context, typeCode string) ([]dto.DictItemResp, error) {
	items, err := s.itemRepo.GetEnabledByTypeCode(ctx, typeCode)
	if err != nil {
		return nil, err
	}

	var resp []dto.DictItemResp
	for _, item := range items {
		var r dto.DictItemResp
		copier.Copy(&r, &item)
		resp = append(resp, r)
	}
	return resp, nil
}

func (s *dictionaryService) GetAllPublicDicts(ctx context.Context) (map[string][]dto.DictItemResp, error) {
	// 获取所有启用的字典类型
	types, err := s.typeRepo.Gets(ctx)
	if err != nil {
		return nil, err
	}

	// 一次性查出所有启用的字典项，避免 N+1 查询
	allItems, err := s.itemRepo.GetAllEnabled(ctx)
	if err != nil {
		return nil, err
	}

	// 构建启用类型的 code 集合
	enabledTypes := make(map[string]bool, len(types))
	for _, t := range types {
		if t.IsEnabled {
			enabledTypes[t.Code] = true
		}
	}

	// 按 typeCode 分组
	result := make(map[string][]dto.DictItemResp)
	for _, item := range allItems {
		if !enabledTypes[item.TypeCode] {
			continue
		}
		var r dto.DictItemResp
		copier.Copy(&r, &item)
		result[item.TypeCode] = append(result[item.TypeCode], r)
	}

	return result, nil
}
