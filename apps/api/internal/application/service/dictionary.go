package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/dictionary"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/rediskey"
	"github.com/jinzhu/copier"
	"github.com/redis/go-redis/v9"
)

// dictSnapshotTTL 是字典快照缓存的兜底过期时间。服务内的每次写入都会主动失效缓存，
// TTL 只用来覆盖绕过服务直接写库的路径（init-db 对账）。
const dictSnapshotTTL = 5 * time.Minute

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

	// 读取接口（走快照缓存）
	// GetEnabledDicts 返回所有启用类型下的启用字典项，供已登录的界面渲染标签。
	GetEnabledDicts(ctx context.Context) (map[string][]dto.DictItemResp, error)
	// GetAllPublicDicts 只返回 IsPublic 的类型，供免认证页面使用。
	GetAllPublicDicts(ctx context.Context) (map[string][]dto.DictItemResp, error)
	// GetPublicDictItems 返回单个公开类型的字典项；类型未公开或未启用时返回 ErrNotFound。
	GetPublicDictItems(ctx context.Context, typeCode string) ([]dto.DictItemResp, error)
}

type dictionaryService struct {
	typeRepo    dictionary.DictTypeRepository
	itemRepo    dictionary.DictItemRepository
	redisClient redis.UniversalClient
	cacheKey    string
}

// NewDictionaryService 创建字典服务。redisClient 为 nil 时不启用缓存（单元测试）。
func NewDictionaryService(
	typeRepo dictionary.DictTypeRepository,
	itemRepo dictionary.DictItemRepository,
	redisClient redis.UniversalClient,
	ns rediskey.Namespace,
) DictionaryService {
	return &dictionaryService{
		typeRepo:    typeRepo,
		itemRepo:    itemRepo,
		redisClient: redisClient,
		cacheKey:    ns.Key(constant.DICT_SNAPSHOT_CACHE_KEY),
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
		Name:        req.Name.ToI18nText(),
		Description: req.Description,
		IsPublic:    req.IsPublic,
		IsEnabled:   true,
		SortOrder:   req.SortOrder,
	}

	if err := s.typeRepo.Create(ctx, item); err != nil {
		return nil, err
	}
	s.invalidateSnapshot(ctx)

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
		item.Name = req.Name.ToI18nText()
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.IsPublic != nil {
		item.IsPublic = *req.IsPublic
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
	s.invalidateSnapshot(ctx)

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
	if err := s.typeRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidateSnapshot(ctx)
	return nil
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
	if !dictionary.IsValidColor(req.Color) {
		return nil, ErrDictItemColorInvalid
	}
	// 所属类型必须存在，否则会留下任何界面都看不到的孤儿项
	if _, err := s.typeRepo.GetByCode(ctx, req.TypeCode); err != nil {
		return nil, err
	}

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
		Label:       req.Label.ToI18nText(),
		Value:       req.Value,
		Description: req.Description,
		Color:       req.Color,
		Icon:        req.Icon,
		IsDefault:   req.IsDefault,
		IsEnabled:   true,
		SortOrder:   req.SortOrder,
	}

	if err := s.itemRepo.Create(ctx, item); err != nil {
		return nil, err
	}
	s.invalidateSnapshot(ctx)

	var resp dto.DictItemResp
	copier.Copy(&resp, item)
	return &resp, nil
}

func (s *dictionaryService) PutItem(ctx context.Context, id uint, req *dto.DictItemPutReq) (*dto.DictItemResp, error) {
	item, err := s.itemRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Color != nil && !dictionary.IsValidColor(*req.Color) {
		return nil, ErrDictItemColorInvalid
	}

	// 如果更新了 value，检查唯一性
	if req.Value != nil && *req.Value != "" && *req.Value != item.Value {
		// 系统项的 value 是与代码里枚举常量的连接键，改掉之后徽章退回原始值、
		// 筛选下拉查不到数据，且不会有任何报错。
		if item.IsSystem {
			return nil, ErrSystemDictItemValueLocked
		}
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
		item.Label = req.Label.ToI18nText()
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.Color != nil {
		item.Color = *req.Color
	}
	if req.Icon != nil {
		item.Icon = *req.Icon
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
	s.invalidateSnapshot(ctx)

	var resp dto.DictItemResp
	copier.Copy(&resp, item)
	return &resp, nil
}

func (s *dictionaryService) DeleteItem(ctx context.Context, id uint) error {
	item, err := s.itemRepo.Get(ctx, id)
	if err != nil {
		return err
	}
	// 系统项被代码引用，只能停用不能删除
	if item.IsSystem {
		return ErrSystemDictItemDelete
	}
	if err := s.itemRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidateSnapshot(ctx)
	return nil
}

// ========== 读取接口 ==========

// dictSnapshot 是读取接口共用的一份快照：所有启用类型下的启用字典项，外加公开类型的编码。
// 三个读取接口都从它派生，因此只有一个缓存键、一处失效点。
type dictSnapshot struct {
	Dicts  map[string][]dto.DictItemResp `json:"dicts"`
	Public []string                      `json:"public"`
}

func (s *dictionaryService) GetEnabledDicts(ctx context.Context) (map[string][]dto.DictItemResp, error) {
	snapshot, err := s.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	return snapshot.Dicts, nil
}

func (s *dictionaryService) GetAllPublicDicts(ctx context.Context) (map[string][]dto.DictItemResp, error) {
	snapshot, err := s.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]dto.DictItemResp, len(snapshot.Public))
	for _, code := range snapshot.Public {
		result[code] = snapshot.Dicts[code]
	}
	return result, nil
}

