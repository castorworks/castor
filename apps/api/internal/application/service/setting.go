package service

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/setting"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/jinzhu/copier"
)

// SettingService 系统配置应用服务接口
type SettingService interface {
	// 管理接口
	Gets(ctx context.Context, category string) ([]dto.SettingResp, error)
	Get(ctx context.Context, id uint) (*dto.SettingResp, error)
	GetByKey(ctx context.Context, key string) (*dto.SettingResp, error)
	Post(ctx context.Context, req *dto.SettingPostReq) (*dto.SettingResp, error)
	Put(ctx context.Context, id uint, req *dto.SettingPutReq) (*dto.SettingResp, error)
	Delete(ctx context.Context, id uint) error
	BatchUpdate(ctx context.Context, req *dto.SettingBatchUpdateReq) error

	// 公开接口
	GetPublicSettings(ctx context.Context) (map[string]interface{}, error)

	// 工具方法
	GetValue(ctx context.Context, key string) (string, error)
	GetBool(ctx context.Context, key string) (bool, error)
	GetInt(ctx context.Context, key string) (int, error)
}

type settingService struct {
	settingRepo setting.Repository
}

// NewSettingService 创建系统配置应用服务
func NewSettingService(settingRepo setting.Repository) SettingService {
	return &settingService{
		settingRepo: settingRepo,
	}
}

func (s *settingService) Gets(ctx context.Context, category string) ([]dto.SettingResp, error) {
	items, err := s.settingRepo.Gets(ctx, category)
	if err != nil {
		return nil, err
	}

	var resp []dto.SettingResp
	for _, item := range items {
		var r dto.SettingResp
		copier.Copy(&r, &item)
		resp = append(resp, r)
	}
	return resp, nil
}

func (s *settingService) Get(ctx context.Context, id uint) (*dto.SettingResp, error) {
	item, err := s.settingRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	var resp dto.SettingResp
	copier.Copy(&resp, item)
	return &resp, nil
}

func (s *settingService) GetByKey(ctx context.Context, key string) (*dto.SettingResp, error) {
	item, err := s.settingRepo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	var resp dto.SettingResp
	copier.Copy(&resp, item)
	return &resp, nil
}

func (s *settingService) Post(ctx context.Context, req *dto.SettingPostReq) (*dto.SettingResp, error) {
	// 检查 key 是否已存在
	exists, err := s.settingRepo.ExistsByKey(ctx, req.Key)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrSettingKeyExists
	}

	settingType := setting.SettingType(req.Type)
	if settingType == "" {
		settingType = setting.SettingTypeString
	}

	// 校验值是否符合声明的类型
	if req.Value != "" {
		normalizedValue, err := normalizeSettingValue(req.Key, req.Value, settingType)
		if err != nil {
			return nil, err
		}
		req.Value = normalizedValue
	}

	item := &setting.Setting{
		Key:         req.Key,
		Value:       req.Value,
		Type:        settingType,
		Category:    setting.SettingCategory(req.Category),
		Name:        req.Name,
		Description: req.Description,
		DefaultVal:  req.DefaultVal,
		IsPublic:    req.IsPublic,
		SortOrder:   req.SortOrder,
	}

	if err := s.settingRepo.Create(ctx, item); err != nil {
		return nil, err
	}

	var resp dto.SettingResp
	copier.Copy(&resp, item)
	return &resp, nil
}

