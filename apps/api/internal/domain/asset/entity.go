package asset

import (
	"fmt"
	"strings"

	"github.com/castorworks/castor/internal/domain/shared"
)

// AssetCategory 资产分类
type AssetCategory string

const (
	CategoryImage    AssetCategory = "IMAGE"
	CategoryVideo    AssetCategory = "VIDEO"
	CategoryAudio    AssetCategory = "AUDIO"
	CategoryDocument AssetCategory = "DOCUMENT"
	CategoryArchive  AssetCategory = "ARCHIVE"
	CategoryOther    AssetCategory = "OTHER"
)

// AssetStatus 资产状态。只有两种：生效，或已归档（保留文件但不再对外提供）。
// 删除就是真删除，不设"已删除"状态；也没有审核流程，不设"待处理"。
type AssetStatus string

const (
	StatusActive   AssetStatus = "ACTIVE"
	StatusArchived AssetStatus = "ARCHIVED"
)

// AssetScope 资产归属范围：决定资产的生命周期由谁管理
type AssetScope string

const (
	// ScopeLibrary 资产库资产：由运营在「资产管理」中上传，长期保留，只能手动删除
	ScopeLibrary AssetScope = "LIBRARY"
	// ScopeAttachment 业务附件：由业务模块上传，失去最后一个引用时自动回收
	ScopeAttachment AssetScope = "ATTACHMENT"
)

// Reference 业务对象对资产的一次引用。
// (OwnerType, OwnerID, Field) 标识业务记录上的一个文件字段，例如 {"user", 42, "avatar"}。
// 被引用的资产不可删除，哈希去重因此才是安全的：多条业务记录可以共用同一份文件。
type Reference struct {
	OwnerType string `json:"ownerType"` // 业务模块标识，小写蛇形，如 "user"
	OwnerID   uint   `json:"ownerId"`   // 业务记录 ID
	Field     string `json:"field"`     // 业务记录上的字段名，如 "avatar"
}

// Field 标识一类业务记录上的一个文件字段（不指定具体记录），用于批量读取，如 {"notification", "attachments"}。
type Field struct {
	OwnerType string
	Name      string
}

// Of 返回该字段在某条业务记录上的引用
func (f Field) Of(ownerID uint) Reference {
	return Reference{OwnerType: f.OwnerType, OwnerID: ownerID, Field: f.Name}
}

// Attached 挂在某条业务记录上的资产
type Attached struct {
	OwnerID uint
	// Name 引用方给这个文件起的显示名；为空时退回资产自己的文件名
	Name  string
	Asset Asset
}

// AttachedFile 业务记录要引用的一个文件
type AttachedFile struct {
	ObjectKey string
	// Name 显示名（通常是上传者本地的文件名）。去重会让多个上传共用一条资产记录，
	// 资产自己的名字属于最早的上传者，所以每个引用各自记名，互不可见。
	Name string
}

// Valid 校验引用的三个组成部分均已填写
func (r Reference) Valid() bool {
	return r.OwnerType != "" && r.OwnerID != 0 && r.Field != ""
}

// Asset 资产实体（纯 Domain 模型，无 ORM 依赖）
// GORM 标签和生命周期钩子已移至 infrastructure/persistence/models/。
type Asset struct {
	shared.BaseModel

	Name        string `json:"name"`
	Filename    string `json:"filename"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	FolderPath  string `json:"folderPath"`

	ObjectKey string `json:"objectKey"`
	Bucket    string `json:"bucket"`

	Extension string `json:"extension"`
	MimeType  string `json:"mimeType"`
	Size      int64  `json:"size"`
	Hash      string `json:"hash"`

	Category AssetCategory `json:"category"`
	Status   AssetStatus   `json:"status"`
	Scope    AssetScope    `json:"scope"`
	IsPublic bool          `json:"isPublic"`

	DownloadCount int `json:"downloadCount"`
}

// GetSizeFormatted 获取格式化的文件大小
func (a *Asset) GetSizeFormatted() string {
	const (
		B  = 1
		KB = 1024 * B
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case a.Size >= GB:
		return fmtSize(float64(a.Size)/GB, "GB")
	case a.Size >= MB:
		return fmtSize(float64(a.Size)/MB, "MB")
	case a.Size >= KB:
		return fmtSize(float64(a.Size)/KB, "KB")
	default:
		return fmtSize(float64(a.Size), "B")
	}
}

func fmtSize(size float64, unit string) string {
	if size == float64(int(size)) {
		return fmt.Sprintf("%d %s", int(size), unit)
	}
	formatted := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", size), "0"), ".")
	return fmt.Sprintf("%s %s", formatted, unit)
}

func (a *Asset) IsImage() bool    { return a.Category == CategoryImage }
func (a *Asset) IsVideo() bool    { return a.Category == CategoryVideo }
func (a *Asset) IsAudio() bool    { return a.Category == CategoryAudio }
func (a *Asset) IsDocument() bool { return a.Category == CategoryDocument }
