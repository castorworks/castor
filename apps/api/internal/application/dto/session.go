package dto

import (
	"time"

	"github.com/castorworks/castor/internal/domain/permission"
)

//
// Response DTOs
//

// SessionResp 在线会话响应（仅列出未撤销、未过期的授权会话）
type SessionResp struct {
	ID           string    `json:"id"`           // 会话 ID
	UserID       uint      `json:"userId"`       // 用户 ID
	Username     string    `json:"username"`     // 用户名
	IpAddr       string    `json:"ipAddr"`       // 登录 IP
	UserAgent    string    `json:"userAgent"`    // 登录时的用户代理
	Remember     bool      `json:"remember"`     // 登录时是否勾选"记住我"
	CreatedAt    time.Time `json:"createdAt"`    // 登录时间
	LastActiveAt time.Time `json:"lastActiveAt"` // 最近一次刷新令牌的时间
	ExpiresAt    time.Time `json:"expiresAt"`    // 会话最长有效期
	Current      bool      `json:"current"`      // 是否为调用者自己的当前会话
}

// FromEntity 从授权会话实体转换
func (r *SessionResp) FromEntity(entity *permission.AuthorizationSession, currentSessionID string) {
	*r = SessionResp{
		ID:           entity.ID,
		UserID:       entity.UserID,
		Username:     entity.Username,
		IpAddr:       entity.IpAddr,
		UserAgent:    entity.UserAgent,
		Remember:     entity.Remember,
		CreatedAt:    entity.CreatedAt,
		LastActiveAt: entity.LastActiveAt,
		ExpiresAt:    entity.ExpiresAt,
		Current:      entity.ID == currentSessionID,
	}
}
