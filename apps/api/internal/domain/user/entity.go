package user

import "time"

// User 用户实体（纯 Domain 模型，无 ORM 依赖）
// GORM 标签和生命周期钩子已移至 infrastructure/persistence/models/。
type User struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	CreatedBy uint      `json:"createdBy"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy uint      `json:"updatedBy"`

	Username               string    `json:"username"`             // 用户名
	AccountSource          string    `json:"accountSource"`        // 账号来源
	Name                   string    `json:"name"`                 // 显示名称
	Avatar                 string    `json:"avatar"`               // 头像
	Password               string    `json:"-"`                    // 密码（禁止序列化）
	Enable                 bool      `json:"enable"`               // 启用状态
	Locked                 bool      `json:"locked"`               // 账号锁定状态
	AccountExpireDate      time.Time `json:"accountExpireDate"`    // 账号过期时间
	CredentialExpireDate   time.Time `json:"credentialExpireDate"` // 凭证过期时间
	AuthorizationSessionID string    `json:"-"`
}
