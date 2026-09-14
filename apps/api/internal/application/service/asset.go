package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/asset"
	"github.com/castorworks/castor/internal/pkg/generator"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/hyperits/gosuite/logger"
	"github.com/hyperits/gosuite/storage/s3"
)

// AssetDownloadResult 下载结果（供 Handler 层使用）
type AssetDownloadResult struct {
	ObjectKey   string
	Bucket      string
	Filename    string
	ContentType string
	Size        int64
}

// AssetService 资产应用服务接口
type AssetService interface {
	// ========================
	// 上传与创建
	// ========================

	// Upload 上传资产
	Upload(ctx context.Context, req *dto.AssetUploadReq, filename string, f multipart.File, contentType string, size int64) (*dto.AssetUploadResp, error)

	// ========================
	// 查询
	// ========================

	// Get 根据 ID 获取资产
	Get(ctx context.Context, id uint) (*dto.AssetResp, error)

	// GetByObjectKey 根据对象键获取资产
	GetByObjectKey(ctx context.Context, objectKey string) (*dto.AssetResp, error)

	// Gets 分页查询资产列表
	Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]dto.AssetResp, int64, error)

	// GetsByCategory 按分类查询资产列表
	GetsByCategory(ctx context.Context, category asset.AssetCategory, page, size int, order string) ([]dto.AssetResp, int64, error)

	// GetsByStatus 按状态查询资产列表
	GetsByStatus(ctx context.Context, status asset.AssetStatus, page, size int, order string) ([]dto.AssetResp, int64, error)

	// GetsByFolderPath 按文件夹路径查询资产列表
	GetsByFolderPath(ctx context.Context, folderPath string, page, size int, order string) ([]dto.AssetResp, int64, error)

	// GetPublicAssets 获取公开资产列表
	GetPublicAssets(ctx context.Context, page, size int, order string) ([]dto.AssetResp, int64, error)

	// GetStats 获取资产统计信息
	GetStats(ctx context.Context) (*dto.AssetStatsResp, error)

	// ========================
	// 更新
	// ========================

	// Update 更新资产信息
	Update(ctx context.Context, id uint, req *dto.AssetUpdateReq) (*dto.AssetResp, error)

	// UpdateStatus 更新资产状态
	UpdateStatus(ctx context.Context, id uint, status asset.AssetStatus) error

	// Move 移动资产到指定文件夹
	Move(ctx context.Context, id uint, req *dto.AssetMoveReq) error

	// ========================
	// 删除
	// ========================

	// Delete 删除资产
	Delete(ctx context.Context, id uint) error

	// DeleteByObjectKey 根据对象键删除资产
	DeleteByObjectKey(ctx context.Context, objectKey string) error

	// BatchDelete 批量删除资产
	BatchDelete(ctx context.Context, ids []uint) error

	// ========================
	// 批量操作
	// ========================

	// BatchUpdateStatus 批量更新资产状态
	BatchUpdateStatus(ctx context.Context, ids []uint, status asset.AssetStatus) error

	// ========================
	// 下载
	// ========================

	// PrepareDownload 准备下载（返回下载所需信息，不直接操作 HTTP）
	PrepareDownload(ctx context.Context, objectKey string, requirePublic bool) (*AssetDownloadResult, error)

	// GetS3Client 获取 S3 客户端（供 Handler 层流式下载使用）
	GetS3Client() *s3.S3Client
}
type assetService struct {
	s3        *s3.S3Client
	assetRepo asset.Repository
	policy    AssetPolicy
}

// NewAssetService 创建资产应用服务
func NewAssetService(s3Client *s3.S3Client, assetRepo asset.Repository, policy AssetPolicy) AssetService {
	return &assetService{
		s3:        s3Client,
		assetRepo: assetRepo,
		policy:    policy,
	}
}

// ========================
// 上传与创建
// ========================

