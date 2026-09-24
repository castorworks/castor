package handler

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/castorworks/castor/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// AdminNotificationHandler 管理员通知处理器
type AdminNotificationHandler struct {
	NotificationService service.NotificationService
	AssetService        service.AssetService
	AuditLogService     service.AuditLogService
}

// Channels 可用的投递通道（表单据此决定能否勾选"同时发邮件"）
// GET /api/v1/admin/notifications/channels
func (h *AdminNotificationHandler) Channels(c *gin.Context) {
	response.Success(c, h.NotificationService.Channels())
}

// NewAdminNotificationHandler 创建管理员通知处理器
func NewAdminNotificationHandler(svc service.NotificationService, assetService service.AssetService, auditLogService service.AuditLogService) *AdminNotificationHandler {
	return &AdminNotificationHandler{
		NotificationService: svc,
		AssetService:        assetService,
		AuditLogService:     auditLogService,
	}
}

// PostAttachment 上传通知附件。通知模块自己的上传通道：能发通知的人不必拥有资产库的权限。
// 文件以私有业务附件入库，随后由创建/更新通知的请求按 objectKey 挂到通知上；
// 最终没被任何通知用上的文件由孤儿清扫回收。
func (h *AdminNotificationHandler) PostAttachment(c *gin.Context) {
	f, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}
	defer f.Close()

	if !validator.IsValidFilename(fileHeader.Filename) {
		response.BadRequestI18n(c, response.ErrInvalidFilename)
		return
	}

	resp, err := h.AssetService.UploadAttachment(c.Request.Context(), service.AttachmentUpload{
		Filename:    fileHeader.Filename,
		File:        f,
		ContentType: fileHeader.Header.Get("Content-Type"),
		Size:        fileHeader.Size,
	})
	// 审计记录服务端生成的对象键，不记录客户端提交的文件名。
	target := "-"
	if err == nil {
		target = resp.ObjectKey
	}
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypeAssetCreate, target, "Upload notification attachment", err)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	// 只回传附件视图：去重命中时这条资产的名称、上传人等属于别人，不该透给上传者。
	attachment := dto.AssetAttachmentResp{
		ObjectKey:     resp.ObjectKey,
		Name:          fileHeader.Filename,
		Extension:     resp.Extension,
		MimeType:      resp.MimeType,
		Size:          resp.Size,
		SizeFormatted: resp.SizeFormatted,
		Category:      resp.Category,
	}
	response.SuccessI18n(c, attachment, response.InfFileUploaded)
}

// Gets 获取通知列表
func (h *AdminNotificationHandler) Gets(c *gin.Context) {
	GenericGets(c, reflect.TypeOf(notification.Notification{}), func(ctx *gin.Context, page, size int, order string, opts ...query.Option) (interface{}, int64, error) {
		return h.NotificationService.Gets(ctx, page, size, order, opts...)
	})
}

// Get 获取单个通知
func (h *AdminNotificationHandler) Get(c *gin.Context) {
	GenericGet(c, func(ctx *gin.Context, id uint) (interface{}, error) {
		return h.NotificationService.Get(ctx, id)
	})
}

// DownloadAttachment 管理员在编辑表单里下载通知已保存的附件。与收件人的下载一样只认
// "这个文件确实挂在这条通知上"，任何失败都回空 404；资产库权限不是前提。
// GET /api/v1/admin/notifications/:id/attachments/:objectKey
func (h *AdminNotificationHandler) DownloadAttachment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	result, err := h.NotificationService.PrepareAdminAttachmentDownload(c.Request.Context(), uint(id), c.Param("objectKey"))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	streamAsset(c, h.AssetService, result)
}

// GetRecipients 获取通知接收人列表
func (h *AdminNotificationHandler) GetRecipients(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}

	page, pageSize := ParsePageParams(c)

	items, total, err := h.NotificationService.GetRecipients(c, uint(id), page, pageSize)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessListPaged(c, total, items, page, pageSize)
}

// Post 创建通知
func (h *AdminNotificationHandler) Post(c *gin.Context) {
	GenericPost(c, &dto.NotificationPostReq{}, func(ctx *gin.Context, req interface{}) (interface{}, error) {
		r := req.(*dto.NotificationPostReq)
		result, err := h.NotificationService.Post(ctx, r)
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeNotificationCreate, r.Title,
			fmt.Sprintf("Create notification: %s", r.Title), err)
		return result, err
	}, func(req interface{}) bool {
		r := req.(*dto.NotificationPostReq)
		return r.Title != ""
	})
}

