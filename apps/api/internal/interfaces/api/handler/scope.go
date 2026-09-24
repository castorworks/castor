package handler

import (
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

// requestScope 解析当前授权会话的数据范围（激活角色的并集）。解析失败时已写好错误响应，
// 调用方直接返回即可。
func requestScope(c *gin.Context, rbac service.RBACService) (permission.AccessScope, bool) {
	scope, err := rbac.SessionScope(c.Request.Context(), ucontext.GetAuthorizationSessionID(c), ucontext.GetUserID(c))
	if err != nil {
		response.HandleError(c, err)
		return permission.AccessScope{}, false
	}
	return scope, true
}
