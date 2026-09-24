package handler

import (
	"fmt"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/gin-gonic/gin"
)

// AdminSettingHandler 管理员系统配置处理器
type AdminSettingHandler struct {
	SettingService  service.SettingService
	AuditLogService service.AuditLogService
}

// NewAdminSettingHandler 创建管理员系统配置处理器
func NewAdminSettingHandler(svc service.SettingService, auditLogService service.AuditLogService) *AdminSettingHandler {
	return &AdminSettingHandler{
		SettingService:  svc,
		AuditLogService: auditLogService,
	}
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
		r := req.(*dto.SettingPostReq)
		result, err := h.SettingService.Post(ctx, r)
		// 配置值可能是凭据，审计只记录配置键。
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeSettingCreate, r.Key,
			fmt.Sprintf("Create setting: %s", r.Key), err)
		return result, err
	}, func(req interface{}) bool {
		r := req.(*dto.SettingPostReq)
		return r.Key != "" && r.Name != ""
	})
}

// Put 更新配置
func (h *AdminSettingHandler) Put(c *gin.Context) {
	GenericPut(c, &dto.SettingPutReq{}, func(ctx *gin.Context, id uint, req interface{}) (interface{}, error) {
		key := h.settingKey(ctx, id)
		result, err := h.SettingService.Put(ctx, id, req.(*dto.SettingPutReq))
		// 只记录“哪个配置被改了”，绝不记录新旧配置值。
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeSettingUpdate, key,
			fmt.Sprintf("Update setting: %s", key), err)
		return result, err
	}, nil)
}

// Delete 删除配置
func (h *AdminSettingHandler) Delete(c *gin.Context) {
	GenericDelete(c, func(ctx *gin.Context, id uint) error {
		key := h.settingKey(ctx, id)
		err := h.SettingService.Delete(ctx, id)
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeSettingDelete, key,
			fmt.Sprintf("Delete setting: %s", key), err)
		return err
	})
}

// BatchUpdate 批量更新配置
func (h *AdminSettingHandler) BatchUpdate(c *gin.Context) {
	var req dto.SettingBatchUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	keys := make([]string, 0, len(req.Settings))
	for _, item := range req.Settings {
		keys = append(keys, item.Key)
	}
	joined := joinAuditKeys(keys)

	err := h.SettingService.BatchUpdate(c, &req)
	// 登录方式、验证码开关、密码策略等都走这个接口，必须留痕；同样只记录键。
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypeSettingUpdate, joined,
		fmt.Sprintf("Batch update %d settings: %s", len(req.Settings), joined), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, gin.H{}, response.InfSettingUpdated)
}

// settingKey 解析配置键用于审计；查不到时退回 ID 形式，绝不阻断主流程。
func (h *AdminSettingHandler) settingKey(c *gin.Context, id uint) string {
	item, err := h.SettingService.Get(c, id)
	if err != nil || item == nil || item.Key == "" {
		return fmt.Sprintf("#%d", id)
	}
	return item.Key
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
