package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/asset"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/pkg/generator"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/castorworks/castor/internal/pkg/ucontext"
)

// AssetDownloadResult 下载描述（供 Handler 层写响应头；内容由 WriteContent 写出）
type AssetDownloadResult struct {
	ObjectKey   string
	Filename    string
	ContentType string
	Size        int64
}

// AttachmentUpload 业务模块上传附件的输入
type AttachmentUpload struct {
	Filename    string
	File        multipart.File
	ContentType string
	Size        int64
	// IsPublic 附件是否允许经免认证的下载接口访问（高风险类型始终不公开）
	IsPublic bool
	// Policy 该业务字段自己的大小/类型限制；零值沿用全局上传策略
	Policy AssetPolicy
}

// AttachOptions 引用资产时对资产本身的要求
type AttachOptions struct {
	// RequirePublic 业务需要经免认证接口访问该文件：私有资产会被提升为公开（记审计），
	// 未生效或高风险类型的资产则拒绝引用
	RequirePublic bool
	// RequireCategory 限定资产分类（如头像只接受 IMAGE）；空值不限制
	RequireCategory asset.AssetCategory
	// DisplayName 引用方给这个文件起的显示名（下载文件名、附件列表里的名字）；
	// 为空时退回资产自己的文件名。Sync 按每个文件各自的 Name 登记，忽略这里的取值。
	DisplayName string
}

// AssetReferencer 业务模块使用资产的接口面：业务记录保存文件字段时登记引用，
// 删除或换文件时解除引用。被引用的资产不可删除；ATTACHMENT 资产失去最后一个引用即被回收。
type AssetReferencer interface {
	// Attach 登记 ref 对 objectKey 的引用（幂等）
	Attach(ctx context.Context, objectKey string, ref asset.Reference, opts AttachOptions) error

	// Detach 解除 ref 对 objectKey 的引用；引用本就不存在时只做孤儿附件回收
	Detach(ctx context.Context, objectKey string, ref asset.Reference) error

	// Replace 让单值字段 ref 只引用 objectKey：解除该字段上的其他引用。
	// objectKey 为空表示清空该字段。调用前 objectKey 必须已经 Attach。
	Replace(ctx context.Context, ref asset.Reference, objectKey string) error

	// DetachOwner 解除某个业务对象的全部引用（业务记录删除时调用）
	DetachOwner(ctx context.Context, ownerType string, ownerID uint) error

	// Sync 让多值字段 ref 恰好引用 files：先登记新增的，再解除不在其中的。
	// 任何一个对象键登记失败都会整体返回错误，此时已有的引用原样保留。
	Sync(ctx context.Context, ref asset.Reference, files []asset.AttachedFile, opts AttachOptions) error

	// ListAttached 批量读取若干业务记录在 field 上的附件（按登记顺序），用于列表页避免 N+1
	ListAttached(ctx context.Context, field asset.Field, ownerIDs []uint) (map[uint][]dto.AssetAttachmentResp, error)

	// PrepareAttachedDownload 供业务模块自己的下载路由使用：调用方先完成自己的鉴权（谁能看这条业务记录），
	// 这里只确认 objectKey 确实挂在 ref 上——否则任何人都能借一条自己可见的记录读走任意文件。
	// 不要求资产公开；未挂载与不存在一律返回 apperror.ErrAssetNotFound。
	PrepareAttachedDownload(ctx context.Context, objectKey string, ref asset.Reference) (*AssetDownloadResult, error)

	// WriteContent 把资产内容写入 w
	WriteContent(ctx context.Context, objectKey string, w io.Writer) error
}

