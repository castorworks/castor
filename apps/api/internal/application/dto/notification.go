package dto

import (
	"time"

	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/jinzhu/copier"
)

//
// Request DTOs
//

// NotificationPostReq 创建通知请求
type NotificationPostReq struct {
	Title        string     `json:"title" binding:"required,min=1,max=200"`                            // 通知标题
	Content      string     `json:"content"`                                                           // 通知内容
	TemplateKey  string     `json:"templateKey"`                                                       // 模板 Key
	TemplateData string     `json:"templateData"`                                                      // 模板数据 JSON
	Type         string     `json:"type" binding:"omitempty,oneof=SYSTEM ANNOUNCE MESSAGE ALERT TASK"` // 通知类型
	Level        string     `json:"level" binding:"omitempty,oneof=INFO SUCCESS WARNING ERROR"`        // 通知级别
	Link         string     `json:"link" binding:"omitempty,max=500"`                                  // 跳转链接
	Extra        string     `json:"extra"`                                                             // 扩展数据
	IsGlobal     bool       `json:"isGlobal"`                                                          // 是否全局通知
	UserIDs      []uint     `json:"userIds"`                                                           // 指定接收用户（非全局时）
	ExpireAt     *time.Time `json:"expireAt"`                                                          // 过期时间
}

// NotificationPutReq 更新通知请求
type NotificationPutReq struct {
	Title        string     `json:"title" binding:"omitempty,max=200"`                                 // 通知标题
	Content      string     `json:"content"`                                                           // 通知内容
	TemplateKey  string     `json:"templateKey"`                                                       // 模板 Key
	TemplateData string     `json:"templateData"`                                                      // 模板数据 JSON
	Type         string     `json:"type" binding:"omitempty,oneof=SYSTEM ANNOUNCE MESSAGE ALERT TASK"` // 通知类型
	Level        string     `json:"level" binding:"omitempty,oneof=INFO SUCCESS WARNING ERROR"`        // 通知级别
	Link         string     `json:"link"`                                                              // 跳转链接
	Extra        string     `json:"extra"`                                                             // 扩展数据
	IsGlobal     *bool      `json:"isGlobal"`                                                          // 是否全局通知
	UserIDs      []uint     `json:"userIds"`                                                           // 指定接收用户
	ExpireAt     *time.Time `json:"expireAt"`                                                          // 过期时间
}

// NotificationBatchDeleteReq 批量删除请求
type NotificationBatchDeleteReq struct {
	Ids []uint `json:"ids" binding:"required,min=1,max=100"` // ID 列表（最多100条）
}

// MarkNotificationsReadReq 标记通知已读请求
type MarkNotificationsReadReq struct {
	Ids []uint `json:"ids" binding:"required,min=1"` // 通知 ID 列表
}

//
// Response DTOs
//

// NotificationResp 通知响应
type NotificationResp struct {
	ID             uint       `json:"id"`
	CreatedAt      time.Time  `json:"createdAt"`
	CreatedBy      uint       `json:"createdBy"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	UpdatedBy      uint       `json:"updatedBy"`
	Title          string     `json:"title"`
	Content        string     `json:"content"`
	TemplateKey    string     `json:"templateKey"`
	TemplateData   string     `json:"templateData"`
	Type           string     `json:"type"`
	Level          string     `json:"level"`
	Link           string     `json:"link"`
	Extra          string     `json:"extra"`
	SenderID       uint       `json:"senderId"`
	IsGlobal       bool       `json:"isGlobal"`
	ExpireAt       *time.Time `json:"expireAt"`
	IsRead         bool       `json:"isRead"`
	ReadAt         *time.Time `json:"readAt,omitempty"`
	IsDismissed    bool       `json:"isDismissed"`
	DismissedAt    *time.Time `json:"dismissedAt,omitempty"`
	RecipientCount int64      `json:"recipientCount"`
	ReadCount      int64      `json:"readCount"`
	UnreadCount    int64      `json:"unreadCount"`
	DismissedCount int64      `json:"dismissedCount"`
}

// FromEntity 从实体转换
func (r *NotificationResp) FromEntity(entity *notification.Notification) error {
	return copier.Copy(r, entity)
}

// NotificationUnreadCountResp 未读数量响应
type NotificationUnreadCountResp struct {
	Count int64 `json:"count"`
}

// NotificationRecipientResp 通知接收人响应
type NotificationRecipientResp struct {
	UserID      uint       `json:"userId"`
	IsRead      bool       `json:"isRead"`
	ReadAt      *time.Time `json:"readAt,omitempty"`
	IsDismissed bool       `json:"isDismissed"`
	DismissedAt *time.Time `json:"dismissedAt,omitempty"`
	DeliveredAt time.Time  `json:"deliveredAt"`
}
