package service

import (
	"context"
	"errors"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/asset"
	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/jinzhu/copier"
)

// NotificationService 通知服务接口
type NotificationService interface {
	// Channels 可用的投递通道
	Channels() dto.NotificationChannelsResp
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
	// PrepareAttachmentDownload 校验通知对该用户可见且 objectKey 确实是它的附件，返回下载描述
	PrepareAttachmentDownload(ctx context.Context, userID, notificationID uint, objectKey string) (*AssetDownloadResult, error)
	// PrepareAdminAttachmentDownload 管理端下载：通知存在且 objectKey 确实是它的附件（不看是否投递给调用者）
	PrepareAdminAttachmentDownload(ctx context.Context, notificationID uint, objectKey string) (*AssetDownloadResult, error)
}

// NotificationAttachments 通知附件字段：多值、私有，只有能看到这条通知的用户可下载
var NotificationAttachments = asset.Field{OwnerType: "notification", Name: "attachments"}

type notificationService struct {
	notificationRepo     notification.NotificationRepository
	userNotificationRepo notification.UserNotificationRepository
	assets               AssetReferencer
	// stream 实时推送；为 nil 时不推送（测试）
	stream NotificationStream
	// emails 邮件通道；为 nil 时视为未配置（测试）
	emails NotificationEmailService
}

// NewNotificationService 创建通知服务
func NewNotificationService(
	notificationRepo notification.NotificationRepository,
	userNotificationRepo notification.UserNotificationRepository,
	assets AssetService,
	stream NotificationStream,
	emails NotificationEmailService,
) NotificationService {
	return &notificationService{
		notificationRepo:     notificationRepo,
		userNotificationRepo: userNotificationRepo,
		assets:               assets,
		stream:               stream,
		emails:               emails,
	}
}

func (s *notificationService) Channels() dto.NotificationChannelsResp {
	return dto.NotificationChannelsResp{Email: s.emailAvailable()}
}

func (s *notificationService) emailAvailable() bool {
	return s.emails != nil && s.emails.Available()
}

// enqueueEmails 为要求发邮件的通知登记邮件；通知已经发出，登记失败只记日志（管理员可在详情里看到没有发件记录）。
func (s *notificationService) enqueueEmails(ctx context.Context, item *notification.Notification) {
	if !item.SendEmail || s.emails == nil {
		return
	}
	if err := s.emails.Enqueue(ctx, item.ID); err != nil {
		log.WarnCtx(ctx).Err(err).Uint("notificationID", item.ID).Msg("Failed to enqueue notification emails")
	}
}

