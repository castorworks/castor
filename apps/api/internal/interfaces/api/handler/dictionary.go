package handler

import (
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/gin-gonic/gin"
)

// AdminDictionaryHandler 管理员字典处理器
type AdminDictionaryHandler struct {
	DictionaryService service.DictionaryService
}

// NewAdminDictionaryHandler 创建管理员字典处理器
func NewAdminDictionaryHandler(svc service.DictionaryService) *AdminDictionaryHandler {
	return &AdminDictionaryHandler{DictionaryService: svc}
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
		return h.DictionaryService.PostType(ctx, req.(*dto.DictTypePostReq))
	}, func(req interface{}) bool {
		r := req.(*dto.DictTypePostReq)
		return r.Code != "" && r.Name != ""
	})
}

// PutType 更新字典类型
func (h *AdminDictionaryHandler) PutType(c *gin.Context) {
	GenericPut(c, &dto.DictTypePutReq{}, func(ctx *gin.Context, id uint, req interface{}) (interface{}, error) {
		return h.DictionaryService.PutType(ctx, id, req.(*dto.DictTypePutReq))
	}, nil)
}

// DeleteType 删除字典类型
func (h *AdminDictionaryHandler) DeleteType(c *gin.Context) {
	GenericDelete(c, func(ctx *gin.Context, id uint) error {
		return h.DictionaryService.DeleteType(ctx, id)
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
		return h.DictionaryService.PostItem(ctx, req.(*dto.DictItemPostReq))
	}, func(req interface{}) bool {
		r := req.(*dto.DictItemPostReq)
		return r.TypeCode != "" && r.Label != "" && r.Value != ""
	})
}

// PutItem 更新字典项
func (h *AdminDictionaryHandler) PutItem(c *gin.Context) {
	GenericPut(c, &dto.DictItemPutReq{}, func(ctx *gin.Context, id uint, req interface{}) (interface{}, error) {
		return h.DictionaryService.PutItem(ctx, id, req.(*dto.DictItemPutReq))
	}, nil)
}

// DeleteItem 删除字典项
func (h *AdminDictionaryHandler) DeleteItem(c *gin.Context) {
	GenericDelete(c, func(ctx *gin.Context, id uint) error {
		return h.DictionaryService.DeleteItem(ctx, id)
	})
}

// ========== 公开接口 ==========

// DictionaryHandler 公开字典处理器
type DictionaryHandler struct {
	DictionaryService service.DictionaryService
}

// NewDictionaryHandler 创建公开字典处理器
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
		response.InternalServerErrorErr(c, err)
		return
	}
	response.Success(c, items)
}

// GetAllPublicDicts 获取所有公开字典
func (h *DictionaryHandler) GetAllPublicDicts(c *gin.Context) {
	dicts, err := h.DictionaryService.GetAllPublicDicts(c)
	if err != nil {
		response.InternalServerErrorErr(c, err)
		return
	}
	response.Success(c, dicts)
}