func (svc *assetService) Upload(ctx context.Context, req *dto.AssetUploadReq, filename string, f multipart.File, contentType string, size int64) (*dto.AssetUploadResp, error) {
	// 验证文件大小
	maxSize := svc.policy.MaxUploadSize
	if maxSize > 0 && size > maxSize {
		return nil, apperror.ErrAssetTooLarge
	}

	// 验证文件扩展名
	ext := strings.ToLower(filepath.Ext(filename))
	if len(svc.policy.AllowedExtensions) > 0 && !svc.isExtensionAllowed(ext) {
		return nil, apperror.ErrAssetInvalidType
	}

	// Do not trust the multipart Content-Type or declared size. Inspect the
	// file bytes and obtain the actual size from the stream before storing it.
	detectedType, actualSize, err := inspectUploadedFile(f)
	if err != nil {
		return nil, apperror.ErrAssetInvalidType
	}
	if actualSize > 0 {
		size = actualSize
	}
	if maxSize > 0 && size > maxSize {
		return nil, apperror.ErrAssetTooLarge
	}
	if !contentTypeMatchesExtension(ext, detectedType) {
		return nil, apperror.ErrAssetInvalidType
	}
	// SVG and archive formats can carry active content or resource exhaustion
	// payloads; never expose them publicly without an asynchronous scan.
	if isHighRiskExtension(ext) {
		req.IsPublic = false
	}
	if detectedType != "" {
		contentType = detectedType
	}

	objectKey := generator.GenerateShortUUIDString() + ext

	// 计算文件哈希
	hash, err := svc.calculateFileHash(f)
	if err != nil {
		return nil, err
	}

	// 重置文件读取位置
	if seeker, ok := f.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
	}

	// 检查是否存在相同哈希的文件（去重）
	existingAsset, err := svc.assetRepo.GetByHash(ctx, hash)
	if err == nil && existingAsset != nil {
		resp := &dto.AssetUploadResp{
			IsDuplicate: true,
		}
		if err := resp.AssetResp.FromEntity(existingAsset); err != nil {
			return nil, err
		}
		return resp, nil
	}

	// 确定资产名称
	assetName := req.Name
	if assetName == "" {
		assetName = strings.TrimSuffix(filename, filepath.Ext(filename))
	}

	// 先上传到 S3（失败不会留下脏数据）
	err = svc.s3.UploadObject(ctx, svc.s3.Config().Bucket, objectKey, f, size, contentType)
	if err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	// 创建资产记录
	assetInfo := &asset.Asset{
		Name:        assetName,
		Filename:    filename,
		Description: req.Description,
		Tags:        req.Tags,
		FolderPath:  req.FolderPath,
		ObjectKey:   objectKey,
		StorageType: asset.StorageS3,
		Bucket:      svc.s3.Config().Bucket,
		Extension:   ext,
		MimeType:    contentType,
		Size:        size,
		Hash:        hash,
		Category:    svc.detectCategory(contentType, ext),
		Status:      asset.StatusActive,
		IsPublic:    req.IsPublic,
	}

	if err := svc.assetRepo.Create(ctx, assetInfo); err != nil {
		// 清理已上传的 S3 对象
		_ = svc.s3.DeleteObject(ctx, svc.s3.Config().Bucket, objectKey)
		return nil, err
	}

	resp := &dto.AssetUploadResp{
		IsDuplicate: false,
	}
	if err := resp.AssetResp.FromEntity(assetInfo); err != nil {
		return nil, err
	}

	return resp, nil
}

// ========================
// 查询
// ========================

func (svc *assetService) Get(ctx context.Context, id uint) (*dto.AssetResp, error) {
	item, err := svc.assetRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := &dto.AssetResp{}
	if err := resp.FromEntity(item); err != nil {
		return nil, err
	}
	return resp, nil
}

func (svc *assetService) GetByObjectKey(ctx context.Context, objectKey string) (*dto.AssetResp, error) {
	item, err := svc.assetRepo.GetByObjectKey(ctx, objectKey)
	if err != nil {
		return nil, err
	}

	resp := &dto.AssetResp{}
	if err := resp.FromEntity(item); err != nil {
		return nil, err
	}
	return resp, nil
}

func (svc *assetService) Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]dto.AssetResp, int64, error) {
	items, total, err := svc.assetRepo.Gets(ctx, page, size, order, opts...)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]dto.AssetResp, len(items))
	for i, item := range items {
		if err := resp[i].FromEntity(&item); err != nil {
			return nil, 0, err
		}
	}
	return resp, total, nil
}
func (svc *assetService) GetsByCategory(ctx context.Context, category asset.AssetCategory, page, size int, order string) ([]dto.AssetResp, int64, error) {
	items, total, err := svc.assetRepo.GetsByCategory(ctx, category, page, size, order)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]dto.AssetResp, len(items))
	for i, item := range items {
		if err := resp[i].FromEntity(&item); err != nil {
			return nil, 0, err
		}
	}
	return resp, total, nil
}

func (svc *assetService) GetsByStatus(ctx context.Context, status asset.AssetStatus, page, size int, order string) ([]dto.AssetResp, int64, error) {
	items, total, err := svc.assetRepo.GetsByStatus(ctx, status, page, size, order)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]dto.AssetResp, len(items))
	for i, item := range items {
		if err := resp[i].FromEntity(&item); err != nil {
			return nil, 0, err
		}
	}
	return resp, total, nil
}

