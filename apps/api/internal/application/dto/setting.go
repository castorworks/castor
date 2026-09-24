package dto

import (
	"time"

	"github.com/castorworks/castor/internal/domain/setting"
	"github.com/jinzhu/copier"
)

//
// Request DTOs
//

// SettingPostReq 创建配置请求
type SettingPostReq struct {
	Key         string `json:"key" binding:"required,min=1,max=100"`                                // 配置键（必填）
	Value       string `json:"value"`                                                               // 配置值
	Type        string `json:"type" binding:"omitempty,oneof=STRING NUMBER BOOL JSON ARRAY SECRET"` // 值类型
	Category    string `json:"category" binding:"omitempty"`                                        // 配置分类
	Name        string `json:"name" binding:"required,min=1,max=100"`                               // 配置名称
	Description string `json:"description" binding:"omitempty,max=500"`                             // 配置描述
	DefaultVal  string `json:"defaultVal"`                                                          // 默认值
	IsPublic    bool   `json:"isPublic"`                                                            // 是否公开
	SortOrder   int    `json:"sortOrder"`                                                           // 排序顺序
}

// SettingPutReq 更新配置请求
type SettingPutReq struct {
	Value       *string `json:"value"`                                   // 配置值（指针，允许设为空字符串）
	Name        *string `json:"name" binding:"omitempty,max=100"`        // 配置名称
	Description *string `json:"description" binding:"omitempty,max=500"` // 配置描述
	IsPublic    *bool   `json:"isPublic"`                                // 是否公开
	SortOrder   *int    `json:"sortOrder"`                               // 排序顺序
}

// SettingBatchUpdateReq 批量更新配置请求
type SettingBatchUpdateReq struct {
	Settings []SettingUpdateItem `json:"settings" binding:"required,min=1"` // 配置列表
}

// SettingUpdateItem 配置更新项
type SettingUpdateItem struct {
	Key   string `json:"key" binding:"required"` // 配置键
	Value string `json:"value"`                  // 配置值
}

//
// Response DTOs
//

// SettingResp 配置响应
type SettingResp struct {
	ID          uint                   `json:"id"`
	CreatedAt   time.Time              `json:"createdAt"`
	CreatedBy   uint                   `json:"createdBy"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	UpdatedBy   uint                   `json:"updatedBy"`
	Key         string                 `json:"key"`
	Value       string                 `json:"value"`
	Type        string                 `json:"type"`
	Category    string                 `json:"category"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	DefaultVal  string                 `json:"defaultVal"`
	Options     setting.SettingOptions `json:"options,omitempty"`
	IsSystem    bool                   `json:"isSystem"`
	IsPublic    bool                   `json:"isPublic"`
	SortOrder   int                    `json:"sortOrder"`
}

// FromEntity 从实体转换
func (r *SettingResp) FromEntity(entity *setting.Setting) error {
	return copier.Copy(r, entity)
}
