package service

import (
	"context"
	"errors"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/jinzhu/copier"
)

// NotificationService 通知服务接口
type NotificationService interface {
	// 管理接口
	Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]dto.NotificationResp, int64, error)
	Get(ctx context.Context, id uint) (*dto.NotificationResp, error)
	GetRecipients(ctx context.Context, notificationID uint, page, size int) ([]dto.NotificationRecipientResp, int64, error)
	Post(ctx context.Context, req *dto.NotificationPostReq) (*dto.NotificationResp, error)
	Put(ctx context.Context, id uint, req *dto.NotificationPutReq) (*dto.NotificationResp, error)
	Delete(ctx context.Context, id uint) error
	BatchDelete(ctx context.Context, ids []uint) error

	// 用户接口
	GetUserNotifications(ctx context.Context, userID uint, page, size int, unreadOnly bool) ([]dto.NotificationResp, int64, error)
	GetUnreadCount(ctx context.Context, userID uint) (int64, error)
	MarkAsRead(ctx context.Context, userID, notificationID uint) error
	BatchMarkAsRead(ctx context.Context, userID uint, notificationIDs []uint) error
	MarkAllAsRead(ctx context.Context, userID uint) error
	DeleteUserNotification(ctx context.Context, userID, notificationID uint) error
}

type notificationService struct {
	notificationRepo     notification.NotificationRepository
	userNotificationRepo notification.UserNotificationRepository
}

// NewNotificationService 创建通知服务
func NewNotificationService(
	notificationRepo notification.NotificationRepository,
	userNotificationRepo notification.UserNotificationRepository,
) NotificationService {
	return &notificationService{
		notificationRepo:     notificationRepo,
		userNotificationRepo: userNotificationRepo,
	}
}

// ========== 管理接口 ==========

func (s *notificationService) Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]dto.NotificationResp, int64, error) {
	items, total, err := s.notificationRepo.Gets(ctx, page, size, order, opts...)
	if err != nil {
		return nil, 0, err
	}

	var resp []dto.NotificationResp
	for _, item := range items {
		r := s.notificationResp(&item)
		if err := s.applyDeliveryStats(ctx, item.ID, &r); err != nil {
			return nil, 0, err
		}
		resp = append(resp, r)
	}
	return resp, total, nil
}

func (s *notificationService) Get(ctx context.Context, id uint) (*dto.NotificationResp, error) {
	item, err := s.notificationRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := s.notificationResp(item)
	if err := s.applyDeliveryStats(ctx, item.ID, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *notificationService) GetRecipients(ctx context.Context, notificationID uint, page, size int) ([]dto.NotificationRecipientResp, int64, error) {
	items, total, err := s.userNotificationRepo.GetRecipients(ctx, notificationID, page, size)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]dto.NotificationRecipientResp, len(items))
	for i, item := range items {
		resp[i] = dto.NotificationRecipientResp{
			UserID:      item.UserID,
			IsRead:      item.IsRead,
			ReadAt:      item.ReadAt,
			IsDismissed: item.IsDismissed,
			DismissedAt: item.DismissedAt,
			DeliveredAt: item.CreatedAt,
		}
	}
	return resp, total, nil
}

