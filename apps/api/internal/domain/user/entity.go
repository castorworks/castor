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

	Username             string    `json:"username"`             // 用户名
	AccountSource        string    `json:"accountSource"`        // 账号来源
	Email                string    `json:"email"`                // 邮箱（可为空；非空时全局唯一）
	Mobile               string    `json:"mobile"`               // 手机号（可为空；非空时全局唯一）
	EmailVerified        bool      `json:"emailVerified"`        // 邮箱是否已验证
	MobileVerified       bool      `json:"mobileVerified"`       // 手机号是否已验证
	Name                 string    `json:"name"`                 // 显示名称
	Avatar               string    `json:"avatar"`               // 头像
	Password             string    `json:"-"`                    // 密码（禁止序列化）
	Enable               bool      `json:"enable"`               // 启用状态
	Locked               bool      `json:"locked"`               // 账号锁定状态
	AccountExpireDate    time.Time `json:"accountExpireDate"`    // 账号过期时间
	CredentialExpireDate time.Time `json:"credentialExpireDate"` // 凭证过期时间
	DepartmentID         *uint     `json:"departmentId"`         // 所属部门（可为空）
	// MuteNotificationEmails 用户不想收通知邮件（缺省收）；只影响邮件通道，站内通知照常送达
	MuteNotificationEmails bool   `json:"muteNotificationEmails"`
	AuthorizationSessionID string `json:"-"`
}