// AssetService 资产应用服务接口
type AssetService interface {
	AssetReferencer

	// ========================
	// 上传
	// ========================

	// Upload 管理端上传：缺省进资产库（LIBRARY，由运营手动管理）；
	// req.Scope 为 ATTACHMENT 时是后台业务表单里的文件字段，随引用回收
	Upload(ctx context.Context, req *dto.AssetUploadReq, filename string, f multipart.File, contentType string, size int64) (*dto.AssetUploadResp, error)

	// UploadAttachment 业务模块上传附件（ATTACHMENT，随引用回收）。
	// 内容与已有资产重复时返回已有资产，调用方随后必须 Attach。
	UploadAttachment(ctx context.Context, in AttachmentUpload) (*dto.AssetResp, error)

	// ========================
	// 查询
	// ========================

	// Get 根据 ID 获取资产详情（含引用方）
	Get(ctx context.Context, id uint) (*dto.AssetDetailResp, error)

	// Gets 分页查询资产列表
	Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]dto.AssetResp, int64, error)

	// GetPublicAssets 获取公开资产列表
	GetPublicAssets(ctx context.Context, page, size int, order string) ([]dto.AssetResp, int64, error)

	// GetStats 获取资产统计信息
	GetStats(ctx context.Context) (*dto.AssetStatsResp, error)

	// ========================
	// 更新
	// ========================

	// Update 更新资产信息；被引用的资产不可取消公开
	Update(ctx context.Context, id uint, req *dto.AssetUpdateReq) (*dto.AssetResp, error)

	// UpdateStatus 更新资产状态；被引用的资产只能保持 ACTIVE
	UpdateStatus(ctx context.Context, id uint, status asset.AssetStatus) error

	// BatchUpdateStatus 批量更新资产状态
	BatchUpdateStatus(ctx context.Context, ids []uint, status asset.AssetStatus) error

	// Move 移动资产到指定虚拟文件夹
	Move(ctx context.Context, id uint, req *dto.AssetMoveReq) error

	// ========================
	// 删除
	// ========================

	// Delete 删除资产；被引用时返回 apperror.ErrAssetInUse
	Delete(ctx context.Context, id uint) error

	// BatchDelete 批量删除资产；任意一条被引用则整体拒绝
	BatchDelete(ctx context.Context, ids []uint) error

	// SweepOrphanAttachments 回收超过宽限期仍无人引用的业务附件（上传后表单没保存、回收曾失败等），
	// 返回回收的数量。olderThan <= 0 时不做任何事。
	SweepOrphanAttachments(ctx context.Context, olderThan time.Duration) (int, error)

	// ========================
	// 下载
	// ========================

	// PrepareDownload 校验可见性并返回下载描述（不直接操作 HTTP）
	PrepareDownload(ctx context.Context, objectKey string, requirePublic bool) (*AssetDownloadResult, error)
}

type assetService struct {
	storage   asset.Storage
	assetRepo asset.Repository
	policy    AssetPolicy
	auditLog  AuditLogService
}

// NewAssetService 创建资产应用服务
func NewAssetService(storage asset.Storage, assetRepo asset.Repository, policy AssetPolicy, auditLog AuditLogService) AssetService {
	return &assetService{
		storage:   storage,
		assetRepo: assetRepo,
		policy:    policy,
		auditLog:  auditLog,
	}
}

// ========================
// 上传
// ========================

func (svc *assetService) Upload(ctx context.Context, req *dto.AssetUploadReq, filename string, f multipart.File, contentType string, size int64) (*dto.AssetUploadResp, error) {
	scope := req.Scope
	if scope == "" {
		scope = asset.ScopeLibrary
	}
	item, duplicate, err := svc.store(ctx, storeInput{
		Filename:    filename,
		File:        f,
		ContentType: contentType,
		Size:        size,
		Name:        req.Name,
		Description: req.Description,
		Tags:        req.Tags,
		FolderPath:  req.FolderPath,
		IsPublic:    req.IsPublic,
		Scope:       scope,
		Policy:      svc.policy,
	})
	if err != nil {
		return nil, err
	}
	// 运营要放进资产库的文件若去重到一条业务附件（如某人的头像）上，必须把它转为资产库资产，
	// 否则它既不出现在资产库列表里，还会随那条业务引用的解除而被回收。
	if duplicate && scope == asset.ScopeLibrary && item.Scope != asset.ScopeLibrary {
		if err := svc.assetRepo.UpdateScope(ctx, item.ID, asset.ScopeLibrary); err != nil {
			return nil, err
		}
		item.Scope = asset.ScopeLibrary
	}

	resp := &dto.AssetUploadResp{IsDuplicate: duplicate}
	if err := resp.AssetResp.FromEntity(item); err != nil {
		return nil, err
	}
	return resp, nil
}

