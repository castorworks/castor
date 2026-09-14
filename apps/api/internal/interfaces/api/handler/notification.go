package handler

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

// AdminNotificationHandler 管理员通知处理器
type AdminNotificationHandler struct {
	NotificationService service.NotificationService
	AuditLogService     service.AuditLogService
}

// NewAdminNotificationHandler 创建管理员通知处理器
func NewAdminNotificationHandler(svc service.NotificationService, auditLogService service.AuditLogService) *AdminNotificationHandler {
	return &AdminNotificationHandler{
		NotificationService: svc,
		AuditLogService:     auditLogService,
	}
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
		h.logAudit(c, audit_log.AuditLogTypeNotificationCreate, r.Title,
			fmt.Sprintf("Created notification: %s", r.Title), err == nil)
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
		h.logAudit(c, audit_log.AuditLogTypeNotificationUpdate, fmt.Sprintf("%d", id),
			fmt.Sprintf("Updated notification ID: %d", id), err == nil)
		return result, err
	}, nil)
}

// Delete 删除通知
func (h *AdminNotificationHandler) Delete(c *gin.Context) {
	GenericDelete(c, func(ctx *gin.Context, id uint) error {
		err := h.NotificationService.Delete(ctx, id)
		h.logAudit(c, audit_log.AuditLogTypeNotificationDelete, fmt.Sprintf("%d", id),
			fmt.Sprintf("Deleted notification ID: %d", id), err == nil)
		return err
	})
}

// BatchDelete 批量删除通知
func (h *AdminNotificationHandler) BatchDelete(c *gin.Context) {
	GenericBatchDelete(c, &dto.NotificationBatchDeleteReq{}, func(ctx *gin.Context, req interface{}) error {
		ids := req.(*dto.NotificationBatchDeleteReq).Ids
		err := h.NotificationService.BatchDelete(ctx, ids)
		h.logAudit(c, audit_log.AuditLogTypeNotificationDelete, fmt.Sprintf("%v", ids),
			fmt.Sprintf("Batch deleted notification IDs: %v", ids), err == nil)
		return err
	}, func(req interface{}) bool {
		return len(req.(*dto.NotificationBatchDeleteReq).Ids) > 0
	})
}

func (h *AdminNotificationHandler) logAudit(c *gin.Context, logType audit_log.AuditLogType, target, details string, success bool) {
	if h.AuditLogService == nil {
		return
	}
	h.AuditLogService.LogAsync(&audit_log.AuditLog{
		LogType:    logType,
		Operator:   ucontext.GetUsername(c),
		OperatorID: ucontext.GetUserID(c),
		Target:     target,
		Details:    details,
		IpAddr:     getClientIP(c),
		Success:    success,
	})
}

// ========== 用户通知处理器 ==========

// NotificationHandler 用户通知处理器
type NotificationHandler struct {
	NotificationService service.NotificationService
}

// NewNotificationHandler 创建用户通知处理器
func NewNotificationHandler(svc service.NotificationService) *NotificationHandler {
	return &NotificationHandler{NotificationService: svc}
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
