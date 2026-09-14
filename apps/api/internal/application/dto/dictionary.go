package dto

import (
	"time"

	"github.com/castorworks/castor/internal/domain/dictionary"
	"github.com/jinzhu/copier"
)

//
// 字典类型 Request DTOs
//

// DictTypePostReq 创建字典类型请求
type DictTypePostReq struct {
	Code        string `json:"code" binding:"required,min=1,max=100"`   // 字典类型编码（必填）
	Name        string `json:"name" binding:"required,min=1,max=100"`   // 字典类型名称（必填）
	Description string `json:"description" binding:"omitempty,max=500"` // 描述
	SortOrder   int    `json:"sortOrder"`                               // 排序顺序
}

// DictTypePutReq 更新字典类型请求
type DictTypePutReq struct {
	Name        *string `json:"name" binding:"omitempty,max=100"`        // 字典类型名称
	Description *string `json:"description" binding:"omitempty,max=500"` // 描述
	IsEnabled   *bool   `json:"isEnabled"`                               // 是否启用
	SortOrder   *int    `json:"sortOrder"`                               // 排序顺序
}

//
// 字典类型 Response DTOs
//

// DictTypeResp 字典类型响应
type DictTypeResp struct {
	ID          uint      `json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	CreatedBy   uint      `json:"createdBy"`
	UpdatedAt   time.Time `json:"updatedAt"`
	UpdatedBy   uint      `json:"updatedBy"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"isSystem"`
	IsEnabled   bool      `json:"isEnabled"`
	SortOrder   int       `json:"sortOrder"`
}

// FromEntity 从实体转换
func (r *DictTypeResp) FromEntity(entity *dictionary.DictType) error {
	return copier.Copy(r, entity)
}

//
// 字典项 Request DTOs
//

// DictItemPostReq 创建字典项请求
type DictItemPostReq struct {
	TypeCode    string `json:"typeCode" binding:"required,min=1,max=100"` // 所属字典类型编码（必填）
	Label       string `json:"label" binding:"required,min=1,max=200"`    // 显示标签（必填）
	Value       string `json:"value" binding:"required,min=1,max=200"`    // 值（必填）
	Description string `json:"description" binding:"omitempty,max=500"`   // 描述
	Extra       string `json:"extra"`                                     // 扩展数据（JSON）
	Color       string `json:"color" binding:"omitempty,max=50"`          // 颜色
	Icon        string `json:"icon" binding:"omitempty,max=100"`          // 图标
	ParentID    *uint  `json:"parentId"`                                  // 父级ID
	IsDefault   bool   `json:"isDefault"`                                 // 是否默认选项
	SortOrder   int    `json:"sortOrder"`                                 // 排序顺序
}

// DictItemPutReq 更新字典项请求
type DictItemPutReq struct {
	Label       *string `json:"label" binding:"omitempty,max=200"`       // 显示标签
	Value       *string `json:"value" binding:"omitempty,max=200"`       // 值
	Description *string `json:"description" binding:"omitempty,max=500"` // 描述
	Extra       *string `json:"extra"`                                   // 扩展数据（JSON）
	Color       *string `json:"color" binding:"omitempty,max=50"`        // 颜色
	Icon        *string `json:"icon" binding:"omitempty,max=100"`        // 图标
	ParentID    *uint   `json:"parentId"`                                // 父级ID
	IsDefault   *bool   `json:"isDefault"`                               // 是否默认选项
	IsEnabled   *bool   `json:"isEnabled"`                               // 是否启用
	SortOrder   *int    `json:"sortOrder"`                               // 排序顺序
}

//
// 字典项 Response DTOs
//

// DictItemResp 字典项响应
type DictItemResp struct {
	ID          uint      `json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	CreatedBy   uint      `json:"createdBy"`
	UpdatedAt   time.Time `json:"updatedAt"`
	UpdatedBy   uint      `json:"updatedBy"`
	TypeCode    string    `json:"typeCode"`
	Label       string    `json:"label"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	Extra       string    `json:"extra"`
	Color       string    `json:"color"`
	Icon        string    `json:"icon"`
	ParentID    *uint     `json:"parentId"`
	IsDefault   bool      `json:"isDefault"`
	IsEnabled   bool      `json:"isEnabled"`
	SortOrder   int       `json:"sortOrder"`
}

// FromEntity 从实体转换
func (r *DictItemResp) FromEntity(entity *dictionary.DictItem) error {
	return copier.Copy(r, entity)
}