func (svc *assetService) UploadAttachment(ctx context.Context, in AttachmentUpload) (*dto.AssetResp, error) {
	item, _, err := svc.store(ctx, storeInput{
		Filename:    in.Filename,
		File:        in.File,
		ContentType: in.ContentType,
		Size:        in.Size,
		IsPublic:    in.IsPublic,
		Scope:       asset.ScopeAttachment,
		Policy:      svc.policy.narrowedBy(in.Policy),
	})
	if err != nil {
		return nil, err
	}

	resp := &dto.AssetResp{}
	if err := resp.FromEntity(item); err != nil {
		return nil, err
	}
	return resp, nil
}

type storeInput struct {
	Filename    string
	File        multipart.File
	ContentType string
	Size        int64
	Name        string
	Description string
	Tags        string
	FolderPath  string
	IsPublic    bool
	Scope       asset.AssetScope
	Policy      AssetPolicy
}

// store 是所有上传共用的流水线：策略校验 → 内容嗅探 → 哈希去重 → 写存储 → 建记录。
// 返回的 bool 表示命中去重、返回的是已有资产（其名称、可见性等属性不受本次请求影响）。
func (svc *assetService) store(ctx context.Context, in storeInput) (*asset.Asset, bool, error) {
	f, size, contentType := in.File, in.Size, in.ContentType
	policy := in.Policy

	maxSize := policy.MaxUploadSize
	if maxSize > 0 && size > maxSize {
		return nil, false, apperror.ErrAssetTooLarge
	}

	ext := strings.ToLower(filepath.Ext(in.Filename))
	if !policy.allowsExtension(ext) {
		return nil, false, apperror.ErrAssetInvalidType
	}

	// Do not trust the multipart Content-Type or declared size. Inspect the
	// file bytes and obtain the actual size from the stream before storing it.
	detectedType, actualSize, err := inspectUploadedFile(f)
	if err != nil {
		return nil, false, apperror.ErrAssetInvalidType
	}
	if actualSize > 0 {
		size = actualSize
	}
	if maxSize > 0 && size > maxSize {
		return nil, false, apperror.ErrAssetTooLarge
	}
	if !policy.allowsMimeType(detectedType) || !contentTypeMatchesExtension(ext, detectedType) {
		return nil, false, apperror.ErrAssetInvalidType
	}
	// SVG and archive formats can carry active content or resource exhaustion
	// payloads; never expose them publicly without an asynchronous scan.
	isPublic := in.IsPublic && !isHighRiskExtension(ext)
	if detectedType != "" {
		contentType = detectedType
	}

	hash, err := calculateFileHash(f)
	if err != nil {
		return nil, false, err
	}
	if seeker, ok := f.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return nil, false, err
		}
	}

	if existing, err := svc.assetRepo.GetByHash(ctx, hash); err == nil && existing != nil {
		svc.renewGracePeriod(ctx, existing)
		return existing, true, nil
	} else if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, false, err
	}

	name := in.Name
	if name == "" {
		name = strings.TrimSuffix(in.Filename, filepath.Ext(in.Filename))
	}

	// 先写存储（失败不会留下脏数据）
	objectKey := generator.GenerateObjectKey() + ext
	if err := svc.storage.Put(ctx, objectKey, f, size, contentType); err != nil {
		return nil, false, fmt.Errorf("upload to storage: %w", err)
	}

	item := &asset.Asset{
		Name:        name,
		Filename:    in.Filename,
		Description: in.Description,
		Tags:        in.Tags,
		FolderPath:  in.FolderPath,
		ObjectKey:   objectKey,
		Bucket:      svc.storage.Bucket(),
		Extension:   ext,
		MimeType:    contentType,
		Size:        size,
		Hash:        hash,
		Category:    detectCategory(contentType, ext),
		Status:      asset.StatusActive,
		Scope:       in.Scope,
		IsPublic:    isPublic,
	}

	if err := svc.assetRepo.Create(ctx, item); err != nil {
		svc.removeObject(ctx, objectKey)
		// 哈希唯一索引裁决了并发上传：另一请求抢先建了记录，返回它即可。
		if existing, getErr := svc.assetRepo.GetByHash(ctx, hash); getErr == nil && existing != nil {
			return existing, true, nil
		}
		return nil, false, err
	}
	return item, false, nil
}

