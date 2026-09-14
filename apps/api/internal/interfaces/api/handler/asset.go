package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/asset"
	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/castorworks/castor/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// AssetHandler 资产处理器（公开接口）
type AssetHandler struct {
	AssetService service.AssetService
}

// NewAssetHandler 创建资产处理器
func NewAssetHandler(assetService service.AssetService) *AssetHandler {
	return &AssetHandler{AssetService: assetService}
}

// Post 上传资产
func (h *AssetHandler) Post(c *gin.Context) {
	f, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}
	defer f.Close()

	filename := fileHeader.Filename
	if !validator.IsValidFilename(filename) {
		response.BadRequestI18n(c, response.ErrInvalidFilename)
		return
	}

	var req dto.AssetUploadReq
	if err := c.ShouldBind(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	size := fileHeader.Size

	resp, err := h.AssetService.Upload(c.Request.Context(), &req, filename, f, contentType, size)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	if resp.IsDuplicate {
		response.SuccessI18n(c, resp, response.InfFileDuplicate)
		return
	}
	response.SuccessI18n(c, resp, response.InfFileUploaded)
}

// Download 下载资产
func (h *AssetHandler) Download(c *gin.Context) {
	objectKey := c.Param("objectKey")
	result, err := h.AssetService.PrepareDownload(c.Request.Context(), objectKey, true)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	h.streamDownload(c, result)
}

// GetPublic 获取公开资产列表
func (h *AssetHandler) GetPublic(c *gin.Context) {
	page, pageSize := ParsePageParams(c)
	order, ok := parseOrderParam(c, reflect.TypeOf(asset.Asset{}), "created_at DESC")
	if !ok {
		return
	}

	items, total, err := h.AssetService.GetPublicAssets(c.Request.Context(), page, pageSize, order)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessListPaged(c, total, items, page, pageSize)
}

// streamDownload 流式下载资产
func (h *AssetHandler) streamDownload(c *gin.Context, result *service.AssetDownloadResult) {
	c.Header("Content-Type", result.ContentType)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(result.Filename)))
	if result.Size > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", result.Size))
	}

	s3Client := h.AssetService.GetS3Client()
	s3Client.GetObject(c, result.Bucket, result.ObjectKey, c.Writer)
}

// ============================================
// AdminAssetHandler 管理员资产处理器
// ============================================

// AdminAssetHandler 管理员资产处理器
type AdminAssetHandler struct {
	AssetService          service.AssetService
	NotificationPublisher service.NotificationPublisher
}

// NewAdminAssetHandler 创建管理员资产处理器
func NewAdminAssetHandler(assetService service.AssetService, notificationPublisher service.NotificationPublisher) *AdminAssetHandler {
	return &AdminAssetHandler{
		AssetService:          assetService,
		NotificationPublisher: notificationPublisher,
	}
}

// Gets 获取资产列表
func (h *AdminAssetHandler) Gets(c *gin.Context) {
	GenericGets(c, reflect.TypeOf(asset.Asset{}), func(ctx *gin.Context, page, size int, order string, opts ...query.Option) (interface{}, int64, error) {
		return h.AssetService.Gets(ctx, page, size, order, opts...)
	})
}

// Get 获取资产详情
func (h *AdminAssetHandler) Get(c *gin.Context) {
	GenericGet(c, func(ctx *gin.Context, id uint) (interface{}, error) {
		return h.AssetService.Get(ctx, id)
	})
}

// Download 下载资产（管理员）
func (h *AdminAssetHandler) Download(c *gin.Context) {
	objectKey := c.Param("objectKey")
	result, err := h.AssetService.PrepareDownload(c.Request.Context(), objectKey, false)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.Header("Content-Type", result.ContentType)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(result.Filename)))
	if result.Size > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", result.Size))
	}

	s3Client := h.AssetService.GetS3Client()
	s3Client.GetObject(c, result.Bucket, result.ObjectKey, c.Writer)
}

// Post 上传资产
func (h *AdminAssetHandler) Post(c *gin.Context) {
	f, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}
	defer f.Close()

	filename := fileHeader.Filename
	if !validator.IsValidFilename(filename) {
		response.BadRequestI18n(c, response.ErrInvalidFilename)
		return
	}

	var req dto.AssetUploadReq
	if err := c.ShouldBind(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	size := fileHeader.Size

	resp, err := h.AssetService.Upload(c.Request.Context(), &req, filename, f, contentType, size)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	if resp.IsDuplicate {
		response.SuccessI18n(c, resp, response.InfFileDuplicate)
		return
	}
	response.SuccessI18n(c, resp, response.InfFileUploaded)
}

