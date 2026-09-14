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

// AssetStatus 资产状态
type AssetStatus string

const (
	StatusPending  AssetStatus = "PENDING"
	StatusActive   AssetStatus = "ACTIVE"
	StatusArchived AssetStatus = "ARCHIVED"
	StatusDeleted  AssetStatus = "DELETED"
)

// StorageType 存储类型
type StorageType string

const (
	StorageLocal StorageType = "LOCAL"
	StorageS3    StorageType = "S3"
	StorageOSS   StorageType = "OSS"
	StorageCOS   StorageType = "COS"
)

// Asset 资产实体（纯 Domain 模型，无 ORM 依赖）
// GORM 标签和生命周期钩子已移至 infrastructure/persistence/models/。
type Asset struct {
	shared.BaseModel

	Name        string `json:"name"`
	Filename    string `json:"filename"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	FolderID    *uint  `json:"folderId"`
	FolderPath  string `json:"folderPath"`

	ObjectKey   string      `json:"objectKey"`
	StorageType StorageType `json:"storageType"`
	Bucket      string      `json:"bucket"`

	Extension string `json:"extension"`
	MimeType  string `json:"mimeType"`
	Size      int64  `json:"size"`
	Hash      string `json:"hash"`

	Category AssetCategory `json:"category"`
	Status   AssetStatus   `json:"status"`
	IsPublic bool          `json:"isPublic"`

	Width    int `json:"width"`
	Height   int `json:"height"`
	Duration int `json:"duration"`

	ThumbnailKey  string `json:"thumbnailKey"`
	DownloadCount int    `json:"downloadCount"`
	ViewCount     int    `json:"viewCount"`
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