// ========================
// 查询
// ========================

func (svc *assetService) Get(ctx context.Context, id uint) (*dto.AssetDetailResp, error) {
	item, err := svc.assetRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	refs, err := svc.assetRepo.GetReferences(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := &dto.AssetDetailResp{References: refs}
	if err := resp.AssetResp.FromEntity(item); err != nil {
		return nil, err
	}
	return resp, nil
}

func (svc *assetService) Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]dto.AssetResp, int64, error) {
	items, total, err := svc.assetRepo.Gets(ctx, page, size, order, opts...)
	if err != nil {
		return nil, 0, err
	}
	return toAssetResps(items, total)
}

func (svc *assetService) GetPublicAssets(ctx context.Context, page, size int, order string) ([]dto.AssetResp, int64, error) {
	items, total, err := svc.assetRepo.GetPublicAssets(ctx, page, size, order)
	if err != nil {
		return nil, 0, err
	}
	return toAssetResps(items, total)
}

func toAssetResps(items []asset.Asset, total int64) ([]dto.AssetResp, int64, error) {
	resp := make([]dto.AssetResp, len(items))
	for i := range items {
		if err := resp[i].FromEntity(&items[i]); err != nil {
			return nil, 0, err
		}
	}
	return resp, total, nil
}

func (svc *assetService) GetStats(ctx context.Context) (*dto.AssetStatsResp, error) {
	categoryStats, statusStats, totalSize, err := svc.assetRepo.GetCombinedStats(ctx)
	if err != nil {
		return nil, err
	}

	var totalCount int64
	for _, count := range categoryStats {
		totalCount += count
	}

	return &dto.AssetStatsResp{
		TotalCount:         totalCount,
		TotalSize:          totalSize,
		TotalSizeFormatted: formatSize(totalSize),
		CategoryStats:      categoryStats,
		StatusStats:        statusStats,
	}, nil
}

// ========================
// 更新
// ========================

func (svc *assetService) Update(ctx context.Context, id uint, req *dto.AssetUpdateReq) (*dto.AssetResp, error) {
	item, err := svc.assetRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		item.Name = *req.Name
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.Tags != nil {
		item.Tags = *req.Tags
	}
	if req.FolderPath != nil {
		item.FolderPath = *req.FolderPath
	}
	if req.IsPublic != nil && *req.IsPublic != item.IsPublic {
		if *req.IsPublic {
			if isHighRiskExtension(strings.ToLower(item.Extension)) {
				return nil, apperror.ErrAssetInvalidType
			}
		} else if err := svc.ensureUnreferenced(ctx, id); err != nil {
			// 引用方可能正通过免认证接口展示这个文件（如头像），取消公开会让它静默失效。
			return nil, err
		}
		item.IsPublic = *req.IsPublic
	}

	if err := svc.assetRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	resp := &dto.AssetResp{}
	if err := resp.FromEntity(item); err != nil {
		return nil, err
	}
	return resp, nil
}

func (svc *assetService) UpdateStatus(ctx context.Context, id uint, status asset.AssetStatus) error {
	if status != asset.StatusActive {
		if err := svc.ensureUnreferenced(ctx, id); err != nil {
			return err
		}
	}
	return svc.assetRepo.UpdateStatus(ctx, id, status)
}

