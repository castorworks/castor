package notification

import (
	"context"

	"github.com/castorworks/castor/internal/pkg/query"
)

// NotificationRepository 通知仓储接口
type NotificationRepository interface {
	// Gets 分页查询通知列表
	Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]Notification, int64, error)
	// Get 根据 ID 获取通知
	Get(ctx context.Context, id uint) (*Notification, error)
	// Create 创建通知
	Create(ctx context.Context, notification *Notification) error
	// Update 更新通知
	Update(ctx context.Context, notification *Notification) error
	// Delete 删除通知
	Delete(ctx context.Context, id uint) error
	// BatchDelete 批量删除通知
	BatchDelete(ctx context.Context, ids []uint) error
	// GetGlobalNotifications 获取全局通知
	GetGlobalNotifications(ctx context.Context) ([]Notification, error)
}

// UserNotificationRepository 用户通知仓储接口
type UserNotificationRepository interface {
	// GetsByUserID 获取用户的通知列表
	GetsByUserID(ctx context.Context, userID uint, page, size int, order string, unreadOnly bool) ([]NotificationWithReadStatus, int64, error)
	// GetDelivered 获取用户已投递且未删除的通知记录
	GetDelivered(ctx context.Context, userID, notificationID uint) (*UserNotification, error)
	// Create 创建用户通知记录
	Create(ctx context.Context, userNotification *UserNotification) error
	// BatchCreate 批量创建用户通知记录
	BatchCreate(ctx context.Context, userNotifications []UserNotification) error
	// ReplaceRecipients 替换通知接收人
	ReplaceRecipients(ctx context.Context, notificationID uint, userIDs []uint) error
	// MarkAsRead 标记为已读
	MarkAsRead(ctx context.Context, userID, notificationID uint) error
	// BatchMarkAsRead 批量标记为已读
	BatchMarkAsRead(ctx context.Context, userID uint, notificationIDs []uint) error
	// MarkAllAsRead 标记全部为已读
	MarkAllAsRead(ctx context.Context, userID uint) error
	// GetUnreadCount 获取未读数量
	GetUnreadCount(ctx context.Context, userID uint) (int64, error)
	// Dismiss 删除用户收件箱中的通知
	Dismiss(ctx context.Context, userID, notificationID uint) error
	// StatsByNotificationID 获取通知投递统计
	StatsByNotificationID(ctx context.Context, notificationID uint) (*NotificationDeliveryStats, error)
	// GetRecipients 获取通知接收人列表
	GetRecipients(ctx context.Context, notificationID uint, page, size int) ([]UserNotification, int64, error)
}
