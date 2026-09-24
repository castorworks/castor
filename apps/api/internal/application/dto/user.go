package dto

import (
	"time"

	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/jinzhu/copier"
)

//
// Request DTOs
//

// UserPostReq 创建用户请求
type UserPostReq struct {
	Username string `json:"username" binding:"required,min=3,max=100"` // 用户名（必填，3-100字符）
	Name     string `json:"name" binding:"omitempty,max=100"`          // 显示名称（可选，最大100字符）
	Password string `json:"password" binding:"required,max=512"`       // 密码（必填，RSA 密文；明文长度在解密后校验）
	Email    string `json:"email" binding:"omitempty,max=255"`         // 邮箱（可选；由管理员代为担保，写入即标记为已验证）
	Mobile   string `json:"mobile" binding:"omitempty,max=32"`         // 手机号（可选；由管理员代为担保，写入即标记为已验证）
	// DepartmentID 所属部门（可选；数据范围不是"全部"的管理员必须指定自己范围内的部门）
	DepartmentID *uint `json:"departmentId"`
}

// UserPutReq 更新用户请求
type UserPutReq struct {
	Name                 string     `json:"name" binding:"omitempty,max=100"`   // 显示名称（可选，最大100字符）
	Avatar               string     `json:"avatar" binding:"omitempty,max=500"` // 头像（可选，公开图片资产的对象键）
	Password             string     `json:"password" binding:"omitempty"`       // 密码（可选，RSA加密密文）
	Email                *string    `json:"email" binding:"omitempty,max=255"`  // 邮箱（nil=不更新，空串=解绑；写入即标记为已验证）
	Mobile               *string    `json:"mobile" binding:"omitempty,max=32"`  // 手机号（nil=不更新，空串=解绑；写入即标记为已验证）
	Enable               *bool      `json:"enable"`                             // 启用状态（nil=不更新）
	Locked               *bool      `json:"locked"`                             // 锁定状态（nil=不更新）
	AccountExpireDate    *time.Time `json:"accountExpireDate"`                  // 账号过期时间（nil=不更新，zero=清除）
	CredentialExpireDate *time.Time `json:"credentialExpireDate"`               // 凭证过期时间（nil=不更新，zero=清除）
	DepartmentID         *uint      `json:"departmentId"`                       // 所属部门（nil=不更新，0=移出部门）
}

// HasUpdatableFields checks if at least one field is provided
func (r *UserPutReq) HasUpdatableFields() bool {
	return r.Name != "" || r.Avatar != "" || r.Password != "" ||
		r.Email != nil || r.Mobile != nil ||
		r.Enable != nil || r.Locked != nil ||
		r.AccountExpireDate != nil || r.CredentialExpireDate != nil || r.DepartmentID != nil
}

// UserNamePutReq 更新用户名请求
// NotificationPreferencesPutReq 通知偏好
type NotificationPreferencesPutReq struct {
	MuteNotificationEmails bool `json:"muteNotificationEmails"`
}

type UserNamePutReq struct {
	Name string `json:"name" binding:"required,min=1,max=100"` // 显示名称（必填，1-100字符）
}

// UserPasswordPutReq 修改密码请求
type UserPasswordPutReq struct {
	CurrentPassword string `json:"currentPassword" binding:"required,min=1"` // 当前密码（必填）
	NewPassword     string `json:"newPassword" binding:"required,max=512"`   // 新密码（必填，RSA 密文；明文长度在解密后校验）
}

// UserPasswordResetPutReq 重置密码请求
type UserPasswordResetPutReq struct {
	Username    string `json:"username" binding:"required,min=3,max=100"`   // 账号标识（必填）：已验证的邮箱/手机号，或验证码登录账号的用户名
	NewPassword string `json:"newPassword" binding:"required,max=512"`      // 新密码（必填，RSA 密文；明文长度在解密后校验）
	ConfirmCode string `json:"confirmCode" binding:"required,min=4,max=10"` // 验证码（必填，4-10字符）
}

// ExpiredPasswordPutReq 凭证过期后修改密码请求（未登录调用，按用户名 + 当前密码认证）
type ExpiredPasswordPutReq struct {
	Username        string `json:"username" binding:"required,max=100"`        // 用户名（必填）
	CurrentPassword string `json:"currentPassword" binding:"required,max=512"` // 当前密码（必填，RSA 密文）
	NewPassword     string `json:"newPassword" binding:"required,max=512"`     // 新密码（必填，RSA 密文；明文在解密后按密码策略校验）
	CaptchaId       string `json:"captchaId" binding:"max=64"`                 // 图形验证码 ID（与密码登录相同的开关）
	CaptchaCode     string `json:"captchaCode" binding:"max=16"`               // 图形验证码
}

// ContactCodePostReq 发送联系方式绑定验证码请求
type ContactCodePostReq struct {
	ContactType string `json:"contactType" binding:"required,max=16"`    // 联系方式类型（必填，EMAIL/MOBILE，大小写不敏感）
	Contact     string `json:"contact" binding:"required,min=3,max=255"` // 邮箱或手机号（必填）
}

// ContactPutReq 绑定联系方式请求
type ContactPutReq struct {
	ContactType string `json:"contactType" binding:"required,max=16"`    // 联系方式类型（必填，EMAIL/MOBILE，大小写不敏感）
	Contact     string `json:"contact" binding:"required,min=3,max=255"` // 邮箱或手机号（必填）
	Code        string `json:"code" binding:"required,min=4,max=10"`     // 绑定验证码（必填，4-10字符）
}

//
// Response DTOs
//

// UserResp 用户响应
type UserResp struct {
	shared.BaseModel

	Username             string     `json:"username"`             // 用户名
	Name                 string     `json:"name"`                 // 显示名称
	Avatar               string     `json:"avatar"`               // 头像
	AccountSource        string     `json:"accountSource"`        // 账号来源 INTERNAL | FACEBOOK | QQ etc.
	Email                string     `json:"email"`                // 邮箱（未绑定为空串）
	Mobile               string     `json:"mobile"`               // 手机号（未绑定为空串）
	EmailVerified        bool       `json:"emailVerified"`        // 邮箱是否已验证
	MobileVerified       bool       `json:"mobileVerified"`       // 手机号是否已验证
	Enable               bool       `json:"enable"`               // 启用状态
	Locked               bool       `json:"locked"`               // 锁定状态
	AccountExpireDate    *time.Time `json:"accountExpireDate"`    // 账号过期时间（nil表示永不过期）
	CredentialExpireDate *time.Time `json:"credentialExpireDate"` // 凭证过期时间（nil表示永不过期）
	DepartmentID         *uint      `json:"departmentId"`         // 所属部门（nil 表示未归属部门）
	// MuteNotificationEmails 用户关闭了通知邮件
	MuteNotificationEmails bool `json:"muteNotificationEmails"`
	// TOTPEnabled 是否开启两步验证；只有管理端的用户列表与详情会填写
	TOTPEnabled *bool `json:"totpEnabled,omitempty"`
}

// FromEntity 从用户实体转换
func (r *UserResp) FromEntity(entity *user.User) error {
	if err := copier.Copy(r, entity); err != nil {
		return err
	}

	// Map zero time to nil pointer (zero = never expires → nil in response)
	if !entity.AccountExpireDate.IsZero() {
		t := entity.AccountExpireDate
		r.AccountExpireDate = &t
	} else {
		r.AccountExpireDate = nil
	}

	if !entity.CredentialExpireDate.IsZero() {
		t := entity.CredentialExpireDate
		r.CredentialExpireDate = &t
	} else {
		r.CredentialExpireDate = nil
	}

	return nil
}
