package persistence

import (
	"context"
	"strings"
	"time"

	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"github.com/castorworks/castor/internal/pkg/query"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ========== NotificationRepository ==========

type notificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository 创建通知仓储
func NewNotificationRepository(db *gorm.DB) notification.NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]notification.Notification, int64, error) {
	ms, total, err := PaginatedQuery[models.NotificationModel](ctx, r.db, page, size, order, opts...)
	if err != nil {
		return nil, 0, err
	}
	result := make([]notification.Notification, len(ms))
	for i := range ms {
		result[i] = *ms[i].ToEntity()
	}
	return result, total, nil
}

func (r *notificationRepository) Get(ctx context.Context, id uint) (*notification.Notification, error) {
	var m models.NotificationModel
	if err := translateError(r.db.WithContext(ctx).First(&m, id).Error); err != nil {
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *notificationRepository) Create(ctx context.Context, item *notification.Notification) error {
	m := models.NotificationModelFromEntity(item)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	item.ID = m.ID
	item.CreatedAt = m.CreatedAt
	item.UpdatedAt = m.UpdatedAt
	item.CreatedBy = m.CreatedBy
	item.UpdatedBy = m.UpdatedBy
	return nil
}

func (r *notificationRepository) Update(ctx context.Context, item *notification.Notification) error {
	m := models.NotificationModelFromEntity(item)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	item.UpdatedAt = m.UpdatedAt
	item.UpdatedBy = m.UpdatedBy
	return nil
}

func (r *notificationRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.NotificationModel{}, id).Error
}

func (r *notificationRepository) BatchDelete(ctx context.Context, ids []uint) error {
	return r.db.WithContext(ctx).Delete(&models.NotificationModel{}, ids).Error
}

// ========== UserNotificationRepository ==========

type userNotificationRepository struct {
	db *gorm.DB
}

// NewUserNotificationRepository 创建用户通知仓储
func NewUserNotificationRepository(db *gorm.DB) notification.UserNotificationRepository {
	return &userNotificationRepository{db: db}
}

// visibleJoin 以 notifications 为主表左连接当前用户的关联行。
// 全局通知无需预先写入 user_notifications 即可被用户看到（读取时惰性投递）。
func (r *userNotificationRepository) visibleJoin(ctx context.Context, userID uint) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("notifications n").
		Joins("LEFT JOIN user_notifications un ON un.notification_id = n.id AND un.user_id = ?", userID).
		Where("(n.is_global = ? OR un.id IS NOT NULL)", true).
		Where("COALESCE(un.is_dismissed, ?) = ?", false, false).
		Where("(n.expire_at IS NULL OR n.expire_at > ?)", time.Now())
}

// ensureGlobalRows 为该用户可见的全局通知补建 user_notifications 行，供后续
// 状态更新（已读 / 删除）落库。notificationIDs 为空表示全部全局通知。
// ON CONFLICT DO NOTHING 保证幂等与并发安全：已存在的行（含已删除的行）保持原状。
func (r *userNotificationRepository) ensureGlobalRows(ctx context.Context, userID uint, notificationIDs []uint) error {
	now := time.Now()
	sql := `INSERT INTO user_notifications (user_id, notification_id, is_read, is_dismissed, created_at)
		SELECT ?, n.id, false, false, ?
		FROM notifications n
		WHERE n.is_global = true AND (n.expire_at IS NULL OR n.expire_at > ?)`
	args := []interface{}{userID, now, now}
	if notificationIDs != nil {
		if len(notificationIDs) == 0 {
			return nil
		}
		sql += ` AND n.id IN ?`
		args = append(args, notificationIDs)
	}
	sql += ` ON CONFLICT (user_id, notification_id) DO NOTHING`
	return r.db.WithContext(ctx).Exec(sql, args...).Error
}

