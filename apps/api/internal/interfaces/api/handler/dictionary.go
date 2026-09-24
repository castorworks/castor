package handler

import (
	"fmt"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/gin-gonic/gin"
)

// AdminDictionaryHandler 管理员字典处理器
type AdminDictionaryHandler struct {
	DictionaryService service.DictionaryService
	AuditLogService   service.AuditLogService
}

// NewAdminDictionaryHandler 创建管理员字典处理器
func NewAdminDictionaryHandler(svc service.DictionaryService, auditLogService service.AuditLogService) *AdminDictionaryHandler {
	return &AdminDictionaryHandler{
		DictionaryService: svc,
		AuditLogService:   auditLogService,
	}
}

// ========== 字典类型 ==========

// GetTypes 获取字典类型列表
func (h *AdminDictionaryHandler) GetTypes(c *gin.Context) {
	items, err := h.DictionaryService.GetTypes(c)
	if err != nil {
		response.InternalServerErrorErr(c, err)
		return
	}
	response.Success(c, items)
}

// GetType 获取单个字典类型
func (h *AdminDictionaryHandler) GetType(c *gin.Context) {
	GenericGet(c, func(ctx *gin.Context, id uint) (interface{}, error) {
		return h.DictionaryService.GetType(ctx, id)
	})
}

// PostType 创建字典类型
func (h *AdminDictionaryHandler) PostType(c *gin.Context) {
	GenericPost(c, &dto.DictTypePostReq{}, func(ctx *gin.Context, req interface{}) (interface{}, error) {
		r := req.(*dto.DictTypePostReq)
		result, err := h.DictionaryService.PostType(ctx, r)
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeDictTypeCreate, r.Code,
			fmt.Sprintf("Create dict type: %s", r.Code), err)
		return result, err
	}, func(req interface{}) bool {
		r := req.(*dto.DictTypePostReq)
		return r.Code != "" && r.Name.ToI18nText().IsComplete()
	})
}

// PutType 更新字典类型
func (h *AdminDictionaryHandler) PutType(c *gin.Context) {
	GenericPut(c, &dto.DictTypePutReq{}, func(ctx *gin.Context, id uint, req interface{}) (interface{}, error) {
		code := h.dictTypeCode(c, id)
		result, err := h.DictionaryService.PutType(ctx, id, req.(*dto.DictTypePutReq))
		// 只记录被改动的字典类型编码，不记录名称/描述等请求内容。
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeDictTypeUpdate, code,
			fmt.Sprintf("Update dict type: %s", code), err)
		return result, err
	}, nil)
}

// DeleteType 删除字典类型
func (h *AdminDictionaryHandler) DeleteType(c *gin.Context) {
	GenericDelete(c, func(ctx *gin.Context, id uint) error {
		code := h.dictTypeCode(c, id)
		err := h.DictionaryService.DeleteType(ctx, id)
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeDictTypeDelete, code,
			fmt.Sprintf("Delete dict type: %s", code), err)
		return err
	})
}

// ========== 字典项 ==========

// GetItems 获取字典项列表
func (h *AdminDictionaryHandler) GetItems(c *gin.Context) {
	typeCode := c.Query("typeCode")
	items, err := h.DictionaryService.GetItems(c, typeCode)
	if err != nil {
		response.InternalServerErrorErr(c, err)
		return
	}
	response.Success(c, items)
}

// GetItemsByTypeCode 根据类型编码获取字典项
func (h *AdminDictionaryHandler) GetItemsByTypeCode(c *gin.Context) {
	typeCode := c.Param("typeCode")
	if typeCode == "" {
		response.BadRequest(c)
		return
	}

	items, err := h.DictionaryService.GetItemsByTypeCode(c, typeCode)
	if err != nil {
		response.InternalServerErrorErr(c, err)
		return
	}
	response.Success(c, items)
}

// GetItem 获取单个字典项
func (h *AdminDictionaryHandler) GetItem(c *gin.Context) {
	GenericGet(c, func(ctx *gin.Context, id uint) (interface{}, error) {
		return h.DictionaryService.GetItem(ctx, id)
	})
}

// PostItem 创建字典项
func (h *AdminDictionaryHandler) PostItem(c *gin.Context) {
	GenericPost(c, &dto.DictItemPostReq{}, func(ctx *gin.Context, req interface{}) (interface{}, error) {
		r := req.(*dto.DictItemPostReq)
		result, err := h.DictionaryService.PostItem(ctx, r)
		// 字典项的 value 由客户端提供，可能是任意载荷，审计只记录所属类型。
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeDictItemCreate, r.TypeCode,
			fmt.Sprintf("Create dict item under type: %s", r.TypeCode), err)
		return result, err
	}, func(req interface{}) bool {
		r := req.(*dto.DictItemPostReq)
		return r.TypeCode != "" && r.Label.ToI18nText().IsComplete() && r.Value != ""
	})
}

// PutItem 更新字典项
func (h *AdminDictionaryHandler) PutItem(c *gin.Context) {
	GenericPut(c, &dto.DictItemPutReq{}, func(ctx *gin.Context, id uint, req interface{}) (interface{}, error) {
		result, err := h.DictionaryService.PutItem(ctx, id, req.(*dto.DictItemPutReq))
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeDictItemUpdate, fmt.Sprintf("%d", id),
			fmt.Sprintf("Update dict item ID: %d", id), err)
		return result, err
	}, nil)
}

// DeleteItem 删除字典项
func (h *AdminDictionaryHandler) DeleteItem(c *gin.Context) {
	GenericDelete(c, func(ctx *gin.Context, id uint) error {
		err := h.DictionaryService.DeleteItem(ctx, id)
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeDictItemDelete, fmt.Sprintf("%d", id),
			fmt.Sprintf("Delete dict item ID: %d", id), err)
		return err
	})
}

// dictTypeCode 解析字典类型编码用于审计；查不到时退回 ID 形式，不阻断主流程。
func (h *AdminDictionaryHandler) dictTypeCode(c *gin.Context, id uint) string {
	item, err := h.DictionaryService.GetType(c, id)
	if err != nil || item == nil || item.Code == "" {
		return fmt.Sprintf("#%d", id)
	}
	return item.Code
}

// ========== 读取接口 ==========

// DictionaryHandler 字典读取处理器：已登录界面用的全量启用字典，与免认证的公开字典
type DictionaryHandler struct {
	DictionaryService service.DictionaryService
}

// NewDictionaryHandler 创建字典读取处理器
func NewDictionaryHandler(svc service.DictionaryService) *DictionaryHandler {
	return &DictionaryHandler{DictionaryService: svc}
}

// GetPublicDictItems 获取公开字典项
func (h *DictionaryHandler) GetPublicDictItems(c *gin.Context) {
	typeCode := c.Param("typeCode")
	if typeCode == "" {
		response.BadRequest(c)
		return
	}

	items, err := h.DictionaryService.GetPublicDictItems(c, typeCode)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// GetEnabledDicts 获取所有启用的字典（需登录），界面据此渲染枚举值的标签、颜色与图标
func (h *DictionaryHandler) GetEnabledDicts(c *gin.Context) {
	dicts, err := h.DictionaryService.GetEnabledDicts(c)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, dicts)
}

// GetAllPublicDicts 获取所有公开字典
func (h *DictionaryHandler) GetAllPublicDicts(c *gin.Context) {
	dicts, err := h.DictionaryService.GetAllPublicDicts(c)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, dicts)
}