func (svc *assetService) GetsByFolderPath(ctx context.Context, folderPath string, page, size int, order string) ([]dto.AssetResp, int64, error) {
	items, total, err := svc.assetRepo.GetsByFolderPath(ctx, folderPath, page, size, order)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]dto.AssetResp, len(items))
	for i, item := range items {
		if err := resp[i].FromEntity(&item); err != nil {
			return nil, 0, err
		}
	}
	return resp, total, nil
}
func (svc *assetService) GetPublicAssets(ctx context.Context, page, size int, order string) ([]dto.AssetResp, int64, error) {
	items, total, err := svc.assetRepo.GetPublicAssets(ctx, page, size, order)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]dto.AssetResp, len(items))
	for i, item := range items {
		if err := resp[i].FromEntity(&item); err != nil {
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
	if req.IsPublic != nil {
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
	return svc.assetRepo.UpdateStatus(ctx, id, status)
}

func (svc *assetService) Move(ctx context.Context, id uint, req *dto.AssetMoveReq) error {
	return svc.assetRepo.MoveToFolder(ctx, id, req.FolderID, req.FolderPath)
}

// ========================
// 删除
// ========================

func (svc *assetService) Delete(ctx context.Context, id uint) error {
	item, err := svc.assetRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	if err := svc.s3.DeleteObject(ctx, svc.s3.Config().Bucket, item.ObjectKey); err != nil {
		return err
	}

	// 删除缩略图（如果存在）
	if item.ThumbnailKey != "" {
		_ = svc.s3.DeleteObject(ctx, svc.s3.Config().Bucket, item.ThumbnailKey)
	}

	return svc.assetRepo.Delete(ctx, id)
}

func (svc *assetService) DeleteByObjectKey(ctx context.Context, objectKey string) error {
	item, err := svc.assetRepo.GetByObjectKey(ctx, objectKey)
	if err != nil {
		return err
	}

	if err := svc.s3.DeleteObject(ctx, svc.s3.Config().Bucket, item.ObjectKey); err != nil {
		return err
	}

	return svc.assetRepo.DeleteByObjectKey(ctx, objectKey)
}
func (svc *assetService) BatchDelete(ctx context.Context, ids []uint) error {
	// 批量查询所有资产
	items := make([]*asset.Asset, 0, len(ids))
	for _, id := range ids {
		item, err := svc.assetRepo.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("get asset %d: %w", id, err)
		}
		items = append(items, item)
	}

	// 批量删除 S3 对象
	for _, item := range items {
		if err := svc.s3.DeleteObject(ctx, svc.s3.Config().Bucket, item.ObjectKey); err != nil {
			return fmt.Errorf("delete s3 object %s: %w", item.ObjectKey, err)
		}
		if item.ThumbnailKey != "" {
			_ = svc.s3.DeleteObject(ctx, svc.s3.Config().Bucket, item.ThumbnailKey)
		}
	}

	// 批量收集 objectKeys 用于数据库批量删除
	objectKeys := make([]string, len(items))
	for i, item := range items {
		objectKeys[i] = item.ObjectKey
	}

	return svc.assetRepo.BatchDeleteByObjectKeys(ctx, objectKeys)
}

// ========================
// 批量操作
// ========================

func (svc *assetService) BatchUpdateStatus(ctx context.Context, ids []uint, status asset.AssetStatus) error {
	return svc.assetRepo.BatchUpdateStatus(ctx, ids, status)
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

	// 增加下载次数
	if err := svc.assetRepo.IncrementDownloadCount(ctx, item.ID); err != nil {
		logger.Warnf("Increment download count of asset %d failed: %v", item.ID, err)
	}

	contentType := item.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	filename := item.Filename
	if filename == "" {
		filename = objectKey
	}

	return &AssetDownloadResult{
		ObjectKey:   item.ObjectKey,
		Bucket:      svc.s3.Config().Bucket,
		Filename:    filename,
		ContentType: contentType,
		Size:        item.Size,
	}, nil
}

func (svc *assetService) GetS3Client() *s3.S3Client {
	return svc.s3
}

// ========================
// 辅助方法
// ========================

// calculateFileHash 计算文件 SHA-256 哈希（用于内容去重；hex 编码长度 64）
func (svc *assetService) calculateFileHash(f multipart.File) (string, error) {
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

// isExtensionAllowed 检查文件扩展名是否在允许列表中
func (svc *assetService) isExtensionAllowed(ext string) bool {
	for _, allowed := range svc.policy.AllowedExtensions {
		if strings.EqualFold(ext, allowed) {
			return true
		}
	}
	return false
}

// detectCategory 根据 MIME 类型和扩展名检测资产分类
func (svc *assetService) detectCategory(mimeType, ext string) asset.AssetCategory {
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