func (s *settingService) Put(ctx context.Context, id uint, req *dto.SettingPutReq) (*dto.SettingResp, error) {
	item, err := s.settingRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// 更新字段（使用指针判断是否传入，允许设为空字符串）
	if req.Value != nil {
		// 校验值是否符合声明的类型
		if *req.Value != "" {
			normalizedValue, err := normalizeSettingValue(item.Key, *req.Value, item.Type)
			if err != nil {
				return nil, err
			}
			*req.Value = normalizedValue
		}
		item.Value = *req.Value
	}
	if req.Name != nil {
		item.Name = *req.Name
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.IsPublic != nil {
		item.IsPublic = *req.IsPublic
	}
	if req.SortOrder != nil {
		item.SortOrder = *req.SortOrder
	}

	if err := s.settingRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	var resp dto.SettingResp
	copier.Copy(&resp, item)
	return &resp, nil
}

func (s *settingService) Delete(ctx context.Context, id uint) error {
	// 在 service 层检查系统配置保护
	item, err := s.settingRepo.Get(ctx, id)
	if err != nil {
		return err
	}
	if item.IsSystem {
		return ErrSystemSettingDelete
	}
	return s.settingRepo.Delete(ctx, id)
}

func (s *settingService) BatchUpdate(ctx context.Context, req *dto.SettingBatchUpdateReq) error {
	// 收集所有 key，批量查询以获取类型信息用于校验
	keys := make([]string, len(req.Settings))
	for i, item := range req.Settings {
		keys[i] = item.Key
	}

	existingSettings, err := s.settingRepo.GetByKeys(ctx, keys)
	if err != nil {
		return err
	}

	// 构建 key → type 映射
	typeMap := make(map[string]setting.SettingType, len(existingSettings))
	for _, es := range existingSettings {
		typeMap[es.Key] = es.Type
	}

	// 校验每个值是否符合对应类型
	var settings []setting.Setting
	for _, item := range req.Settings {
		value := item.Value
		if item.Value != "" {
			if st, ok := typeMap[item.Key]; ok {
				normalizedValue, err := normalizeSettingValue(item.Key, item.Value, st)
				if err != nil {
					return err
				}
				value = normalizedValue
			}
		}
		settings = append(settings, setting.Setting{
			Key:   item.Key,
			Value: value,
		})
	}

	return s.settingRepo.BatchUpdate(ctx, settings)
}

func (s *settingService) GetPublicSettings(ctx context.Context) (map[string]interface{}, error) {
	items, err := s.settingRepo.GetPublicSettings(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for _, item := range items {
		result[item.Key] = parseSettingValue(item.Value, item.Type)
	}
	return result, nil
}

func (s *settingService) GetValue(ctx context.Context, key string) (string, error) {
	item, err := s.settingRepo.GetByKey(ctx, key)
	if err != nil {
		return "", err
	}
	return item.Value, nil
}

func (s *settingService) GetBool(ctx context.Context, key string) (bool, error) {
	value, err := s.GetValue(ctx, key)
	if err != nil {
		return false, err
	}
	return value == "true" || value == "1", nil
}

func (s *settingService) GetInt(ctx context.Context, key string) (int, error) {
	value, err := s.GetValue(ctx, key)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(value)
}

// validateSettingValue 根据类型校验配置值是否合法
func validateSettingValue(value string, settingType setting.SettingType) error {
	switch settingType {
	case setting.SettingTypeBool:
		if value != "true" && value != "false" && value != "1" && value != "0" {
			return ErrInvalidSettingValue
		}
	case setting.SettingTypeNumber:
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return ErrInvalidSettingValue
		}
	case setting.SettingTypeJSON:
		if !json.Valid([]byte(value)) {
			return ErrInvalidSettingValue
		}
	case setting.SettingTypeArray:
		// ARRAY 类型期望是 JSON 数组
		var arr []interface{}
		if err := json.Unmarshal([]byte(value), &arr); err != nil {
			return ErrInvalidSettingValue
		}
	case setting.SettingTypeString, setting.SettingTypeSecret:
		// 字符串和密钥类型不做额外校验
	}
	return nil
}

func normalizeSettingValue(key, value string, settingType setting.SettingType) (string, error) {
	if key == setting.KeySecurityLoginAllowedMethods {
		return normalizeAllowedLoginMethods(value)
	}
	if err := validateSettingValue(value, settingType); err != nil {
		return "", err
	}
	return value, nil
}

func normalizeAllowedLoginMethods(value string) (string, error) {
	var methods []string
	if err := json.Unmarshal([]byte(value), &methods); err != nil {
		return "", ErrInvalidLoginMethodSetting
	}

	seen := make(map[string]bool, len(methods))
	for _, method := range methods {
		switch method {
		case constant.LOGIN_METHOD_PASSWORD, constant.LOGIN_METHOD_EMAIL, constant.LOGIN_METHOD_MOBILE:
			seen[method] = true
		default:
			return "", ErrInvalidLoginMethodSetting
		}
	}

	if len(seen) == 0 {
		return "", ErrInvalidLoginMethodSetting
	}

	ordered := make([]string, 0, 3)
	for _, method := range []string{
		constant.LOGIN_METHOD_PASSWORD,
		constant.LOGIN_METHOD_EMAIL,
		constant.LOGIN_METHOD_MOBILE,
	} {
		if seen[method] {
			ordered = append(ordered, method)
		}
	}

	normalized, err := json.Marshal(ordered)
	if err != nil {
		return "", err
	}
	return string(normalized), nil
}

// parseSettingValue 根据类型解析配置值
func parseSettingValue(value string, settingType setting.SettingType) interface{} {
	switch settingType {
	case setting.SettingTypeBool:
		return value == "true" || value == "1"
	case setting.SettingTypeNumber:
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
		return value
	case setting.SettingTypeArray:
		var arr []string
		if err := json.Unmarshal([]byte(value), &arr); err == nil {
			return arr
		}
		var generic []interface{}
		if err := json.Unmarshal([]byte(value), &generic); err == nil {
			return generic
		}
		return value
	default:
		return value
	}
}
