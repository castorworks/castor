package shared

import "time"

// BaseModel 通用的基础模型，仅包含审计字段（无 ORM 依赖）。
// GORM 标签和生命周期钩子已移至 infrastructure/persistence/models/ 层。
type BaseModel struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	CreatedBy uint      `json:"createdBy"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy uint      `json:"updatedBy"`
}
