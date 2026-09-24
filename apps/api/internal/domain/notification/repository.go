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
}

// UserNotificationRepository 用户通知仓储接口
//
// 全局通知采用「读取时惰性投递」：发布时不为每个用户写入 user_notifications 行，
// 读取路径直接以 notifications 为主表左连接用户关联行；只有用户实际操作
// （已读 / 删除）时才实体化该用户的关联行。
type UserNotificationRepository interface {
	// GetsByUserID 获取用户的通知列表（全局通知 + 定向投递给该用户的通知）
	GetsByUserID(ctx context.Context, userID uint, page, size int, order string, unreadOnly bool) ([]NotificationWithReadStatus, int64, error)
	// GetVisible 获取通知对该用户的可见状态：全局通知或已定向投递给该用户，
	// 且未被该用户删除、未过期。全局通知尚未实体化关联行时返回合成的初始状态。
	GetVisible(ctx context.Context, userID, notificationID uint) (*UserNotification, error)
	// Create 创建用户通知记录
	Create(ctx context.Context, userNotification *UserNotification) error
	// BatchCreate 批量创建用户通知记录
	BatchCreate(ctx context.Context, userNotifications []UserNotification) error
	// ReplaceRecipients 替换通知接收人
	ReplaceRecipients(ctx context.Context, notificationID uint, userIDs []uint) error
	// MarkAsRead 标记为已读（必要时先实体化全局通知的关联行）
	MarkAsRead(ctx context.Context, userID, notificationID uint) error
	// BatchMarkAsRead 批量标记为已读（必要时先实体化全局通知的关联行）
	BatchMarkAsRead(ctx context.Context, userID uint, notificationIDs []uint) error
	// MarkAllAsRead 标记全部为已读（必要时先实体化全局通知的关联行）
	MarkAllAsRead(ctx context.Context, userID uint) error
	// GetUnreadCount 获取未读数量（全局通知 + 定向投递给该用户的通知）
	GetUnreadCount(ctx context.Context, userID uint) (int64, error)
	// Dismiss 删除用户收件箱中的通知（必要时先实体化全局通知的关联行）
	Dismiss(ctx context.Context, userID, notificationID uint) error
	// StatsByNotificationID 获取通知投递统计；全局通知的接收人数按启用用户总数计算
	StatsByNotificationID(ctx context.Context, notificationID uint) (*NotificationDeliveryStats, error)
	// GetRecipients 获取通知接收人列表
	GetRecipients(ctx context.Context, notificationID uint, page, size int) ([]UserNotification, int64, error)
}
