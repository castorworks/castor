package notification

import (
	"context"
	"time"
)

// EmailStatus 一封通知邮件的状态；取值与字典 notification_email_status 对应。
type EmailStatus string

const (
	EmailPending EmailStatus = "PENDING"
	EmailSent    EmailStatus = "SENT"
	EmailFailed  EmailStatus = "FAILED"
	// EmailSkipped 到发送时已不必再发（通知已过期、收件人关闭了通知邮件或邮箱不再可用）
	EmailSkipped EmailStatus = "SKIPPED"
)

// EmailDelivery 发件箱里的一封通知邮件：一条通知对一个收件人。
type EmailDelivery struct {
	ID             uint
	NotificationID uint
	UserID         uint
	Email          string
	Status         EmailStatus
	Attempts       int
	LastError      string
	NextAttemptAt  time.Time
	SentAt         *time.Time
	CreatedAt      time.Time
}

// PendingEmail 待发送的邮件及发送时需要的通知与收件人信息
type PendingEmail struct {
	EmailDelivery
	Notification Notification
	// RecipientName 收件人显示名（为空时用用户名）
	RecipientName string
	// StillWanted 收件人仍然启用、邮箱仍已验证且未关闭通知邮件
	StillWanted bool
}

// EmailRepository 通知邮件发件箱
type EmailRepository interface {
	// Enqueue 为通知当前的收件人登记邮件：全局通知为全部用户，否则为定向投递的用户。
	// 只登记账号启用、邮箱已验证、没有关闭通知邮件的人；已登记过的不重复登记。返回新登记的封数。
	Enqueue(ctx context.Context, notificationID uint) (int64, error)
	// Due 取出到期待发的邮件（按登记顺序，至多 limit 封）
	Due(ctx context.Context, now time.Time, limit int) ([]PendingEmail, error)
	MarkSent(ctx context.Context, id uint, at time.Time) error
	// MarkRetry 记下失败并安排下次尝试
	MarkRetry(ctx context.Context, id uint, attempts int, lastError string, next time.Time) error
	// MarkFinal 不再尝试：FAILED（尝试次数用尽）或 SKIPPED
	MarkFinal(ctx context.Context, id uint, status EmailStatus, attempts int, reason string) error
	// Stats 一条通知各状态的邮件数
	Stats(ctx context.Context, notificationID uint) (map[EmailStatus]int64, error)
	// StatusOf 一条通知给指定收件人的邮件状态
	StatusOf(ctx context.Context, notificationID uint, userIDs []uint) (map[uint]EmailStatus, error)
}
