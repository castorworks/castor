package handler

import (
	"fmt"
	"reflect"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

// AdminUserHandler 管理员用户处理器
type AdminUserHandler struct {
	UserService           service.UserService
	AuditLogService       service.AuditLogService
	NotificationPublisher service.NotificationPublisher
}

// NewAdminUserHandler 创建管理员用户处理器
func NewAdminUserHandler(userService service.UserService, auditLogService service.AuditLogService, notificationPublisher service.NotificationPublisher) *AdminUserHandler {
	return &AdminUserHandler{
		UserService:           userService,
		AuditLogService:       auditLogService,
		NotificationPublisher: notificationPublisher,
	}
}

// Gets 获取用户列表
func (h *AdminUserHandler) Gets(c *gin.Context) {
	GenericGets(c, reflect.TypeOf(user.User{}), func(ctx *gin.Context, page, size int, order string, opts ...query.Option) (interface{}, int64, error) {
		return h.UserService.Gets(ctx, page, size, order, opts...)
	})
}

// Get 获取用户详情
func (h *AdminUserHandler) Get(c *gin.Context) {
	GenericGet(c, func(ctx *gin.Context, id uint) (interface{}, error) {
		return h.UserService.Get(ctx, id)
	})
}

// Post 创建用户
func (h *AdminUserHandler) Post(c *gin.Context) {
	GenericPost(c, &dto.UserPostReq{}, func(ctx *gin.Context, req interface{}) (interface{}, error) {
		r := req.(*dto.UserPostReq)
		result, err := h.UserService.Post(ctx, r)
		h.logAudit(c, audit_log.AuditLogTypeUserCreate, r.Username,
			fmt.Sprintf("Created user: %s", r.Username), err == nil)
		if err == nil && result != nil && result.ID > 0 {
			h.publishUserCreatedNotification(c, result.ID)
		}
		return result, err
	}, func(req interface{}) bool {
		r := req.(*dto.UserPostReq)
		if r.Username == constant.CASTOR_RESERVED_USER_ROLE_NAME {
			return false
		}
		return r.Username != "" && r.Password != ""
	})
}

// Put 更新用户
func (h *AdminUserHandler) Put(c *gin.Context) {
	GenericPut(c, &dto.UserPutReq{}, func(ctx *gin.Context, id uint, req interface{}) (interface{}, error) {
		result, err := h.UserService.Put(ctx, id, req.(*dto.UserPutReq))
		h.logAudit(c, audit_log.AuditLogTypeUserUpdate, fmt.Sprintf("%d", id),
			fmt.Sprintf("Updated user ID: %d", id), err == nil)
		return result, err
	}, func(req interface{}) bool {
		r := req.(*dto.UserPutReq)
		return r.HasUpdatableFields()
	})
}

// Delete 删除用户
func (h *AdminUserHandler) Delete(c *gin.Context) {
	GenericDelete(c, func(ctx *gin.Context, id uint) error {
		err := h.UserService.Delete(ctx, id)
		h.logAudit(c, audit_log.AuditLogTypeUserDelete, fmt.Sprintf("%d", id),
			fmt.Sprintf("Deleted user ID: %d", id), err == nil)
		return err
	})
}

// logAudit 记录审计日志（异步）
func (h *AdminUserHandler) logAudit(c *gin.Context, logType audit_log.AuditLogType, target, details string, success bool) {
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

func (h *AdminUserHandler) publishUserCreatedNotification(c *gin.Context, userID uint) {
	if h.NotificationPublisher == nil {
		return
	}
	if err := h.NotificationPublisher.PublishToUsers(c.Request.Context(), service.NotificationPublishRequest{
		TemplateKey: service.NotificationTemplateAccountCreated,
		Type:        string(notification.TypeSystem),
		Level:       string(notification.LevelSuccess),
		Link:        "/dashboard/profile",
		UserIDs:     []uint{userID},
	}); err != nil {
		log.Err(c, err).Msg("Failed to publish notification")
	}
}
