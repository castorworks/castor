package dto

import (
	"time"

	"github.com/castorworks/castor/internal/domain/dictionary"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/jinzhu/copier"
)

//
// 字典类型 Request DTOs
//

// DictTypePostReq 创建字典类型请求
type DictTypePostReq struct {
	Code        string      `json:"code" binding:"required,min=1,max=100"`   // 字典类型编码（必填）
	Name        I18nTextReq `json:"name" binding:"required"`                 // 字典类型名称，四语言必填
	Description string      `json:"description" binding:"omitempty,max=500"` // 描述
	IsPublic    bool        `json:"isPublic"`                                // 是否出现在免认证的公开字典接口里
	SortOrder   int         `json:"sortOrder"`                               // 排序顺序
}

// DictTypePutReq 更新字典类型请求
type DictTypePutReq struct {
	Name        *I18nTextReq `json:"name"`                                    // 字典类型名称，四语言必填
	Description *string      `json:"description" binding:"omitempty,max=500"` // 描述
	IsPublic    *bool        `json:"isPublic"`                                // 是否公开
	IsEnabled   *bool        `json:"isEnabled"`                               // 是否启用
	SortOrder   *int         `json:"sortOrder"`                               // 排序顺序
}

//
// 字典类型 Response DTOs
//

// DictTypeResp 字典类型响应
type DictTypeResp struct {
	ID          uint            `json:"id"`
	CreatedAt   time.Time       `json:"createdAt"`
	CreatedBy   uint            `json:"createdBy"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	UpdatedBy   uint            `json:"updatedBy"`
	Code        string          `json:"code"`
	Name        shared.I18nText `json:"name"`
	Description string          `json:"description"`
	IsSystem    bool            `json:"isSystem"`
	IsPublic    bool            `json:"isPublic"`
	IsEnabled   bool            `json:"isEnabled"`
	SortOrder   int             `json:"sortOrder"`
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
	TypeCode    string      `json:"typeCode" binding:"required,min=1,max=100"` // 所属字典类型编码（必填）
	Label       I18nTextReq `json:"label" binding:"required"`                  // 显示标签，四语言必填
	Value       string      `json:"value" binding:"required,min=1,max=200"`    // 值（必填）
	Description string      `json:"description" binding:"omitempty,max=500"`   // 描述
	Color       string      `json:"color" binding:"omitempty,max=50"`          // 颜色：dictionary.Colors 中的 token 名，由服务层校验
	Icon        string      `json:"icon" binding:"omitempty,alphanum,max=100"` // 图标：前端 Icons 的 key
	IsDefault   bool        `json:"isDefault"`                                 // 是否默认选项
	SortOrder   int         `json:"sortOrder"`                                 // 排序顺序
}

// DictItemPutReq 更新字典项请求
type DictItemPutReq struct {
	Label       *I18nTextReq `json:"label"`                                     // 显示标签，四语言必填
	Value       *string      `json:"value" binding:"omitempty,max=200"`         // 值；系统项不可修改
	Description *string      `json:"description" binding:"omitempty,max=500"`   // 描述
	Color       *string      `json:"color" binding:"omitempty,max=50"`          // 颜色：空串表示清除
	Icon        *string      `json:"icon" binding:"omitempty,alphanum,max=100"` // 图标：空串表示清除
	IsDefault   *bool        `json:"isDefault"`                                 // 是否默认选项
	IsEnabled   *bool        `json:"isEnabled"`                                 // 是否启用
	SortOrder   *int         `json:"sortOrder"`                                 // 排序顺序
}

//
// 字典项 Response DTOs
//

// DictItemResp 字典项响应
type DictItemResp struct {
	ID          uint            `json:"id"`
	CreatedAt   time.Time       `json:"createdAt"`
	CreatedBy   uint            `json:"createdBy"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	UpdatedBy   uint            `json:"updatedBy"`
	TypeCode    string          `json:"typeCode"`
	Label       shared.I18nText `json:"label"`
	Value       string          `json:"value"`
	Description string          `json:"description"`
	Color       string          `json:"color"`
	Icon        string          `json:"icon"`
	IsSystem    bool            `json:"isSystem"`
	IsDefault   bool            `json:"isDefault"`
	IsEnabled   bool            `json:"isEnabled"`
	SortOrder   int             `json:"sortOrder"`
}

// FromEntity 从实体转换
func (r *DictItemResp) FromEntity(entity *dictionary.DictItem) error {
	return copier.Copy(r, entity)
}

// I18nTextReq 是请求体里的四语言文案。
//
// 菜单标题已经要求四种语言都填，字典标签同理：少填一种语言，
// 那种语言的用户在徽章和下拉框里看到的就是另一种语言的字面量。
type I18nTextReq struct {
	En string `json:"en" binding:"required,min=1,max=200"`
	Zh string `json:"zh" binding:"required,min=1,max=200"`
	Ja string `json:"ja" binding:"required,min=1,max=200"`
	Ko string `json:"ko" binding:"required,min=1,max=200"`
}

// ToI18nText 转换为领域文案。
func (r I18nTextReq) ToI18nText() shared.I18nText {
	return shared.Text(r.En, r.Zh, r.Ja, r.Ko)
}