func (svc *assetService) BatchUpdateStatus(ctx context.Context, ids []uint, status asset.AssetStatus) error {
	if status != asset.StatusActive {
		if err := svc.ensureUnreferenced(ctx, ids...); err != nil {
			return err
		}
	}
	return svc.assetRepo.BatchUpdateStatus(ctx, ids, status)
}

func (svc *assetService) Move(ctx context.Context, id uint, req *dto.AssetMoveReq) error {
	return svc.assetRepo.UpdateFolderPath(ctx, id, req.FolderPath)
}

// ========================
// 删除
// ========================

func (svc *assetService) Delete(ctx context.Context, id uint) error {
	return svc.BatchDelete(ctx, []uint{id})
}

func (svc *assetService) BatchDelete(ctx context.Context, ids []uint) error {
	items := make([]*asset.Asset, 0, len(ids))
	for _, id := range ids {
		item, err := svc.assetRepo.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("get asset %d: %w", id, err)
		}
		items = append(items, item)
	}
	if err := svc.ensureUnreferenced(ctx, ids...); err != nil {
		return err
	}

	// 先删记录再删对象：asset_references 的外键是最终裁决者，检查之后才登记的引用会让这里失败，
	// 此时文件必须原样保留。反过来，对象删除失败只会留下一个无记录的孤儿对象，不会弄坏业务数据。
	if err := svc.assetRepo.BatchDelete(ctx, ids); err != nil {
		if refErr := svc.ensureUnreferenced(ctx, ids...); refErr != nil {
			return refErr
		}
		return err
	}
	for _, item := range items {
		svc.removeObject(ctx, item.ObjectKey)
	}
	return nil
}

// ensureUnreferenced 是所有破坏性操作（删除、下线、取消公开）的共同前置检查。
func (svc *assetService) ensureUnreferenced(ctx context.Context, ids ...uint) error {
	referenced, err := svc.assetRepo.GetReferencedIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(referenced) > 0 {
		return apperror.ErrAssetInUse
	}
	return nil
}

// ========================
// 引用
// ========================

func (svc *assetService) Attach(ctx context.Context, objectKey string, ref asset.Reference, opts AttachOptions) error {
	if !ref.Valid() {
		return fmt.Errorf("attach asset: incomplete reference %+v", ref)
	}
	item, err := svc.assetRepo.GetByObjectKey(ctx, objectKey)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return apperror.ErrAssetNotFound
		}
		return err
	}

	if opts.RequireCategory != "" && item.Category != opts.RequireCategory {
		return apperror.ErrAssetInvalidType
	}
	if opts.RequirePublic {
		if item.Status != asset.StatusActive || isHighRiskExtension(strings.ToLower(item.Extension)) {
			return apperror.ErrAssetInvalidType
		}
		if !item.IsPublic {
			// 去重可能把业务上传对到一条私有资产上；公开是业务的显式要求，不能悄悄忽略。
			if err := svc.assetRepo.UpdateIsPublic(ctx, item.ID, true); err != nil {
				return err
			}
			svc.auditPublished(ctx, item, ref)
		}
	}
	return svc.assetRepo.AddReference(ctx, item.ID, ref, opts.DisplayName)
}

func (svc *assetService) Detach(ctx context.Context, objectKey string, ref asset.Reference) error {
	item, err := svc.assetRepo.GetByObjectKey(ctx, objectKey)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil
		}
		return err
	}
	if err := svc.assetRepo.RemoveReference(ctx, item.ID, ref); err != nil {
		return err
	}
	svc.collectIfOrphan(ctx, item)
	return nil
}

func (svc *assetService) Replace(ctx context.Context, ref asset.Reference, objectKey string) error {
	ids, err := svc.assetRepo.GetIDsByReference(ctx, ref)
	if err != nil {
		return err
	}
	for _, id := range ids {
		item, err := svc.assetRepo.Get(ctx, id)
		if err != nil {
			if errors.Is(err, shared.ErrNotFound) {
				continue
			}
			return err
		}
		if objectKey != "" && item.ObjectKey == objectKey {
			continue
		}
		if err := svc.assetRepo.RemoveReference(ctx, id, ref); err != nil {
			return err
		}
		svc.collectIfOrphan(ctx, item)
	}
	return nil
}