// Put 更新通知
func (h *AdminNotificationHandler) Put(c *gin.Context) {
	GenericPut(c, &dto.NotificationPutReq{}, func(ctx *gin.Context, id uint, req interface{}) (interface{}, error) {
		result, err := h.NotificationService.Put(ctx, id, req.(*dto.NotificationPutReq))
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeNotificationUpdate, fmt.Sprintf("%d", id),
			fmt.Sprintf("Update notification ID: %d", id), err)
		return result, err
	}, nil)
}

// Delete 删除通知
func (h *AdminNotificationHandler) Delete(c *gin.Context) {
	GenericDelete(c, func(ctx *gin.Context, id uint) error {
		err := h.NotificationService.Delete(ctx, id)
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeNotificationDelete, fmt.Sprintf("%d", id),
			fmt.Sprintf("Delete notification ID: %d", id), err)
		return err
	})
}

// BatchDelete 批量删除通知
func (h *AdminNotificationHandler) BatchDelete(c *gin.Context) {
	GenericBatchDelete(c, &dto.NotificationBatchDeleteReq{}, func(ctx *gin.Context, req interface{}) error {
		ids := req.(*dto.NotificationBatchDeleteReq).Ids
		err := h.NotificationService.BatchDelete(ctx, ids)
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeNotificationDelete, joinAuditIDs(ids),
			fmt.Sprintf("Batch delete %d notifications", len(ids)), err)
		return err
	}, func(req interface{}) bool {
		return len(req.(*dto.NotificationBatchDeleteReq).Ids) > 0
	})
}

// ========== 用户通知处理器 ==========

// NotificationHandler 用户通知处理器
type NotificationHandler struct {
	NotificationService service.NotificationService
	AssetService        service.AssetService
}

// NewNotificationHandler 创建用户通知处理器
func NewNotificationHandler(svc service.NotificationService, assetService service.AssetService) *NotificationHandler {
	return &NotificationHandler{NotificationService: svc, AssetService: assetService}
}

// DownloadAttachment 下载通知附件。附件是私有资产：通知对当前用户不可见、
// 或文件并不属于这条通知时，一律表现为不存在。
func (h *NotificationHandler) DownloadAttachment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	result, err := h.NotificationService.PrepareAttachmentDownload(c.Request.Context(), ucontext.GetUserID(c), uint(id), c.Param("objectKey"))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	streamAsset(c, h.AssetService, result)
}

// Gets 获取当前用户的通知列表
func (h *NotificationHandler) Gets(c *gin.Context) {
	userID := ucontext.GetUserID(c)

	page, pageSize := ParsePageParams(c)
	unreadOnly := c.Query("unreadOnly") == "true"

	items, total, err := h.NotificationService.GetUserNotifications(c, userID, page, pageSize, unreadOnly)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessListPaged(c, total, items, page, pageSize)
}

// GetUnreadCount 获取未读通知数量
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID := ucontext.GetUserID(c)

	count, err := h.NotificationService.GetUnreadCount(c, userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, dto.NotificationUnreadCountResp{Count: count})
}

// MarkAsRead 标记通知为已读
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID := ucontext.GetUserID(c)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if err := h.NotificationService.MarkAsRead(c, userID, uint(id)); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessI18n(c, gin.H{}, response.InfNotificationRead)
}

// BatchMarkAsRead 批量标记通知为已读
func (h *NotificationHandler) BatchMarkAsRead(c *gin.Context) {
	userID := ucontext.GetUserID(c)

	var req dto.MarkNotificationsReadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if err := h.NotificationService.BatchMarkAsRead(c, userID, req.Ids); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessI18n(c, gin.H{}, response.InfNotificationRead)
}

// MarkAllAsRead 标记所有通知为已读
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := ucontext.GetUserID(c)

	if err := h.NotificationService.MarkAllAsRead(c, userID); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessI18n(c, gin.H{}, response.InfAllMarkedRead)
}

// Delete 删除用户通知
func (h *NotificationHandler) Delete(c *gin.Context) {
	userID := ucontext.GetUserID(c)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if err := h.NotificationService.DeleteUserNotification(c, userID, uint(id)); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessI18n(c, gin.H{}, response.InfDeleteSuccess)
}
