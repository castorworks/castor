package handler

import (
	"reflect"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/login_history"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

// LoginHistoryHandler 用户登录历史处理器（用户查询自己的登录记录）
type LoginHistoryHandler struct {
	LoginHistoryService service.LoginHistoryService
}

// NewLoginHistoryHandler 创建用户登录历史处理器
func NewLoginHistoryHandler(svc service.LoginHistoryService) *LoginHistoryHandler {
	return &LoginHistoryHandler{LoginHistoryService: svc}
}

// Gets 用户查询自己的登录历史
func (h *LoginHistoryHandler) Gets(c *gin.Context) {
	username := ucontext.GetUsername(c)
	if username == "" {
		return
	}

	GenericGets(c, reflect.TypeOf(login_history.LoginHistory{}), func(ctx *gin.Context, page, size int, order string, opts ...query.Option) (interface{}, int64, error) {
		// 强制追加 username 过滤条件
		usernameOpt := *query.NewOption("username = ?", username)
		opts = append(opts, usernameOpt)
		return h.LoginHistoryService.Gets(ctx, page, size, order, opts...)
	})
}

// AdminLoginHistoryHandler 管理员登录历史处理器
type AdminLoginHistoryHandler struct {
	LoginHistoryService service.LoginHistoryService
}

// NewAdminLoginHistoryHandler 创建管理员登录历史处理器
func NewAdminLoginHistoryHandler(svc service.LoginHistoryService) *AdminLoginHistoryHandler {
	return &AdminLoginHistoryHandler{LoginHistoryService: svc}
}

// Gets 获取登录历史列表
func (h *AdminLoginHistoryHandler) Gets(c *gin.Context) {
	GenericGets(c, reflect.TypeOf(login_history.LoginHistory{}), func(ctx *gin.Context, page, size int, order string, opts ...query.Option) (interface{}, int64, error) {
		return h.LoginHistoryService.Gets(ctx, page, size, order, opts...)
	})
}

// Delete 删除登录历史
func (h *AdminLoginHistoryHandler) Delete(c *gin.Context) {
	GenericDelete(c, func(ctx *gin.Context, id uint) error {
		return h.LoginHistoryService.Delete(ctx, id)
	})
}

// BatchDelete 批量删除登录历史
func (h *AdminLoginHistoryHandler) BatchDelete(c *gin.Context) {
	GenericBatchDelete(c, &dto.LoginHistoryBatchDeleteReq{}, func(ctx *gin.Context, req interface{}) error {
		ids := req.(*dto.LoginHistoryBatchDeleteReq).Ids
		return h.LoginHistoryService.BatchDelete(ctx, ids)
	}, func(req interface{}) bool {
		return len(req.(*dto.LoginHistoryBatchDeleteReq).Ids) > 0
	})
}