func (svc *assetService) DetachOwner(ctx context.Context, ownerType string, ownerID uint) error {
	ids, err := svc.assetRepo.GetIDsByOwner(ctx, ownerType, ownerID)
	if err != nil {
		return err
	}
	if err := svc.assetRepo.RemoveOwnerReferences(ctx, ownerType, ownerID); err != nil {
		return err
	}
	for _, id := range ids {
		item, err := svc.assetRepo.Get(ctx, id)
		if err != nil {
			continue
		}
		svc.collectIfOrphan(ctx, item)
	}
	return nil
}

// renewGracePeriod 让被去重复用的附件重新获得完整的宽限期：它可能是一条等待清扫的孤儿，
// 而调用方紧接着就要 Attach 它。
func (svc *assetService) renewGracePeriod(ctx context.Context, item *asset.Asset) {
	if item.Scope != asset.ScopeAttachment {
		return
	}
	if err := svc.assetRepo.Touch(ctx, item.ID); err != nil {
		log.WarnCtx(ctx).Err(err).Uint("assetID", item.ID).Msg("Failed to renew attachment grace period")
	}
}

// sweepBatchSize 单次清扫查询的上限，避免一次把海量孤儿读进内存
const sweepBatchSize = 200

func (svc *assetService) SweepOrphanAttachments(ctx context.Context, olderThan time.Duration) (int, error) {
	if olderThan <= 0 {
		return 0, nil
	}
	before := time.Now().Add(-olderThan)
	collected := 0
	for {
		items, err := svc.assetRepo.GetOrphanAttachments(ctx, before, sweepBatchSize)
		if err != nil {
			return collected, err
		}
		progressed := false
		for i := range items {
			// 外键兜底：查询之后才被 Attach 的附件删不掉，原样保留。
			if err := svc.assetRepo.Delete(ctx, items[i].ID); err != nil {
				log.WarnCtx(ctx).Err(err).Uint("assetID", items[i].ID).Msg("Failed to sweep orphan attachment")
				continue
			}
			svc.removeObject(ctx, items[i].ObjectKey)
			collected++
			progressed = true
		}
		// 不足一批说明已清完；整批都删不掉时也必须停下，否则会原地打转。
		if len(items) < sweepBatchSize || !progressed {
			return collected, nil
		}
	}
}

func (svc *assetService) Sync(ctx context.Context, ref asset.Reference, files []asset.AttachedFile, opts AttachOptions) error {
	keep := make(map[string]struct{}, len(files))
	for _, file := range files {
		if _, seen := keep[file.ObjectKey]; seen {
			continue
		}
		keep[file.ObjectKey] = struct{}{}
		opts.DisplayName = file.Name
		if err := svc.Attach(ctx, file.ObjectKey, ref, opts); err != nil {
			return err
		}
	}

	ids, err := svc.assetRepo.GetIDsByReference(ctx, ref)
	if err != nil {
		return err
	}
	for _, id := range ids {
		item, err := svc.assetRepo.Get(ctx, id)
		if err != nil {
			if errors.Is(err, shared.ErrNotFound) {
				continue
			}
			return err
		}
		if _, ok := keep[item.ObjectKey]; ok {
			continue
		}
		if err := svc.assetRepo.RemoveReference(ctx, id, ref); err != nil {
			return err
		}
		svc.collectIfOrphan(ctx, item)
	}
	return nil
}

func (svc *assetService) ListAttached(ctx context.Context, field asset.Field, ownerIDs []uint) (map[uint][]dto.AssetAttachmentResp, error) {
	attached, err := svc.assetRepo.GetAttached(ctx, field, ownerIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[uint][]dto.AssetAttachmentResp, len(ownerIDs))
	for i := range attached {
		var resp dto.AssetAttachmentResp
		resp.FromEntity(&attached[i].Asset, attached[i].Name)
		result[attached[i].OwnerID] = append(result[attached[i].OwnerID], resp)
	}
	return result, nil
}