// publish 通知实时连接；写库已成功，推送失败不影响结果。
func (s *notificationService) publish(ctx context.Context, event NotificationEvent) {
	if s.stream != nil {
		s.stream.Publish(ctx, event)
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
	if err := s.applyAttachments(ctx, resp); err != nil {
		return nil, 0, err
	}
	return resp, total, nil
}

func (s *notificationService) Get(ctx context.Context, id uint) (*dto.NotificationResp, error) {
	item, err := s.notificationRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.detailResp(ctx, item)
}

// detailResp 组装单条通知的完整响应（投递统计 + 附件）
func (s *notificationService) detailResp(ctx context.Context, item *notification.Notification) (*dto.NotificationResp, error) {
	resp := []dto.NotificationResp{s.notificationResp(item)}
	if err := s.applyDeliveryStats(ctx, item.ID, &resp[0]); err != nil {
		return nil, err
	}
	if err := s.applyAttachments(ctx, resp); err != nil {
		return nil, err
	}
	return &resp[0], nil
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
	if s.emails != nil && len(items) > 0 {
		ids := make([]uint, len(items))
		for i := range items {
			ids[i] = items[i].UserID
		}
		statuses, err := s.emails.StatusOf(ctx, notificationID, ids)
		if err != nil {
			return nil, 0, err
		}
		for i := range resp {
			resp[i].EmailStatus = string(statuses[resp[i].UserID])
		}
	}
	return resp, total, nil
}

func (s *notificationService) Post(ctx context.Context, req *dto.NotificationPostReq) (*dto.NotificationResp, error) {
	userIDs := uniqueUintIDs(req.UserIDs)
	if !req.IsGlobal && len(userIDs) == 0 {
		return nil, apperror.ErrNotificationRecipientsRequired
	}
	if req.SendEmail && !s.emailAvailable() {
		return nil, ErrDeliveryChannelNotConfigured
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
		SendEmail:    req.SendEmail,
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

	// 附件先于投递登记：对象键无效时通知还没人看得到，整条撤回即可。
	if len(req.Attachments) > 0 {
		if err := s.assets.Sync(ctx, NotificationAttachments.Of(item.ID), attachedFiles(req.Attachments), AttachOptions{}); err != nil {
			s.releaseAttachments(ctx, item.ID)
			if delErr := s.notificationRepo.Delete(ctx, item.ID); delErr != nil {
				log.WarnCtx(ctx).Err(delErr).Uint("notificationID", item.ID).Msg("Failed to roll back notification")
			}
			return nil, err
		}
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

	s.enqueueEmails(ctx, item)
	// 全局通知推给所有在线的人（UserIDs 为空）。
	s.publish(ctx, NotificationEvent{Kind: NotificationEventNew, UserIDs: userIDs, Notification: &NotificationSummary{
		ID: item.ID, Title: item.Title, Type: string(item.Type), Level: string(item.Level), Link: item.Link,
	}})
	return s.detailResp(ctx, item)
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
	if req.SendEmail != nil {
		if *req.SendEmail && !s.emailAvailable() {
			return nil, ErrDeliveryChannelNotConfigured
		}
		item.SendEmail = *req.SendEmail
	}

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
	if req.Attachments != nil {
		if err := s.assets.Sync(ctx, NotificationAttachments.Of(item.ID), attachedFiles(*req.Attachments), AttachOptions{}); err != nil {
			return nil, err
		}
	}

	// 打开了邮件或换了收件人：给还没登记过的收件人补发（已登记的不重复）。
	if req.SendEmail != nil || shouldReplaceRecipients {
		s.enqueueEmails(ctx, item)
	}
	// 收件人或过期时间可能变了：让在线的人重新取未读数。
	s.publish(ctx, NotificationEvent{Kind: NotificationEventChanged})
	return s.detailResp(ctx, item)
}

func (s *notificationService) Delete(ctx context.Context, id uint) error {
	if err := s.notificationRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.releaseAttachments(ctx, id)
	s.publish(ctx, NotificationEvent{Kind: NotificationEventChanged})
	return nil
}

func (s *notificationService) BatchDelete(ctx context.Context, ids []uint) error {
	if err := s.notificationRepo.BatchDelete(ctx, ids); err != nil {
		return err
	}
	for _, id := range ids {
		s.releaseAttachments(ctx, id)
	}
	s.publish(ctx, NotificationEvent{Kind: NotificationEventChanged})
	return nil
}

// releaseAttachments 通知已不存在，释放它引用的附件；失败只会留下可被清扫的孤儿，不影响删除结果。
func (s *notificationService) releaseAttachments(ctx context.Context, notificationID uint) {
	if err := s.assets.DetachOwner(ctx, NotificationAttachments.OwnerType, notificationID); err != nil {
		log.WarnCtx(ctx).Err(err).Uint("notificationID", notificationID).Msg("Failed to release notification attachments")
	}
}

func attachedFiles(reqs []dto.NotificationAttachmentReq) []asset.AttachedFile {
	files := make([]asset.AttachedFile, len(reqs))
	for i, req := range reqs {
		files[i] = asset.AttachedFile{ObjectKey: req.ObjectKey, Name: req.Name}
	}
	return files
}

// applyAttachments 一次查询为一页通知补上附件
func (s *notificationService) applyAttachments(ctx context.Context, resp []dto.NotificationResp) error {
	ids := make([]uint, len(resp))
	for i := range resp {
		ids[i] = resp[i].ID
	}
	attachments, err := s.assets.ListAttached(ctx, NotificationAttachments, ids)
	if err != nil {
		return err
	}
	for i := range resp {
		resp[i].Attachments = attachments[resp[i].ID]
		if resp[i].Attachments == nil {
			resp[i].Attachments = []dto.AssetAttachmentResp{}
		}
	}
	return nil
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
	if err := s.applyAttachments(ctx, resp); err != nil {
		return nil, 0, err
	}
	return resp, total, nil
}

func (s *notificationService) PrepareAttachmentDownload(ctx context.Context, userID, notificationID uint, objectKey string) (*AssetDownloadResult, error) {
	// 先确认这条通知对该用户可见，再确认文件确实挂在这条通知上：两者缺一，
	// 用户都能借一条自己看得到的通知读走别的文件。
	if _, err := s.getVisibleNotification(ctx, userID, notificationID); err != nil {
		return nil, err
	}
	return s.assets.PrepareAttachedDownload(ctx, objectKey, NotificationAttachments.Of(notificationID))
}

func (s *notificationService) PrepareAdminAttachmentDownload(ctx context.Context, notificationID uint, objectKey string) (*AssetDownloadResult, error) {
	if _, err := s.notificationRepo.Get(ctx, notificationID); err != nil {
		return nil, err
	}
	return s.assets.PrepareAttachedDownload(ctx, objectKey, NotificationAttachments.Of(notificationID))
}

func (s *notificationService) GetUnreadCount(ctx context.Context, userID uint) (int64, error) {
	return s.userNotificationRepo.GetUnreadCount(ctx, userID)
}

func (s *notificationService) MarkAsRead(ctx context.Context, userID, notificationID uint) error {
	if _, err := s.getVisibleNotification(ctx, userID, notificationID); err != nil {
		return err
	}
	if err := s.userNotificationRepo.MarkAsRead(ctx, userID, notificationID); err != nil {
		return err
	}
	s.publish(ctx, NotificationEvent{Kind: NotificationEventChanged, UserIDs: []uint{userID}})
	return nil
}

func (s *notificationService) BatchMarkAsRead(ctx context.Context, userID uint, notificationIDs []uint) error {
	for _, notificationID := range notificationIDs {
		if _, err := s.getVisibleNotification(ctx, userID, notificationID); err != nil {
			return err
		}
	}
	if err := s.userNotificationRepo.BatchMarkAsRead(ctx, userID, notificationIDs); err != nil {
		return err
	}
	s.publish(ctx, NotificationEvent{Kind: NotificationEventChanged, UserIDs: []uint{userID}})
	return nil
}

func (s *notificationService) MarkAllAsRead(ctx context.Context, userID uint) error {
	if err := s.userNotificationRepo.MarkAllAsRead(ctx, userID); err != nil {
		return err
	}
	s.publish(ctx, NotificationEvent{Kind: NotificationEventChanged, UserIDs: []uint{userID}})
	return nil
}

func (s *notificationService) DeleteUserNotification(ctx context.Context, userID, notificationID uint) error {
	if _, err := s.getVisibleNotification(ctx, userID, notificationID); err != nil {
		return err
	}
	if err := s.userNotificationRepo.Dismiss(ctx, userID, notificationID); err != nil {
		return err
	}
	s.publish(ctx, NotificationEvent{Kind: NotificationEventChanged, UserIDs: []uint{userID}})
	return nil
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
	if resp.SendEmail && s.emails != nil {
		emailStats, err := s.emails.Stats(ctx, notificationID)
		if err != nil {
			return err
		}
		resp.EmailStats = &dto.NotificationEmailStats{
			Pending: emailStats[notification.EmailPending], Sent: emailStats[notification.EmailSent],
			Failed: emailStats[notification.EmailFailed], Skipped: emailStats[notification.EmailSkipped],
		}
	}
	return nil
}

// getVisibleNotification 校验通知对该用户可见（全局通知或已定向投递），
// 不要求 user_notifications 中已存在关联行。
func (s *notificationService) getVisibleNotification(ctx context.Context, userID, notificationID uint) (*notification.UserNotification, error) {
	item, err := s.userNotificationRepo.GetVisible(ctx, userID, notificationID)
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