func (r *userNotificationRepository) GetsByUserID(ctx context.Context, userID uint, page, size int, order string, unreadOnly bool) ([]notification.NotificationWithReadStatus, int64, error) {
	var results []notification.NotificationWithReadStatus
	var total int64

	order, err := safeOrder[models.NotificationModel](r.db, order)
	if err != nil {
		return nil, 0, err
	}
	if order == "" {
		order = "created_at DESC"
	}
	// Both joined tables have created_at; qualify every validated column with the notifications alias.
	terms := strings.Split(order, ", ")
	for i := range terms {
		terms[i] = "n." + terms[i]
	}
	order = strings.Join(terms, ", ")

	query := r.visibleJoin(ctx, userID).
		Select("n.*, COALESCE(un.is_read, false) AS is_read, un.read_at, COALESCE(un.is_dismissed, false) AS is_dismissed, un.dismissed_at")

	if unreadOnly {
		query = query.Where("COALESCE(un.is_read, ?) = ?", false, false)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Order(order).Offset(offset).Limit(size).Scan(&results).Error; err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

func (r *userNotificationRepository) GetVisible(ctx context.Context, userID, notificationID uint) (*notification.UserNotification, error) {
	var row struct {
		ID          uint
		CreatedAt   *time.Time
		IsRead      bool
		ReadAt      *time.Time
		IsDismissed bool
		DismissedAt *time.Time
	}
	tx := r.visibleJoin(ctx, userID).
		Select("COALESCE(un.id, 0) AS id, un.created_at AS created_at, COALESCE(un.is_read, false) AS is_read, un.read_at AS read_at, COALESCE(un.is_dismissed, false) AS is_dismissed, un.dismissed_at AS dismissed_at").
		Where("n.id = ?", notificationID).
		Limit(1).
		Scan(&row)
	if tx.Error != nil {
		return nil, tx.Error
	}
	// 全局通知尚未实体化关联行时，RowsAffected 仍为 1，此处只有真正不可见才为 0。
	if tx.RowsAffected == 0 {
		return nil, shared.ErrNotFound
	}

	item := &notification.UserNotification{
		ID:             row.ID,
		UserID:         userID,
		NotificationID: notificationID,
		IsRead:         row.IsRead,
		ReadAt:         row.ReadAt,
		IsDismissed:    row.IsDismissed,
		DismissedAt:    row.DismissedAt,
	}
	if row.CreatedAt != nil {
		item.CreatedAt = *row.CreatedAt
	}
	return item, nil
}

func (r *userNotificationRepository) Create(ctx context.Context, item *notification.UserNotification) error {
	m := &models.UserNotificationModel{
		UserID:         item.UserID,
		NotificationID: item.NotificationID,
		IsRead:         item.IsRead,
		ReadAt:         item.ReadAt,
		IsDismissed:    item.IsDismissed,
		DismissedAt:    item.DismissedAt,
	}
	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(m).Error; err != nil {
		return err
	}
	item.ID = m.ID
	item.CreatedAt = m.CreatedAt
	return nil
}

func (r *userNotificationRepository) BatchCreate(ctx context.Context, items []notification.UserNotification) error {
	if len(items) == 0 {
		return nil
	}
	ms := make([]models.UserNotificationModel, len(items))
	for i := range items {
		ms[i] = models.UserNotificationModel{
			UserID:         items[i].UserID,
			NotificationID: items[i].NotificationID,
			IsRead:         items[i].IsRead,
			ReadAt:         items[i].ReadAt,
			IsDismissed:    items[i].IsDismissed,
			DismissedAt:    items[i].DismissedAt,
		}
	}
	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&ms).Error; err != nil {
		return err
	}
	for i := range items {
		items[i].ID = ms[i].ID
		items[i].CreatedAt = ms[i].CreatedAt
	}
	return nil
}

func (r *userNotificationRepository) ReplaceRecipients(ctx context.Context, notificationID uint, userIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("notification_id = ?", notificationID).Delete(&models.UserNotificationModel{}).Error; err != nil {
			return err
		}
		if len(userIDs) == 0 {
			return nil
		}
		rows := make([]models.UserNotificationModel, 0, len(userIDs))
		for _, userID := range userIDs {
			rows = append(rows, models.UserNotificationModel{
				UserID:         userID,
				NotificationID: notificationID,
			})
		}
		return tx.Create(&rows).Error
	})
}