// collectIfOrphan 回收失去全部引用的业务附件。回收失败不影响业务操作：
// 残留的附件仍可在「资产管理」中按范围筛出并手动删除。
func (svc *assetService) collectIfOrphan(ctx context.Context, item *asset.Asset) {
	if item.Scope != asset.ScopeAttachment {
		return
	}
	count, err := svc.assetRepo.CountReferences(ctx, item.ID)
	if err != nil || count > 0 {
		return
	}
	// 与 BatchDelete 同理先删记录：并发的 Attach 会被外键拦下，附件得以保留。
	if err := svc.assetRepo.Delete(ctx, item.ID); err != nil {
		log.WarnCtx(ctx).Err(err).Uint("assetID", item.ID).Msg("Failed to collect orphan attachment record")
		return
	}
	svc.removeObject(ctx, item.ObjectKey)
}

func (svc *assetService) auditPublished(ctx context.Context, item *asset.Asset, ref asset.Reference) {
	if svc.auditLog == nil {
		return
	}
	actor := ucontext.ActorFromContext(ctx)
	svc.auditLog.LogAsync(&audit_log.AuditLog{
		LogType:    audit_log.AuditLogTypeAssetUpdate,
		Operator:   actor.Username,
		OperatorID: actor.UserID,
		IpAddr:     actor.IP,
		Target:     item.ObjectKey,
		Details:    fmt.Sprintf("Asset ID %d made public by reference %s.%s", item.ID, ref.OwnerType, ref.Field),
		Success:    true,
	})
}

func (svc *assetService) removeObject(ctx context.Context, objectKey string) {
	if err := svc.storage.Delete(ctx, objectKey); err != nil {
		log.WarnCtx(ctx).Err(err).Str("objectKey", objectKey).Msg("Failed to delete asset object from storage")
	}
}

// ========================
// 下载
// ========================

func (svc *assetService) PrepareDownload(ctx context.Context, objectKey string, requirePublic bool) (*AssetDownloadResult, error) {
	item, err := svc.assetRepo.GetByObjectKey(ctx, objectKey)
	if err != nil {
		return nil, apperror.ErrAssetNotFound
	}

	if requirePublic && (!item.IsPublic || item.Status != asset.StatusActive || isHighRiskExtension(strings.ToLower(item.Extension))) {
		return nil, apperror.ErrAssetNotFound
	}

	// 业务附件的原始文件名是终端用户的输入（可能含个人信息），不随免认证下载外泄。
	filename := item.Filename
	if filename == "" || (requirePublic && item.Scope == asset.ScopeAttachment) {
		filename = item.ObjectKey
	}
	return svc.download(ctx, item, filename), nil
}

func (svc *assetService) download(ctx context.Context, item *asset.Asset, filename string) *AssetDownloadResult {
	if err := svc.assetRepo.IncrementDownloadCount(ctx, item.ID); err != nil {
		log.WarnCtx(ctx).Err(err).Uint("assetID", item.ID).Msg("Failed to increment asset download count")
	}
	contentType := item.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return &AssetDownloadResult{
		ObjectKey:   item.ObjectKey,
		Filename:    filename,
		ContentType: contentType,
		Size:        item.Size,
	}
}

func (svc *assetService) PrepareAttachedDownload(ctx context.Context, objectKey string, ref asset.Reference) (*AssetDownloadResult, error) {
	item, err := svc.assetRepo.GetByObjectKey(ctx, objectKey)
	if err != nil {
		return nil, apperror.ErrAssetNotFound
	}
	name, err := svc.assetRepo.GetReferenceName(ctx, item.ID, ref)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, apperror.ErrAssetNotFound
		}
		return nil, err
	}
	if item.Status != asset.StatusActive {
		return nil, apperror.ErrAssetNotFound
	}
	return svc.download(ctx, item, attachmentName(item, name)), nil
}

