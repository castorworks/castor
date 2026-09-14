package persistence

import (
	"context"
	"strings"
	"time"

	"github.com/castorworks/castor/internal/domain/notification"
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

func (r *notificationRepository) GetGlobalNotifications(ctx context.Context) ([]notification.Notification, error) {
	var ms []models.NotificationModel
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Where("is_global = ? AND (expire_at IS NULL OR expire_at > ?)", true, now).
		Order("created_at DESC").
		Find(&ms).Error; err != nil {
		return nil, err
	}
	result := make([]notification.Notification, len(ms))
	for i := range ms {
		result[i] = *ms[i].ToEntity()
	}
	return result, nil
}

// ========== UserNotificationRepository ==========

type userNotificationRepository struct {
	db *gorm.DB
}

// NewUserNotificationRepository 创建用户通知仓储
func NewUserNotificationRepository(db *gorm.DB) notification.UserNotificationRepository {
	return &userNotificationRepository{db: db}
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

	query := r.db.WithContext(ctx).
		Table("user_notifications un").
		Select("n.*, un.is_read, un.read_at, un.is_dismissed, un.dismissed_at").
		Joins("JOIN notifications n ON n.id = un.notification_id").
		Where("un.user_id = ?", userID).
		Where("un.is_dismissed = ?", false).
		Where("(n.expire_at IS NULL OR n.expire_at > ?)", time.Now())

	if unreadOnly {
		query = query.Where("un.is_read = ?", false)
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

func (r *userNotificationRepository) GetDelivered(ctx context.Context, userID, notificationID uint) (*notification.UserNotification, error) {
	var m models.UserNotificationModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND notification_id = ? AND is_dismissed = ?", userID, notificationID, false).
		First(&m).Error; err != nil {
		return nil, translateError(err)
	}
	return userNotificationModelToEntity(&m), nil
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

	// 统计用户的未读通知数量（包括全局通知和私有通知）
	err := r.db.WithContext(ctx).
		Table("user_notifications un").
		Joins("JOIN notifications n ON n.id = un.notification_id").
		Where("un.user_id = ?", userID).
		Where("un.is_dismissed = ?", false).
		Where("(n.expire_at IS NULL OR n.expire_at > ?)", time.Now()).
		Where("un.is_read = ?", false).
		Count(&count).Error

	return count, err
}

func (r *userNotificationRepository) Dismiss(ctx context.Context, userID, notificationID uint) error {
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
	err := r.db.WithContext(ctx).
		Table("user_notifications").
		Select(`
			COUNT(*) AS recipient_count,
			SUM(CASE WHEN is_read = true AND is_dismissed = false THEN 1 ELSE 0 END) AS read_count,
			SUM(CASE WHEN is_read = false AND is_dismissed = false THEN 1 ELSE 0 END) AS unread_count,
			SUM(CASE WHEN is_dismissed = true THEN 1 ELSE 0 END) AS dismissed_count
		`).
		Where("notification_id = ?", notificationID).
		Scan(&stats).Error
	return &stats, err
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
