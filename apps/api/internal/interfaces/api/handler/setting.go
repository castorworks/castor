package handler

import (
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/gin-gonic/gin"
)

// AdminSettingHandler 管理员系统配置处理器
type AdminSettingHandler struct {
	SettingService service.SettingService
}

// NewAdminSettingHandler 创建管理员系统配置处理器
func NewAdminSettingHandler(svc service.SettingService) *AdminSettingHandler {
	return &AdminSettingHandler{SettingService: svc}
}

// Gets 获取配置列表
func (h *AdminSettingHandler) Gets(c *gin.Context) {
	category := c.Query("category")
	items, err := h.SettingService.Gets(c, category)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// Get 获取单个配置
func (h *AdminSettingHandler) Get(c *gin.Context) {
	GenericGet(c, func(ctx *gin.Context, id uint) (interface{}, error) {
		return h.SettingService.Get(ctx, id)
	})
}

// GetByKey 根据 Key 获取配置
func (h *AdminSettingHandler) GetByKey(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.BadRequestI18n(c, response.ErrFieldRequired)
		return
	}

	item, err := h.SettingService.GetByKey(c, key)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, item)
}

// Post 创建配置
func (h *AdminSettingHandler) Post(c *gin.Context) {
	GenericPost(c, &dto.SettingPostReq{}, func(ctx *gin.Context, req interface{}) (interface{}, error) {
		return h.SettingService.Post(ctx, req.(*dto.SettingPostReq))
	}, func(req interface{}) bool {
		r := req.(*dto.SettingPostReq)
		return r.Key != "" && r.Name != ""
	})
}

// Put 更新配置
func (h *AdminSettingHandler) Put(c *gin.Context) {
	GenericPut(c, &dto.SettingPutReq{}, func(ctx *gin.Context, id uint, req interface{}) (interface{}, error) {
		return h.SettingService.Put(ctx, id, req.(*dto.SettingPutReq))
	}, nil)
}

// Delete 删除配置
func (h *AdminSettingHandler) Delete(c *gin.Context) {
	GenericDelete(c, func(ctx *gin.Context, id uint) error {
		return h.SettingService.Delete(ctx, id)
	})
}

// BatchUpdate 批量更新配置
func (h *AdminSettingHandler) BatchUpdate(c *gin.Context) {
	var req dto.SettingBatchUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if err := h.SettingService.BatchUpdate(c, &req); err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, gin.H{}, response.InfSettingUpdated)
}

// SettingHandler 公开配置处理器（无需认证）
type SettingHandler struct {
	SettingService service.SettingService
}

// NewSettingHandler 创建公开配置处理器
func NewSettingHandler(svc service.SettingService) *SettingHandler {
	return &SettingHandler{SettingService: svc}
}

// GetPublicSettings 获取公开配置
func (h *SettingHandler) GetPublicSettings(c *gin.Context) {
	settings, err := h.SettingService.GetPublicSettings(c)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, settings)
}
