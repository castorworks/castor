package handler

import (
	"fmt"
	"reflect"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/domain/permission"
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
	RBAC                  service.RBACService
	Departments           service.DepartmentService
	Dictionary            service.DictionaryService
	MFA                   service.MFAService
}

// NewAdminUserHandler 创建管理员用户处理器
func NewAdminUserHandler(userService service.UserService, auditLogService service.AuditLogService, notificationPublisher service.NotificationPublisher, rbac service.RBACService, departments service.DepartmentService, dictionary service.DictionaryService, mfa service.MFAService) *AdminUserHandler {
	return &AdminUserHandler{
		UserService:           userService,
		AuditLogService:       auditLogService,
		NotificationPublisher: notificationPublisher,
		RBAC:                  rbac,
		Departments:           departments,
		Dictionary:            dictionary,
		MFA:                   mfa,
	}
}

// Gets 获取用户列表（限于调用者的数据范围）
func (h *AdminUserHandler) Gets(c *gin.Context) {
	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	GenericGets(c, reflect.TypeOf(user.User{}), func(ctx *gin.Context, page, size int, order string, opts ...query.Option) (interface{}, int64, error) {
		items, total, err := h.UserService.Gets(ctx, scope, page, size, order, opts...)
		if err != nil {
			return nil, 0, err
		}
		return items, total, h.markTOTP(ctx, items)
	})
}

// Get 获取用户详情
func (h *AdminUserHandler) Get(c *gin.Context) {
	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	GenericGet(c, func(ctx *gin.Context, id uint) (interface{}, error) {
		item, err := h.UserService.Get(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		items := []dto.UserResp{*item}
		if err := h.markTOTP(ctx, items); err != nil {
			return nil, err
		}
		return &items[0], nil
	})
}

// Post 创建用户
func (h *AdminUserHandler) Post(c *gin.Context) {
	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	GenericPost(c, &dto.UserPostReq{}, func(ctx *gin.Context, req interface{}) (interface{}, error) {
		r := req.(*dto.UserPostReq)
		result, err := h.UserService.Post(ctx, scope, r)
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeUserCreate, r.Username,
			fmt.Sprintf("Create user: %s", r.Username), err)
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
	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	GenericPut(c, &dto.UserPutReq{}, func(ctx *gin.Context, id uint, req interface{}) (interface{}, error) {
		result, err := h.UserService.Put(ctx, scope, id, req.(*dto.UserPutReq))
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeUserUpdate, fmt.Sprintf("%d", id),
			fmt.Sprintf("Update user ID: %d", id), err)
		return result, err
	}, func(req interface{}) bool {
		r := req.(*dto.UserPutReq)
		return r.HasUpdatableFields()
	})
}

// Delete 删除用户
func (h *AdminUserHandler) Delete(c *gin.Context) {
	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	GenericDelete(c, func(ctx *gin.Context, id uint) error {
		err := h.UserService.Delete(ctx, scope, id)
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeUserDelete, fmt.Sprintf("%d", id),
			fmt.Sprintf("Delete user ID: %d", id), err)
		return err
	})
}

func (h *AdminUserHandler) publishUserCreatedNotification(c *gin.Context, userIDs ...uint) {
	if h.NotificationPublisher == nil || len(userIDs) == 0 {
		return
	}
	if err := h.NotificationPublisher.PublishToUsers(c.Request.Context(), service.NotificationPublishRequest{
		TemplateKey: service.NotificationTemplateAccountCreated,
		Type:        string(notification.TypeSystem),
		Level:       string(notification.LevelSuccess),
		Link:        "/dashboard/profile",
		UserIDs:     userIDs,
	}); err != nil {
		log.Err(c, err).Msg("Failed to publish notification")
	}
}

func (h *AdminUserHandler) resetTOTP(c *gin.Context, scope permission.AccessScope, id uint) error {
	target, err := h.UserService.Get(c.Request.Context(), scope, id)
	if err != nil {
		return err
	}
	if err := h.RBAC.EnsureCanManageUser(c.Request.Context(), ucontext.GetUserID(c), target.ID); err != nil {
		return err
	}
	return h.MFA.ResetTOTP(c.Request.Context(), target.ID)
}

// markTOTP 标出哪些用户开启了两步验证（一次查询）
func (h *AdminUserHandler) markTOTP(c *gin.Context, items []dto.UserResp) error {
	if h.MFA == nil {
		return nil
	}
	ids := make([]uint, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	enabled, err := h.MFA.EnabledAmong(c.Request.Context(), ids)
	if err != nil {
		return err
	}
	for i := range items {
		on := enabled[items[i].ID]
		items[i].TOTPEnabled = &on
	}
	return nil
}
