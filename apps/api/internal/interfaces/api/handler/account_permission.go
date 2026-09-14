package handler

import (
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

type AccountPermissionHandler struct {
	rbac service.RBACService
}

func NewAccountPermissionHandler(rbac service.RBACService) *AccountPermissionHandler {
	return &AccountPermissionHandler{rbac: rbac}
}

func (h *AccountPermissionHandler) GetMyRoles(c *gin.Context) {
	h.getAccess(c)
}

func (h *AccountPermissionHandler) GetMyPermissions(c *gin.Context) {
	h.getAccess(c)
}

func (h *AccountPermissionHandler) PutActiveRoles(c *gin.Context) {
	var request struct {
		RoleCodes []string `json:"roleCodes"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequestErr(c, err)
		return
	}
	access, err := h.rbac.SetActiveRoles(c.Request.Context(), ucontext.GetAuthorizationSessionID(c), ucontext.GetUserID(c), request.RoleCodes)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, access)
}

func (h *AccountPermissionHandler) getAccess(c *gin.Context) {
	access, err := h.rbac.GetSessionAccess(c.Request.Context(), ucontext.GetAuthorizationSessionID(c), ucontext.GetUserID(c))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, access)
}