func (s *dictionaryService) GetPublicDictItems(ctx context.Context, typeCode string) ([]dto.DictItemResp, error) {
	snapshot, err := s.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	for _, code := range snapshot.Public {
		if code == typeCode {
			return snapshot.Dicts[code], nil
		}
	}
	// 未公开与不存在对外不作区分，避免免认证接口被用来探测内部字典类型
	return nil, apperror.ErrRecordNotFound
}

// snapshot 优先读缓存；缓存不可用时退回数据库，Redis 故障不应让界面标签整体失效。
func (s *dictionaryService) snapshot(ctx context.Context) (*dictSnapshot, error) {
	if cached := s.cachedSnapshot(ctx); cached != nil {
		return cached, nil
	}
	snapshot, err := s.loadSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	s.storeSnapshot(ctx, snapshot)
	return snapshot, nil
}

func (s *dictionaryService) loadSnapshot(ctx context.Context) (*dictSnapshot, error) {
	types, err := s.typeRepo.Gets(ctx)
	if err != nil {
		return nil, err
	}
	// 一次性查出所有启用的字典项，避免 N+1 查询
	items, err := s.itemRepo.GetAllEnabled(ctx)
	if err != nil {
		return nil, err
	}

	snapshot := &dictSnapshot{Dicts: make(map[string][]dto.DictItemResp), Public: []string{}}
	for _, t := range types {
		if !t.IsEnabled {
			continue
		}
		// 启用但暂无字典项的类型也要出现，调用方据此区分"空字典"与"类型不存在"
		snapshot.Dicts[t.Code] = []dto.DictItemResp{}
		if t.IsPublic {
			snapshot.Public = append(snapshot.Public, t.Code)
		}
	}
	for i := range items {
		if _, enabled := snapshot.Dicts[items[i].TypeCode]; !enabled {
			continue
		}
		var r dto.DictItemResp
		if err := r.FromEntity(&items[i]); err != nil {
			return nil, err
		}
		snapshot.Dicts[items[i].TypeCode] = append(snapshot.Dicts[items[i].TypeCode], r)
	}
	return snapshot, nil
}

func (s *dictionaryService) cachedSnapshot(ctx context.Context) *dictSnapshot {
	if s.redisClient == nil {
		return nil
	}
	raw, err := s.redisClient.Get(ctx, s.cacheKey).Bytes()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			log.WarnCtx(ctx).Err(err).Msg("Read dictionary snapshot cache failed")
		}
		return nil
	}
	var snapshot dictSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil || snapshot.Dicts == nil {
		// 结构变更后的旧缓存：当作未命中，由下一次写入覆盖
		return nil
	}
	return &snapshot
}

func (s *dictionaryService) storeSnapshot(ctx context.Context, snapshot *dictSnapshot) {
	if s.redisClient == nil {
		return
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		log.WarnCtx(ctx).Err(err).Msg("Encode dictionary snapshot cache failed")
		return
	}
	if err := s.redisClient.Set(ctx, s.cacheKey, raw, dictSnapshotTTL).Err(); err != nil {
		log.WarnCtx(ctx).Err(err).Msg("Write dictionary snapshot cache failed")
	}
}

// invalidateSnapshot 在每次写入后调用。失效失败只记日志：数据已经落库，
// 让写请求因缓存故障而报错只会误导调用方，TTL 会兜底收敛。
func (s *dictionaryService) invalidateSnapshot(ctx context.Context) {
	if s.redisClient == nil {
		return
	}
	if err := s.redisClient.Del(ctx, s.cacheKey).Err(); err != nil {
		log.WarnCtx(ctx).Err(err).Msg("Invalidate dictionary snapshot cache failed")
	}
}