// attachmentName 附件对引用方显示的名字：引用自己登记的显示名优先。
func attachmentName(item *asset.Asset, referenceName string) string {
	if referenceName != "" {
		return referenceName
	}
	if item.Filename != "" {
		return item.Filename
	}
	return item.ObjectKey
}

func (svc *assetService) WriteContent(ctx context.Context, objectKey string, w io.Writer) error {
	if err := svc.storage.Get(ctx, objectKey, w); err != nil {
		return fmt.Errorf("download asset %s: %w", objectKey, err)
	}
	return nil
}

// ========================
// 辅助方法
// ========================

// calculateFileHash 计算文件 SHA-256 哈希（用于内容去重；hex 编码长度 64）
func calculateFileHash(f multipart.File) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func inspectUploadedFile(f multipart.File) (string, int64, error) {
	seeker, ok := f.(io.Seeker)
	if !ok {
		return "", 0, fmt.Errorf("uploaded file is not seekable")
	}
	if _, err := seeker.Seek(0, io.SeekStart); err != nil {
		return "", 0, err
	}
	header := make([]byte, 512)
	n, readErr := io.ReadFull(f, header)
	if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
		return "", 0, readErr
	}
	if _, err := seeker.Seek(0, io.SeekEnd); err != nil {
		return "", 0, err
	}
	pos, err := seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return "", 0, err
	}
	if _, err := seeker.Seek(0, io.SeekStart); err != nil {
		return "", 0, err
	}
	return http.DetectContentType(header[:n]), pos, nil
}

func contentTypeMatchesExtension(ext, detected string) bool {
	if detected == "" || detected == "application/octet-stream" {
		return true
	}
	switch ext {
	case ".jpg", ".jpeg":
		return detected == "image/jpeg"
	case ".png":
		return detected == "image/png"
	case ".gif":
		return detected == "image/gif"
	case ".webp":
		return detected == "image/webp"
	case ".svg":
		return strings.HasPrefix(detected, "text/") || detected == "application/xml"
	case ".pdf":
		return detected == "application/pdf"
	}
	return true
}

func isHighRiskExtension(ext string) bool {
	switch ext {
	case ".svg", ".zip", ".rar", ".7z", ".tar", ".gz", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
		return true
	default:
		return false
	}
}

// detectCategory 根据 MIME 类型和扩展名检测资产分类
func detectCategory(mimeType, ext string) asset.AssetCategory {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return asset.CategoryImage
	case strings.HasPrefix(mimeType, "video/"):
		return asset.CategoryVideo
	case strings.HasPrefix(mimeType, "audio/"):
		return asset.CategoryAudio
	case strings.HasPrefix(mimeType, "application/pdf"),
		strings.HasPrefix(mimeType, "application/msword"),
		strings.HasPrefix(mimeType, "application/vnd.openxmlformats-officedocument"),
		strings.HasPrefix(mimeType, "application/vnd.ms-"),
		strings.HasPrefix(mimeType, "text/"):
		return asset.CategoryDocument
	case strings.HasPrefix(mimeType, "application/zip"),
		strings.HasPrefix(mimeType, "application/x-rar"),
		strings.HasPrefix(mimeType, "application/x-7z"),
		strings.HasPrefix(mimeType, "application/gzip"),
		strings.HasPrefix(mimeType, "application/x-tar"):
		return asset.CategoryArchive
	}

	ext = strings.ToLower(ext)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".ico":
		return asset.CategoryImage
	case ".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv", ".webm":
		return asset.CategoryVideo
	case ".mp3", ".wav", ".flac", ".aac", ".ogg", ".wma":
		return asset.CategoryAudio
	case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".md", ".csv":
		return asset.CategoryDocument
	case ".zip", ".rar", ".7z", ".tar", ".gz", ".bz2":
		return asset.CategoryArchive
	}

	return asset.CategoryOther
}

// formatSize 格式化文件大小
func formatSize(size int64) string {
	const (
		B  = 1
		KB = 1024 * B
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
