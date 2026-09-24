package dto

import (
	"time"

	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/jinzhu/copier"
)

//
// Request DTOs
//

// NotificationAttachmentsMax 单条通知最多携带的附件数
const NotificationAttachmentsMax = 10

// NotificationAttachmentReq 通知要携带的一个附件：先经上传接口或资产库拿到 objectKey
type NotificationAttachmentReq struct {
	ObjectKey string `json:"objectKey" binding:"required,max=500"` // 资产对象键
	Name      string `json:"name" binding:"omitempty,max=255"`     // 收件人看到的文件名；为空时用资产自己的文件名
}

// NotificationPostReq 创建通知请求
type NotificationPostReq struct {
	Title        string                      `json:"title" binding:"required,min=1,max=200"`                            // 通知标题
	Content      string                      `json:"content"`                                                           // 通知内容
	TemplateKey  string                      `json:"templateKey"`                                                       // 模板 Key
	TemplateData string                      `json:"templateData"`                                                      // 模板数据 JSON
	Type         string                      `json:"type" binding:"omitempty,oneof=SYSTEM ANNOUNCE MESSAGE ALERT TASK"` // 通知类型
	Level        string                      `json:"level" binding:"omitempty,oneof=INFO SUCCESS WARNING ERROR"`        // 通知级别
	Link         string                      `json:"link" binding:"omitempty,max=500"`                                  // 跳转链接
	Extra        string                      `json:"extra"`                                                             // 扩展数据
	IsGlobal     bool                        `json:"isGlobal"`                                                          // 是否全局通知
	UserIDs      []uint                      `json:"userIds"`                                                           // 指定接收用户（非全局时）
	ExpireAt     *time.Time                  `json:"expireAt"`                                                          // 过期时间
	Attachments  []NotificationAttachmentReq `json:"attachments" binding:"omitempty,max=10,dive"`                       // 附件（私有，仅收件人可下载）
	// SendEmail 同时发邮件（需要已配置邮件服务）
	SendEmail bool `json:"sendEmail"`
}

// NotificationPutReq 更新通知请求
type NotificationPutReq struct {
	Title        string                       `json:"title" binding:"omitempty,max=200"`                                 // 通知标题
	Content      string                       `json:"content"`                                                           // 通知内容
	TemplateKey  string                       `json:"templateKey"`                                                       // 模板 Key
	TemplateData string                       `json:"templateData"`                                                      // 模板数据 JSON
	Type         string                       `json:"type" binding:"omitempty,oneof=SYSTEM ANNOUNCE MESSAGE ALERT TASK"` // 通知类型
	Level        string                       `json:"level" binding:"omitempty,oneof=INFO SUCCESS WARNING ERROR"`        // 通知级别
	Link         string                       `json:"link"`                                                              // 跳转链接
	Extra        string                       `json:"extra"`                                                             // 扩展数据
	IsGlobal     *bool                        `json:"isGlobal"`                                                          // 是否全局通知
	UserIDs      []uint                       `json:"userIds"`                                                           // 指定接收用户
	ExpireAt     *time.Time                   `json:"expireAt"`                                                          // 过期时间
	Attachments  *[]NotificationAttachmentReq `json:"attachments" binding:"omitempty,max=10,dive"`                       // 附件（nil=不更新，空数组=清空）
	// SendEmail nil=不变；打开或改了收件人时，给还没发过的收件人补发
	SendEmail *bool `json:"sendEmail"`
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
	ID             uint                  `json:"id"`
	CreatedAt      time.Time             `json:"createdAt"`
	CreatedBy      uint                  `json:"createdBy"`
	UpdatedAt      time.Time             `json:"updatedAt"`
	UpdatedBy      uint                  `json:"updatedBy"`
	Title          string                `json:"title"`
	Content        string                `json:"content"`
	TemplateKey    string                `json:"templateKey"`
	TemplateData   string                `json:"templateData"`
	Type           string                `json:"type"`
	Level          string                `json:"level"`
	Link           string                `json:"link"`
	Extra          string                `json:"extra"`
	SenderID       uint                  `json:"senderId"`
	IsGlobal       bool                  `json:"isGlobal"`
	ExpireAt       *time.Time            `json:"expireAt"`
	IsRead         bool                  `json:"isRead"`
	ReadAt         *time.Time            `json:"readAt,omitempty"`
	IsDismissed    bool                  `json:"isDismissed"`
	DismissedAt    *time.Time            `json:"dismissedAt,omitempty"`
	RecipientCount int64                 `json:"recipientCount"`
	ReadCount      int64                 `json:"readCount"`
	UnreadCount    int64                 `json:"unreadCount"`
	DismissedCount int64                 `json:"dismissedCount"`
	Attachments    []AssetAttachmentResp `json:"attachments"`
	SendEmail      bool                  `json:"sendEmail"`
	// EmailStats 邮件投递统计（只在管理端、且通知要求发邮件时填写）
	EmailStats *NotificationEmailStats `json:"emailStats,omitempty"`
}

// NotificationEmailStats 通知邮件各状态的封数
type NotificationEmailStats struct {
	Pending int64 `json:"pending"`
	Sent    int64 `json:"sent"`
	Failed  int64 `json:"failed"`
	Skipped int64 `json:"skipped"`
}

// NotificationChannelsResp 可用的投递通道
type NotificationChannelsResp struct {
	// Email 是否配置了邮件服务
	Email bool `json:"email"`
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
	// EmailStatus 给该收件人的邮件状态（没有邮件时为空）
	EmailStatus string `json:"emailStatus,omitempty"`
}