func (s *notificationService) Post(ctx context.Context, req *dto.NotificationPostReq) (*dto.NotificationResp, error) {
	userIDs := uniqueUintIDs(req.UserIDs)
	if !req.IsGlobal && len(userIDs) == 0 {
		return nil, apperror.ErrNotificationRecipientsRequired
	}

	item := &notification.Notification{
		Title:        req.Title,
		Content:      req.Content,
		TemplateKey:  req.TemplateKey,
		TemplateData: req.TemplateData,
		Type:         notification.NotificationType(req.Type),
		Level:        notification.NotificationLevel(req.Level),
		Link:         req.Link,
		Extra:        req.Extra,
		IsGlobal:     req.IsGlobal,
		ExpireAt:     req.ExpireAt,
	}

	// 设置默认值
	if item.Type == "" {
		item.Type = notification.TypeSystem
	}
	if item.Level == "" {
		item.Level = notification.LevelInfo
	}

	if err := s.notificationRepo.Create(ctx, item); err != nil {
		return nil, err
	}

	// 如果不是全局通知，需要创建用户通知记录
	if len(userIDs) > 0 {
		var userNotifications []notification.UserNotification
		for _, userID := range userIDs {
			userNotifications = append(userNotifications, notification.UserNotification{
				UserID:         userID,
				NotificationID: item.ID,
			})
		}
		if err := s.userNotificationRepo.BatchCreate(ctx, userNotifications); err != nil {
			return nil, err
		}
	}

	resp := s.notificationResp(item)
	if err := s.applyDeliveryStats(ctx, item.ID, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *notificationService) Put(ctx context.Context, id uint, req *dto.NotificationPutReq) (*dto.NotificationResp, error) {
	item, err := s.notificationRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Title 是必填字段，只有非空时才更新
	if req.Title != "" {
		item.Title = req.Title
	}
	// 以下字段允许清空，直接赋值
	item.Content = req.Content
	item.TemplateKey = req.TemplateKey
	item.TemplateData = req.TemplateData
	if req.Type != "" {
		item.Type = notification.NotificationType(req.Type)
	}
	if req.Level != "" {
		item.Level = notification.NotificationLevel(req.Level)
	}
	item.Link = req.Link
	item.Extra = req.Extra
	if req.IsGlobal != nil {
		item.IsGlobal = *req.IsGlobal
	}
	item.ExpireAt = req.ExpireAt

	userIDs := uniqueUintIDs(req.UserIDs)
	shouldReplaceRecipients := req.IsGlobal != nil || req.UserIDs != nil
	if shouldReplaceRecipients && !item.IsGlobal && len(userIDs) == 0 {
		return nil, apperror.ErrNotificationRecipientsRequired
	}

	if err := s.notificationRepo.Update(ctx, item); err != nil {
		return nil, err
	}
	if shouldReplaceRecipients {
		if err := s.userNotificationRepo.ReplaceRecipients(ctx, item.ID, userIDs); err != nil {
			return nil, err
		}
	}

	resp := s.notificationResp(item)
	if err := s.applyDeliveryStats(ctx, item.ID, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *notificationService) Delete(ctx context.Context, id uint) error {
	return s.notificationRepo.Delete(ctx, id)
}

func (s *notificationService) BatchDelete(ctx context.Context, ids []uint) error {
	return s.notificationRepo.BatchDelete(ctx, ids)
}

// ========== 用户接口 ==========

func (s *notificationService) GetUserNotifications(ctx context.Context, userID uint, page, size int, unreadOnly bool) ([]dto.NotificationResp, int64, error) {
	items, total, err := s.userNotificationRepo.GetsByUserID(ctx, userID, page, size, "created_at DESC", unreadOnly)
	if err != nil {
		return nil, 0, err
	}

	var resp []dto.NotificationResp
	for _, item := range items {
		r := dto.NotificationResp{
			ID:          item.ID,
			CreatedAt:   item.CreatedAt,
			CreatedBy:   item.CreatedBy,
			UpdatedAt:   item.UpdatedAt,
			UpdatedBy:   item.UpdatedBy,
			Title:       item.Title,
			Content:     item.Content,
			Type:        string(item.Type),
			Level:       string(item.Level),
			Link:        item.Link,
			Extra:       item.Extra,
			SenderID:    item.SenderID,
			IsGlobal:    item.IsGlobal,
			ExpireAt:    item.ExpireAt,
			IsRead:      item.IsRead,
			ReadAt:      item.ReadAt,
			IsDismissed: item.IsDismissed,
			DismissedAt: item.DismissedAt,
		}
		resp = append(resp, r)
	}
	return resp, total, nil
}

func (s *notificationService) GetUnreadCount(ctx context.Context, userID uint) (int64, error) {
	return s.userNotificationRepo.GetUnreadCount(ctx, userID)
}

func (s *notificationService) MarkAsRead(ctx context.Context, userID, notificationID uint) error {
	if _, err := s.getDeliveredNotification(ctx, userID, notificationID); err != nil {
		return err
	}
	return s.userNotificationRepo.MarkAsRead(ctx, userID, notificationID)
}

func (s *notificationService) BatchMarkAsRead(ctx context.Context, userID uint, notificationIDs []uint) error {
	for _, notificationID := range notificationIDs {
		if _, err := s.getDeliveredNotification(ctx, userID, notificationID); err != nil {
			return err
		}
	}
	return s.userNotificationRepo.BatchMarkAsRead(ctx, userID, notificationIDs)
}

func (s *notificationService) MarkAllAsRead(ctx context.Context, userID uint) error {
	return s.userNotificationRepo.MarkAllAsRead(ctx, userID)
}

func (s *notificationService) DeleteUserNotification(ctx context.Context, userID, notificationID uint) error {
	if _, err := s.getDeliveredNotification(ctx, userID, notificationID); err != nil {
		return err
	}
	return s.userNotificationRepo.Dismiss(ctx, userID, notificationID)
}

func (s *notificationService) notificationResp(item *notification.Notification) dto.NotificationResp {
	var resp dto.NotificationResp
	copier.Copy(&resp, item)
	resp.Type = string(item.Type)
	resp.Level = string(item.Level)
	return resp
}

func (s *notificationService) applyDeliveryStats(ctx context.Context, notificationID uint, resp *dto.NotificationResp) error {
	stats, err := s.userNotificationRepo.StatsByNotificationID(ctx, notificationID)
	if err != nil {
		return err
	}
	resp.RecipientCount = stats.RecipientCount
	resp.ReadCount = stats.ReadCount
	resp.UnreadCount = stats.UnreadCount
	resp.DismissedCount = stats.DismissedCount
	return nil
}

func (s *notificationService) getDeliveredNotification(ctx context.Context, userID, notificationID uint) (*notification.UserNotification, error) {
	item, err := s.userNotificationRepo.GetDelivered(ctx, userID, notificationID)
	if err == nil {
		return item, nil
	}
	if errors.Is(err, shared.ErrNotFound) {
		return nil, apperror.ErrNotificationNotDelivered
	}
	return nil, err
}

func uniqueUintIDs(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
