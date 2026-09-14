package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/pkg/generator"
	"github.com/hyperits/gosuite/storage/s3"
)

const (
	// AvatarPrefix 头像存储路径前缀
	AvatarPrefix = "avatars/"
	// AvatarMaxSize 头像最大尺寸 (2MB)
	AvatarMaxSize int64 = 2 * 1024 * 1024
)

// 允许的头像 MIME 类型
var allowedAvatarMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// 允许的头像扩展名
var allowedAvatarExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

// UserAvatarResp 用户头像响应
type UserAvatarResp struct {
	ObjectKey string `json:"objectKey"` // 存储对象键
	URL       string `json:"url"`       // 访问 URL（可选，如果配置了 CDN）
}

// UserAvatarService 用户头像服务接口
type UserAvatarService interface {
	// Upload 上传用户头像
	// 返回存储的 ObjectKey
	Upload(ctx context.Context, filename string, f multipart.File, contentType string, size int64) (*UserAvatarResp, error)

	// Delete 删除用户头像
	Delete(ctx context.Context, objectKey string) error

	// Download 将头像内容写入 w；非头像路径返回 apperror.ErrAssetNotFound
	Download(ctx context.Context, objectKey string, w io.Writer) error

	// ValidateFile 验证头像文件
	ValidateFile(filename string, contentType string, size int64) error
}

type userAvatarService struct {
	s3 *s3.S3Client
}

// NewUserAvatarService 创建用户头像服务
func NewUserAvatarService(s3Client *s3.S3Client) UserAvatarService {
	return &userAvatarService{
		s3: s3Client,
	}
}

// ValidateFile 验证头像文件
func (svc *userAvatarService) ValidateFile(filename string, contentType string, size int64) error {
	// 验证文件大小
	if size > AvatarMaxSize {
		return fmt.Errorf("avatar file size exceeds limit (max: %d bytes)", AvatarMaxSize)
	}

	// 验证 MIME 类型
	if !allowedAvatarMimeTypes[contentType] {
		return fmt.Errorf("invalid avatar content type: %s, allowed: jpeg, png, gif, webp", contentType)
	}

	// 验证扩展名
	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedAvatarExtensions[ext] {
		return fmt.Errorf("invalid avatar extension: %s, allowed: .jpg, .jpeg, .png, .gif, .webp", ext)
	}

	return nil
}

// Upload 上传用户头像
func (svc *userAvatarService) Upload(ctx context.Context, filename string, f multipart.File, contentType string, size int64) (*UserAvatarResp, error) {
	// 验证文件
	if err := svc.ValidateFile(filename, contentType, size); err != nil {
		return nil, err
	}

	// 生成唯一的对象键
	ext := filepath.Ext(filename)
	objectKey := AvatarPrefix + generator.GenerateShortUUIDString() + ext

	// 上传到 S3
	err := svc.s3.UploadObject(ctx, svc.s3.Config().Bucket, objectKey, f, size, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload avatar: %w", err)
	}

	return &UserAvatarResp{
		ObjectKey: objectKey,
	}, nil
}

// Delete 删除用户头像
func (svc *userAvatarService) Delete(ctx context.Context, objectKey string) error {
	// 验证是否为头像路径
	if objectKey == "" {
		return nil // 空路径不需要删除
	}

	if !strings.HasPrefix(objectKey, AvatarPrefix) {
		return fmt.Errorf("invalid avatar object key: %s", objectKey)
	}

	// 从 S3 删除
	if err := svc.s3.DeleteObject(ctx, svc.s3.Config().Bucket, objectKey); err != nil {
		return fmt.Errorf("failed to delete avatar: %w", err)
	}

	return nil
}

// Download 将头像内容写入 w
func (svc *userAvatarService) Download(ctx context.Context, objectKey string, w io.Writer) error {
	// 只允许访问头像路径，避免借公开头像接口读取其他对象
	if !strings.HasPrefix(objectKey, AvatarPrefix) {
		return apperror.ErrAssetNotFound
	}
	if err := svc.s3.GetObject(ctx, svc.s3.Config().Bucket, objectKey, w); err != nil {
		return fmt.Errorf("download avatar %s: %w", objectKey, err)
	}
	return nil
}

// GetDownloadURL 获取头像下载 URL（如果需要生成签名 URL）
func (svc *userAvatarService) GetDownloadURL(ctx context.Context, objectKey string) (string, error) {
	if objectKey == "" {
		return "", nil
	}

	if !strings.HasPrefix(objectKey, AvatarPrefix) {
		return "", fmt.Errorf("invalid avatar object key: %s", objectKey)
	}

	// 如果需要签名 URL，可以在这里生成
	// 目前返回相对路径，由前端拼接完整 URL
	return "/api/v1/avatars/" + objectKey, nil
}