// Put 更新资产信息
func (h *AdminAssetHandler) Put(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequestI18n(c, response.ErrInvalidID)
		return
	}

	var req dto.AssetUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	resp, err := h.AssetService.Update(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessI18n(c, resp, response.InfFileUpdated)
}

// Delete 删除资产
func (h *AdminAssetHandler) Delete(c *gin.Context) {
	GenericDelete(c, func(ctx *gin.Context, id uint) error {
		return h.AssetService.Delete(ctx, id)
	})
}

// UpdateStatus 更新资产状态
func (h *AdminAssetHandler) UpdateStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequestI18n(c, response.ErrInvalidID)
		return
	}

	var req dto.AssetStatusUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	assetID := uint(id)
	if err := h.AssetService.UpdateStatus(c.Request.Context(), assetID, req.Status); err != nil {
		response.HandleError(c, err)
		return
	}

	h.publishAssetStatusNotification(c, assetID, req.Status)

	response.SuccessI18n(c, nil, response.InfStatusUpdated)
}

// Move 移动资产到指定文件夹
func (h *AdminAssetHandler) Move(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequestI18n(c, response.ErrInvalidID)
		return
	}

	var req dto.AssetMoveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if err := h.AssetService.Move(c.Request.Context(), uint(id), &req); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessI18n(c, nil, response.InfMoveSuccess)
}

// BatchDelete 批量删除资产
func (h *AdminAssetHandler) BatchDelete(c *gin.Context) {
	var req dto.AssetBatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if err := h.AssetService.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessI18n(c, nil, response.InfBatchDeleteSuccess)
}

// BatchUpdateStatus 批量更新资产状态
func (h *AdminAssetHandler) BatchUpdateStatus(c *gin.Context) {
	var req dto.AssetBatchStatusUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if err := h.AssetService.BatchUpdateStatus(c.Request.Context(), req.IDs, req.Status); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessI18n(c, nil, response.InfBatchStatusUpdated)
}

// GetsByCategory 按分类获取资产列表
func (h *AdminAssetHandler) GetsByCategory(c *gin.Context) {
	category := asset.AssetCategory(c.Param("category"))
	page, pageSize := ParsePageParams(c)
	order, ok := parseOrderParam(c, reflect.TypeOf(asset.Asset{}), "created_at DESC")
	if !ok {
		return
	}

	items, total, err := h.AssetService.GetsByCategory(c.Request.Context(), category, page, pageSize, order)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessListPaged(c, total, items, page, pageSize)
}

// GetsByStatus 按状态获取资产列表
func (h *AdminAssetHandler) GetsByStatus(c *gin.Context) {
	status := asset.AssetStatus(c.Param("status"))
	page, pageSize := ParsePageParams(c)
	order, ok := parseOrderParam(c, reflect.TypeOf(asset.Asset{}), "created_at DESC")
	if !ok {
		return
	}

	items, total, err := h.AssetService.GetsByStatus(c.Request.Context(), status, page, pageSize, order)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessListPaged(c, total, items, page, pageSize)
}

// GetsByFolderPath 按文件夹路径获取资产列表
func (h *AdminAssetHandler) GetsByFolderPath(c *gin.Context) {
	folderPath := c.Query("folderPath")
	page, pageSize := ParsePageParams(c)
	order, ok := parseOrderParam(c, reflect.TypeOf(asset.Asset{}), "created_at DESC")
	if !ok {
		return
	}

	items, total, err := h.AssetService.GetsByFolderPath(c.Request.Context(), folderPath, page, pageSize, order)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessListPaged(c, total, items, page, pageSize)
}

// GetStats 获取资产统计信息
func (h *AdminAssetHandler) GetStats(c *gin.Context) {
	stats, err := h.AssetService.GetStats(c.Request.Context())
	if err != nil {
		response.InternalServerErrorErr(c, err)
		return
	}

	response.Success(c, stats)
}

func (h *AdminAssetHandler) publishAssetStatusNotification(c *gin.Context, assetID uint, status asset.AssetStatus) {
	if h.NotificationPublisher == nil {
		return
	}
	assetResp, err := h.AssetService.Get(c.Request.Context(), assetID)
	if err != nil || assetResp == nil || assetResp.CreatedBy == 0 {
		return
	}
	if err := h.NotificationPublisher.PublishToUsers(c.Request.Context(), service.NotificationPublishRequest{
		TemplateKey:  service.NotificationTemplateAssetStatusUpdated,
		TemplateData: map[string]any{"status": status},
		Type:         string(notification.TypeSystem),
		Level:        string(notification.LevelInfo),
		Link:         "/dashboard/assets",
		UserIDs:      []uint{assetResp.CreatedBy},
	}); err != nil {
		log.Err(c, err).Msg("Failed to publish notification")
	}
}
