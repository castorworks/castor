package persistence

import (
	"context"
	"time"

	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

type notificationEmailRepository struct{ db *gorm.DB }

func NewNotificationEmailRepository(db *gorm.DB) notification.EmailRepository {
	return &notificationEmailRepository{db: db}
}

// eligibleRecipients 能收通知邮件的用户：启用、邮箱已验证且非空、没有关闭通知邮件。
const eligibleRecipients = `u.enable = true AND u.email_verified = true AND u.email <> '' AND u.mute_notification_emails = false`

func (r *notificationEmailRepository) Enqueue(ctx context.Context, notificationID uint) (int64, error) {
	now := time.Now()
	// 收件人取自通知本身：全局通知为全部用户，否则为 user_notifications 里定向投递的人。
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO notification_emails (notification_id, user_id, email, status, attempts, last_error, next_attempt_at, created_at)
		SELECT n.id, u.id, u.email, ?, 0, '', ?, ?
		FROM notifications n JOIN users u ON (n.is_global OR u.id IN (SELECT un.user_id FROM user_notifications un WHERE un.notification_id = n.id))
		WHERE n.id = ? AND n.send_email AND `+eligibleRecipients+`
		ON CONFLICT (notification_id, user_id) DO NOTHING`, string(notification.EmailPending), now, now, notificationID)
	return result.RowsAffected, result.Error
}

type pendingRow struct {
	models.NotificationEmailModel
	NotifTitle    string
	NotifContent  string
	NotifLink     string
	NotifType     string
	NotifLevel    string
	NotifExpireAt *time.Time
	UserName      string
	Username      string
	StillWanted   bool
}

func (r *notificationEmailRepository) Due(ctx context.Context, now time.Time, limit int) ([]notification.PendingEmail, error) {
	var rows []pendingRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT e.*, n.title AS notif_title, n.content AS notif_content, n.link AS notif_link, n.type AS notif_type,
			n.level AS notif_level, n.expire_at AS notif_expire_at, u.name AS user_name, u.username AS username,
			(`+eligibleRecipients+` AND u.email = e.email) AS still_wanted
		FROM notification_emails e
		JOIN notifications n ON n.id = e.notification_id
		JOIN users u ON u.id = e.user_id
		WHERE e.status = ? AND e.next_attempt_at <= ?
		ORDER BY e.id
		LIMIT ?`, string(notification.EmailPending), now, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]notification.PendingEmail, len(rows))
	for i, row := range rows {
		name := row.UserName
		if name == "" {
			name = row.Username
		}
		result[i] = notification.PendingEmail{
			EmailDelivery: row.NotificationEmailModel.ToEntity(),
			Notification: notification.Notification{
				ID: row.NotificationID, Title: row.NotifTitle, Content: row.NotifContent, Link: row.NotifLink,
				Type: notification.NotificationType(row.NotifType), Level: notification.NotificationLevel(row.NotifLevel),
				ExpireAt: row.NotifExpireAt, SendEmail: true,
			},
			RecipientName: name,
			StillWanted:   row.StillWanted,
		}
	}
	return result, nil
}

func (r *notificationEmailRepository) update(ctx context.Context, id uint, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&models.NotificationEmailModel{}).Where("id = ?", id).Updates(fields).Error
}

func (r *notificationEmailRepository) MarkSent(ctx context.Context, id uint, at time.Time) error {
	return r.update(ctx, id, map[string]any{"status": string(notification.EmailSent), "sent_at": at, "attempts": gorm.Expr("attempts + 1"), "last_error": ""})
}

func (r *notificationEmailRepository) MarkRetry(ctx context.Context, id uint, attempts int, lastError string, next time.Time) error {
	return r.update(ctx, id, map[string]any{"attempts": attempts, "last_error": lastError, "next_attempt_at": next})
}

func (r *notificationEmailRepository) MarkFinal(ctx context.Context, id uint, status notification.EmailStatus, attempts int, reason string) error {
	return r.update(ctx, id, map[string]any{"status": string(status), "attempts": attempts, "last_error": reason})
}

func (r *notificationEmailRepository) Stats(ctx context.Context, notificationID uint) (map[notification.EmailStatus]int64, error) {
	var rows []struct {
		Status string
		Count  int64
	}
	if err := r.db.WithContext(ctx).Model(&models.NotificationEmailModel{}).Select("status, COUNT(*) AS count").
		Where("notification_id = ?", notificationID).Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := map[notification.EmailStatus]int64{}
	for _, row := range rows {
		result[notification.EmailStatus(row.Status)] = row.Count
	}
	return result, nil
}

func (r *notificationEmailRepository) StatusOf(ctx context.Context, notificationID uint, userIDs []uint) (map[uint]notification.EmailStatus, error) {
	result := map[uint]notification.EmailStatus{}
	if len(userIDs) == 0 {
		return result, nil
	}
	var rows []models.NotificationEmailModel
	if err := r.db.WithContext(ctx).Where("notification_id = ? AND user_id IN ?", notificationID, userIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.UserID] = notification.EmailStatus(row.Status)
	}
	return result, nil
}
