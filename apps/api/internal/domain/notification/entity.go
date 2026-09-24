package notification

import "time"

// NotificationType 通知类型
type NotificationType string

const (
	TypeSystem   NotificationType = "SYSTEM"
	TypeAnnounce NotificationType = "ANNOUNCE"
	TypeMessage  NotificationType = "MESSAGE"
	TypeAlert    NotificationType = "ALERT"
	TypeTask     NotificationType = "TASK"
)

// NotificationLevel 通知级别
type NotificationLevel string

const (
	LevelInfo    NotificationLevel = "INFO"
	LevelSuccess NotificationLevel = "SUCCESS"
	LevelWarning NotificationLevel = "WARNING"
	LevelError   NotificationLevel = "ERROR"
)

// Notification 通知实体（纯 Domain 模型，无 ORM 依赖）
type Notification struct {
	ID           uint              `json:"id"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	CreatedBy    uint              `json:"createdBy"`
	UpdatedBy    uint              `json:"updatedBy"`
	Title        string            `json:"title"`
	Content      string            `json:"content"`
	TemplateKey  string            `json:"templateKey"`
	TemplateData string            `json:"templateData"`
	Type         NotificationType  `json:"type"`
	Level        NotificationLevel `json:"level"`
	Link         string            `json:"link"`
	Extra        string            `json:"extra"`
	SenderID     uint              `json:"senderId"`
	IsGlobal     bool              `json:"isGlobal"`
	ExpireAt     *time.Time        `json:"expireAt"`
	// SendEmail 同时发邮件给有已验证邮箱、且没关闭通知邮件的收件人
	SendEmail bool `json:"sendEmail"`
}

// UserNotification 用户通知关联（纯 Domain 模型，无 ORM 依赖）
type UserNotification struct {
	ID             uint       `json:"id"`
	CreatedAt      time.Time  `json:"createdAt"`
	UserID         uint       `json:"userId"`
	NotificationID uint       `json:"notificationId"`
	IsRead         bool       `json:"isRead"`
	ReadAt         *time.Time `json:"readAt"`
	IsDismissed    bool       `json:"isDismissed"`
	DismissedAt    *time.Time `json:"dismissedAt"`
}

// NotificationWithReadStatus 带阅读状态的通知
type NotificationWithReadStatus struct {
	Notification
	IsRead      bool       `json:"isRead"`
	ReadAt      *time.Time `json:"readAt"`
	IsDismissed bool       `json:"isDismissed"`
	DismissedAt *time.Time `json:"dismissedAt"`
}

// NotificationDeliveryStats 通知投递统计
type NotificationDeliveryStats struct {
	RecipientCount int64 `json:"recipientCount"`
	ReadCount      int64 `json:"readCount"`
	UnreadCount    int64 `json:"unreadCount"`
	DismissedCount int64 `json:"dismissedCount"`
}
