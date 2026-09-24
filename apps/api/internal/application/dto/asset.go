package dto

import (
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
	// 归属范围：缺省 LIBRARY；后台业务表单里的文件字段传 ATTACHMENT，随引用回收
	Scope asset.AssetScope `form:"scope" binding:"omitempty,oneof=LIBRARY ATTACHMENT"`
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
	Status asset.AssetStatus `json:"status" binding:"required,oneof=ACTIVE ARCHIVED"` // 状态
}

// AssetMoveReq 资产移动请求
type AssetMoveReq struct {
	FolderPath string `json:"folderPath" binding:"max=500"` // 目标虚拟文件夹路径
}

// AssetBatchDeleteReq 批量删除请求
type AssetBatchDeleteReq struct {
	IDs []uint `json:"ids" binding:"required,min=1,max=100"` // 资产 ID 列表（最多100条）
}

// AssetBatchStatusUpdateReq 批量状态更新请求
type AssetBatchStatusUpdateReq struct {
	IDs    []uint            `json:"ids" binding:"required,min=1,max=100"`            // 资产 ID 列表（最多100条）
	Status asset.AssetStatus `json:"status" binding:"required,oneof=ACTIVE ARCHIVED"` // 状态
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
	FolderPath  string `json:"folderPath"`  // 虚拟文件夹路径

	// 存储信息
	ObjectKey string `json:"objectKey"` // 存储对象键（业务记录用它引用资产）
	Bucket    string `json:"bucket"`    // 存储桶名称

	// 文件属性
	Extension string `json:"extension"` // 文件扩展名
	MimeType  string `json:"mimeType"`  // MIME 类型
	Size      int64  `json:"size"`      // 文件大小（字节）
	Hash      string `json:"hash"`      // 文件内容 SHA-256 哈希

	// 分类与状态
	Category asset.AssetCategory `json:"category"` // 资产分类
	Status   asset.AssetStatus   `json:"status"`   // 资产状态
	Scope    asset.AssetScope    `json:"scope"`    // 归属范围
	IsPublic bool                `json:"isPublic"` // 是否公开

	// 访问统计
	DownloadCount int `json:"downloadCount"` // 下载次数

	// 扩展信息（不存数据库）
	SizeFormatted string `json:"sizeFormatted"` // 格式化的文件大小
	URL           string `json:"url,omitempty"` // 免认证下载地址（仅公开资产可访问）
}

// AssetDetailResp 资产详情响应：在 AssetResp 之上附带引用方，解释资产为何不可删除
type AssetDetailResp struct {
	AssetResp
	References []asset.Reference `json:"references"`
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

// AssetAttachmentResp 业务记录上的一个附件：只含展示与下载所需的字段，
// 不带存储桶、哈希、上传人等资产管理信息——它会出现在面向终端用户的响应里。
type AssetAttachmentResp struct {
	ObjectKey     string              `json:"objectKey"`
	Name          string              `json:"name"`
	Extension     string              `json:"extension"`
	MimeType      string              `json:"mimeType"`
	Size          int64               `json:"size"`
	SizeFormatted string              `json:"sizeFormatted"`
	Category      asset.AssetCategory `json:"category"`
}

// FromEntity 从资产实体转换；name 是引用方登记的显示名，为空时退回资产自己的文件名
func (r *AssetAttachmentResp) FromEntity(entity *asset.Asset, name string) {
	r.ObjectKey = entity.ObjectKey
	r.Name = name
	if r.Name == "" {
		r.Name = entity.Filename
	}
	r.Extension = entity.Extension
	r.MimeType = entity.MimeType
	r.Size = entity.Size
	r.SizeFormatted = entity.GetSizeFormatted()
	r.Category = entity.Category
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
	IsDuplicate bool `json:"isDuplicate"` // 是否命中去重（返回的是已有资产）
}