func (r *userNotificationRepository) MarkAsRead(ctx context.Context, userID, notificationID uint) error {
	if err := r.ensureGlobalRows(ctx, userID, []uint{notificationID}); err != nil {
		return err
	}
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.UserNotificationModel{}).
		Where("user_id = ? AND notification_id = ? AND is_dismissed = ?", userID, notificationID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

func (r *userNotificationRepository) BatchMarkAsRead(ctx context.Context, userID uint, notificationIDs []uint) error {
	if len(notificationIDs) == 0 {
		return nil
	}
	if err := r.ensureGlobalRows(ctx, userID, notificationIDs); err != nil {
		return err
	}
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.UserNotificationModel{}).
		Where("user_id = ? AND notification_id IN ? AND is_dismissed = ?", userID, notificationIDs, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

func (r *userNotificationRepository) MarkAllAsRead(ctx context.Context, userID uint) error {
	// nil 表示补建全部未过期全局通知的关联行
	if err := r.ensureGlobalRows(ctx, userID, nil); err != nil {
		return err
	}
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.UserNotificationModel{}).
		Where("user_id = ? AND is_read = ? AND is_dismissed = ?", userID, false, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

func (r *userNotificationRepository) GetUnreadCount(ctx context.Context, userID uint) (int64, error) {
	var count int64

	// 统计用户的未读通知数量（包括全局通知和定向通知）
	err := r.visibleJoin(ctx, userID).
		Where("COALESCE(un.is_read, ?) = ?", false, false).
		Count(&count).Error

	return count, err
}

func (r *userNotificationRepository) Dismiss(ctx context.Context, userID, notificationID uint) error {
	if err := r.ensureGlobalRows(ctx, userID, []uint{notificationID}); err != nil {
		return err
	}
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.UserNotificationModel{}).
		Where("user_id = ? AND notification_id = ? AND is_dismissed = ?", userID, notificationID, false).
		Updates(map[string]interface{}{
			"is_dismissed": true,
			"dismissed_at": now,
		}).Error
}

func (r *userNotificationRepository) StatsByNotificationID(ctx context.Context, notificationID uint) (*notification.NotificationDeliveryStats, error) {
	var stats notification.NotificationDeliveryStats
	if err := r.db.WithContext(ctx).
		Table("user_notifications").
		Select(`
			COUNT(*) AS recipient_count,
			SUM(CASE WHEN is_read = true AND is_dismissed = false THEN 1 ELSE 0 END) AS read_count,
			SUM(CASE WHEN is_read = false AND is_dismissed = false THEN 1 ELSE 0 END) AS unread_count,
			SUM(CASE WHEN is_dismissed = true THEN 1 ELSE 0 END) AS dismissed_count
		`).
		Where("notification_id = ?", notificationID).
		Scan(&stats).Error; err != nil {
		return nil, err
	}

	// 全局通知按需投递，user_notifications 只有操作过的用户才有行；
	// 接收人数按启用用户总数计算，未读 = 接收人数 - 已读。
	var isGlobal bool
	if err := r.db.WithContext(ctx).
		Model(&models.NotificationModel{}).
		Select("is_global").
		Where("id = ?", notificationID).
		Scan(&isGlobal).Error; err != nil {
		return nil, err
	}
	if !isGlobal {
		return &stats, nil
	}

	var recipients int64
	if err := r.db.WithContext(ctx).
		Model(&models.UserModel{}).
		Where("enable = ?", true).
		Count(&recipients).Error; err != nil {
		return nil, err
	}
	stats.RecipientCount = recipients
	stats.UnreadCount = max(recipients-stats.ReadCount, 0)
	return &stats, nil
}

func (r *userNotificationRepository) GetRecipients(ctx context.Context, notificationID uint, page, size int) ([]notification.UserNotification, int64, error) {
	var ms []models.UserNotificationModel
	var total int64

	q := r.db.WithContext(ctx).
		Model(&models.UserNotificationModel{}).
		Where("notification_id = ?", notificationID)

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := q.Order("created_at DESC").
		Offset(offset).
		Limit(size).
		Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	result := make([]notification.UserNotification, len(ms))
	for i := range ms {
		result[i] = *userNotificationModelToEntity(&ms[i])
	}
	return result, total, nil
}

func userNotificationModelToEntity(m *models.UserNotificationModel) *notification.UserNotification {
	return &notification.UserNotification{
		ID:             m.ID,
		CreatedAt:      m.CreatedAt,
		UserID:         m.UserID,
		NotificationID: m.NotificationID,
		IsRead:         m.IsRead,
		ReadAt:         m.ReadAt,
		IsDismissed:    m.IsDismissed,
		DismissedAt:    m.DismissedAt,
	}
}
