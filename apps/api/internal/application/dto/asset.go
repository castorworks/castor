package dto

import (
	"time"

	"github.com/castorworks/castor/internal/domain/asset"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/jinzhu/copier"
)

// ======================================
// Request DTOs
// ======================================

// AssetUploadReq 资产上传请求（通过 form-data 提交，文件字段为 file）
type AssetUploadReq struct {
	Name        string `form:"name"`        // 资产名称（可选，默认使用原始文件名）
	Description string `form:"description"` // 描述信息
	Tags        string `form:"tags"`        // 标签（逗号分隔）
	FolderPath  string `form:"folderPath"`  // 虚拟文件夹路径
	IsPublic    bool   `form:"isPublic"`    // 是否公开
}

// AssetUpdateReq 资产更新请求
// 使用指针类型以区分 "未提交字段" 和 "提交空值"
type AssetUpdateReq struct {
	Name        *string `json:"name"`        // 资产名称
	Description *string `json:"description"` // 描述信息
	Tags        *string `json:"tags"`        // 标签（逗号分隔）
	FolderPath  *string `json:"folderPath"`  // 虚拟文件夹路径
	IsPublic    *bool   `json:"isPublic"`    // 是否公开
}

// AssetStatusUpdateReq 资产状态更新请求
type AssetStatusUpdateReq struct {
	Status asset.AssetStatus `json:"status" binding:"required,oneof=PENDING ACTIVE ARCHIVED DELETED"` // 状态
}

// AssetMoveReq 资产移动请求
type AssetMoveReq struct {
	FolderID   *uint  `json:"folderId"`   // 目标文件夹 ID（可选）
	FolderPath string `json:"folderPath"` // 目标文件夹路径
}

// AssetBatchDeleteReq 批量删除请求
type AssetBatchDeleteReq struct {
	IDs []uint `json:"ids" binding:"required,min=1,max=100"` // 资产 ID 列表（最多100条）
}

// AssetBatchStatusUpdateReq 批量状态更新请求
type AssetBatchStatusUpdateReq struct {
	IDs    []uint            `json:"ids" binding:"required,min=1,max=100"`                            // 资产 ID 列表（最多100条）
	Status asset.AssetStatus `json:"status" binding:"required,oneof=PENDING ACTIVE ARCHIVED DELETED"` // 状态
}

// AssetQueryReq 资产查询请求
type AssetQueryReq struct {
	Category   string `form:"category"`   // 分类筛选
	Status     string `form:"status"`     // 状态筛选
	FolderPath string `form:"folderPath"` // 文件夹路径筛选
	Tag        string `form:"tag"`        // 标签筛选
	IsPublic   *bool  `form:"isPublic"`   // 公开状态筛选
	Keyword    string `form:"keyword"`    // 关键词搜索（名称、文件名、描述）
}

// ======================================
// Response DTOs
// ======================================

// AssetResp 资产响应
type AssetResp struct {
	shared.BaseModel

	// 基本信息
	Name        string `json:"name"`        // 资产名称
	Filename    string `json:"filename"`    // 原始文件名
	Description string `json:"description"` // 描述信息
	Tags        string `json:"tags"`        // 标签
	FolderID    *uint  `json:"folderId"`    // 所属文件夹 ID
	FolderPath  string `json:"folderPath"`  // 虚拟文件夹路径

	// 存储信息
	ObjectKey   string            `json:"objectKey"`   // 存储对象键
	StorageType asset.StorageType `json:"storageType"` // 存储类型
	Bucket      string            `json:"bucket"`      // 存储桶名称

	// 文件属性
	Extension string `json:"extension"` // 文件扩展名
	MimeType  string `json:"mimeType"`  // MIME 类型
	Size      int64  `json:"size"`      // 文件大小（字节）
	Hash      string `json:"hash"`      // 文件 MD5 哈希

	// 分类与状态
	Category asset.AssetCategory `json:"category"` // 资产分类
	Status   asset.AssetStatus   `json:"status"`   // 资产状态
	IsPublic bool                `json:"isPublic"` // 是否公开

	// 媒体元数据
	Width    int `json:"width,omitempty"`    // 宽度
	Height   int `json:"height,omitempty"`   // 高度
	Duration int `json:"duration,omitempty"` // 时长

	// 缩略图
	ThumbnailKey string `json:"thumbnailKey,omitempty"` // 缩略图对象键

	// 访问统计
	DownloadCount int `json:"downloadCount"` // 下载次数
	ViewCount     int `json:"viewCount"`     // 查看次数

	// 扩展信息（不存数据库）
	SizeFormatted string `json:"sizeFormatted"`          // 格式化的文件大小
	URL           string `json:"url,omitempty"`          // 访问 URL（可选）
	ThumbnailURL  string `json:"thumbnailUrl,omitempty"` // 缩略图 URL（可选）
}

// FromEntity 从资产实体转换
func (r *AssetResp) FromEntity(entity *asset.Asset) error {
	if err := copier.Copy(r, entity); err != nil {
		return err
	}
	r.SizeFormatted = entity.GetSizeFormatted()
	// 生成访问 URL
	if entity.ObjectKey != "" {
		r.URL = "/api/v1/assets/download/" + entity.ObjectKey
	}
	return nil
}

// AssetBriefResp 资产简要响应（用于列表）
type AssetBriefResp struct {
	ID           uint                `json:"id"`
	Name         string              `json:"name"`
	Filename     string              `json:"filename"`
	ObjectKey    string              `json:"objectKey"`
	Extension    string              `json:"extension"`
	MimeType     string              `json:"mimeType"`
	Size         int64               `json:"size"`
	Category     asset.AssetCategory `json:"category"`
	Status       asset.AssetStatus   `json:"status"`
	IsPublic     bool                `json:"isPublic"`
	ThumbnailKey string              `json:"thumbnailKey,omitempty"`
	CreatedAt    time.Time           `json:"createdAt"`
}

// FromEntity 从资产实体转换
func (r *AssetBriefResp) FromEntity(entity *asset.Asset) error {
	return copier.Copy(r, entity)
}

// AssetStatsResp 资产统计响应
type AssetStatsResp struct {
	TotalCount         int64                         `json:"totalCount"`         // 总数量
	TotalSize          int64                         `json:"totalSize"`          // 总大小（字节）
	TotalSizeFormatted string                        `json:"totalSizeFormatted"` // 格式化的总大小
	CategoryStats      map[asset.AssetCategory]int64 `json:"categoryStats"`      // 分类统计
	StatusStats        map[asset.AssetStatus]int64   `json:"statusStats"`        // 状态统计
}

// AssetUploadResp 资产上传响应
type AssetUploadResp struct {
	AssetResp
	IsDuplicate bool   `json:"isDuplicate"`       // 是否为重复文件
	Message     string `json:"message,omitempty"` // 提示信息
}
