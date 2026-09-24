package service

import (
	"context"
	"mime/multipart"

	"github.com/castorworks/castor/internal/domain/asset"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/log"
)

// 头像是资产能力的参考使用者：users.avatar 存资产对象键，
// 引用登记为 {user, <id>, avatar}，前端经免认证的资产下载接口展示。
const (
	// UserAssetOwnerType 用户作为资产引用方的标识
	UserAssetOwnerType = "user"
	// UserAvatarField 用户头像字段
	UserAvatarField = "avatar"
	// AvatarMaxSize 头像最大尺寸 (2MB)
	AvatarMaxSize int64 = 2 * 1024 * 1024
)

// avatarPolicy 头像字段的上传限制：只接受真实内容为位图的小文件
var avatarPolicy = AssetPolicy{
	MaxUploadSize:     AvatarMaxSize,
	AllowedExtensions: []string{".jpg", ".jpeg", ".png", ".gif", ".webp"},
	AllowedMimeTypes:  []string{"image/jpeg", "image/png", "image/gif", "image/webp"},
}

// avatarAttachOptions 头像要在登录页之外的任何地方展示，必须是可公开访问的图片
var avatarAttachOptions = AttachOptions{RequirePublic: true, RequireCategory: asset.CategoryImage}

// UserAvatarRef 返回用户头像字段对资产的引用
func UserAvatarRef(userID uint) asset.Reference {
	return asset.Reference{OwnerType: UserAssetOwnerType, OwnerID: userID, Field: UserAvatarField}
}

// UserAvatarResp 用户头像响应
type UserAvatarResp struct {
	ObjectKey string `json:"objectKey"` // 资产对象键
	URL       string `json:"url"`       // 免认证下载地址
}

// UserAvatarService 用户头像服务接口
type UserAvatarService interface {
	// Upload 上传文件并设为用户头像，旧头像随引用解除而回收
	Upload(ctx context.Context, userID uint, filename string, f multipart.File, contentType string, size int64) (*UserAvatarResp, error)
}

type userAvatarService struct {
	assets   AssetService
	userRepo user.Repository
}

// NewUserAvatarService 创建用户头像服务
func NewUserAvatarService(assets AssetService, userRepo user.Repository) UserAvatarService {
	return &userAvatarService{assets: assets, userRepo: userRepo}
}

func (svc *userAvatarService) Upload(ctx context.Context, userID uint, filename string, f multipart.File, contentType string, size int64) (*UserAvatarResp, error) {
	item, err := svc.userRepo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	uploaded, err := svc.assets.UploadAttachment(ctx, AttachmentUpload{
		Filename:    filename,
		File:        f,
		ContentType: contentType,
		Size:        size,
		IsPublic:    true,
		Policy:      avatarPolicy,
	})
	if err != nil {
		return nil, err
	}

	err = changeUserAvatar(ctx, svc.assets, item, uploaded.ObjectKey, func() error {
		return svc.userRepo.Update(ctx, item)
	})
	if err != nil {
		return nil, err
	}
	return &UserAvatarResp{ObjectKey: uploaded.ObjectKey, URL: uploaded.URL}, nil
}

// changeUserAvatar 把 item 的头像换成 objectKey 指向的资产，save 负责持久化用户记录。
// 顺序是「先登记新引用 → 保存用户 → 再解除旧引用」：任何一步失败，旧头像都还在。
func changeUserAvatar(ctx context.Context, assets AssetReferencer, item *user.User, objectKey string, save func() error) error {
	ref := UserAvatarRef(item.ID)
	previous := item.Avatar
	// 放弃 objectKey：解除可能已登记的引用；它若是刚上传、再无人引用的附件，会随之被回收。
	// 与现有头像是同一份文件时什么都不能动。
	abandon := func() {
		if objectKey == previous {
			return
		}
		if err := assets.Detach(ctx, objectKey, ref); err != nil {
			log.WarnCtx(ctx).Err(err).Str("objectKey", objectKey).Msg("Failed to roll back avatar reference")
		}
	}

	if err := assets.Attach(ctx, objectKey, ref, avatarAttachOptions); err != nil {
		abandon()
		return err
	}

	item.Avatar = objectKey
	if err := save(); err != nil {
		item.Avatar = previous
		abandon()
		return err
	}

	if err := assets.Replace(ctx, ref, objectKey); err != nil {
		log.WarnCtx(ctx).Err(err).Uint("userID", item.ID).Msg("Failed to release previous avatar")
	}
	return nil
}
