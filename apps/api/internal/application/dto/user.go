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
	Password string `json:"password" binding:"required,min=6,max=128"` // 密码（必填，6-128字符）
}

// UserPutReq 更新用户请求
type UserPutReq struct {
	Name                 string     `json:"name" binding:"omitempty,max=100"`   // 显示名称（可选，最大100字符）
	Avatar               string     `json:"avatar" binding:"omitempty,max=500"` // 头像（可选，最大500字符）
	Password             string     `json:"password" binding:"omitempty"`       // 密码（可选，RSA加密密文）
	Enable               *bool      `json:"enable"`                             // 启用状态（nil=不更新）
	Locked               *bool      `json:"locked"`                             // 锁定状态（nil=不更新）
	AccountExpireDate    *time.Time `json:"accountExpireDate"`                  // 账号过期时间（nil=不更新，zero=清除）
	CredentialExpireDate *time.Time `json:"credentialExpireDate"`               // 凭证过期时间（nil=不更新，zero=清除）
}

// HasUpdatableFields checks if at least one field is provided
func (r *UserPutReq) HasUpdatableFields() bool {
	return r.Name != "" || r.Avatar != "" || r.Password != "" ||
		r.Enable != nil || r.Locked != nil ||
		r.AccountExpireDate != nil || r.CredentialExpireDate != nil
}

// UserNamePutReq 更新用户名请求
type UserNamePutReq struct {
	Name string `json:"name" binding:"required,min=1,max=100"` // 显示名称（必填，1-100字符）
}

// UserPasswordPutReq 修改密码请求
type UserPasswordPutReq struct {
	CurrentPassword string `json:"currentPassword" binding:"required,min=1"`     // 当前密码（必填）
	NewPassword     string `json:"newPassword" binding:"required,min=6,max=128"` // 新密码（必填，6-128字符）
}

// UserPasswordResetPutReq 重置密码请求
type UserPasswordResetPutReq struct {
	Username    string `json:"username" binding:"required,min=3,max=100"`    // 用户名（必填，3-100字符）
	NewPassword string `json:"newPassword" binding:"required,min=6,max=128"` // 新密码（必填，6-128字符）
	ConfirmCode string `json:"confirmCode" binding:"required,min=4,max=10"`  // 验证码（必填，4-10字符）
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
	Enable               bool       `json:"enable"`               // 启用状态
	Locked               bool       `json:"locked"`               // 锁定状态
	AccountExpireDate    *time.Time `json:"accountExpireDate"`    // 账号过期时间（nil表示永不过期）
	CredentialExpireDate *time.Time `json:"credentialExpireDate"` // 凭证过期时间（nil表示永不过期）
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
