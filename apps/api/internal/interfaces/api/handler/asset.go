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
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/castorworks/castor/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// AssetHandler 资产处理器（公开接口）
type AssetHandler struct {
	AssetService    service.AssetService
	AuditLogService service.AuditLogService
}

// NewAssetHandler 创建资产处理器
func NewAssetHandler(assetService service.AssetService, auditLogService service.AuditLogService) *AssetHandler {
	return &AssetHandler{
		AssetService:    assetService,
		AuditLogService: auditLogService,
	}
}

// Download 下载资产
func (h *AssetHandler) Download(c *gin.Context) {
	objectKey := c.Param("objectKey")
	result, err := h.AssetService.PrepareDownload(c.Request.Context(), objectKey, true)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	streamAsset(c, h.AssetService, result)
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

// streamAsset 写出下载响应头并流式输出资产内容，公开与管理端下载共用。
// 一律以附件形式下发并禁止嗅探，避免上传的内容被浏览器当作页面内联执行。
func streamAsset(c *gin.Context, svc service.AssetService, result *service.AssetDownloadResult) {
	c.Header("Content-Type", result.ContentType)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(result.Filename)))
	if result.Size > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", result.Size))
	}

	if err := svc.WriteContent(c.Request.Context(), result.ObjectKey, c.Writer); err != nil {
		log.Err(c, err).Str("objectKey", result.ObjectKey).Msg("Asset download interrupted")
		if !c.Writer.Written() {
			c.Status(http.StatusNotFound)
		}
	}
}

// ============================================
// AdminAssetHandler 管理员资产处理器
// ============================================

// AdminAssetHandler 管理员资产处理器
type AdminAssetHandler struct {
	AssetService          service.AssetService
	NotificationPublisher service.NotificationPublisher
	AuditLogService       service.AuditLogService
}

// NewAdminAssetHandler 创建管理员资产处理器
func NewAdminAssetHandler(assetService service.AssetService, notificationPublisher service.NotificationPublisher, auditLogService service.AuditLogService) *AdminAssetHandler {
	return &AdminAssetHandler{
		AssetService:          assetService,
		NotificationPublisher: notificationPublisher,
		AuditLogService:       auditLogService,
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

	streamAsset(c, h.AssetService, result)
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
	// 原始文件名是客户端输入，审计记录服务端生成的对象键。
	target, details := "-", "Upload asset"
	if err == nil {
		target, details = resp.ObjectKey, fmt.Sprintf("Upload asset %d", resp.ID)
	}
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypeAssetCreate, target, details, err)
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
	// 名称/描述/标签是自由文本，审计只记录哪条资产被改动。
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypeAssetUpdate, fmt.Sprintf("%d", id),
		fmt.Sprintf("Update asset ID: %d", id), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessI18n(c, resp, response.InfFileUpdated)
}

// Delete 删除资产
func (h *AdminAssetHandler) Delete(c *gin.Context) {
	GenericDelete(c, func(ctx *gin.Context, id uint) error {
		err := h.AssetService.Delete(ctx, id)
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeAssetDelete, fmt.Sprintf("%d", id),
			fmt.Sprintf("Delete asset ID: %d", id), err)
		return err
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
	err = h.AssetService.UpdateStatus(c.Request.Context(), assetID, req.Status)
	// 状态是受 binding 白名单约束的枚举，可以安全写入审计详情。
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypeAssetUpdate, fmt.Sprintf("%d", assetID),
		fmt.Sprintf("Update asset ID %d status to: %s", assetID, req.Status), err)
	if err != nil {
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

	err = h.AssetService.Move(c.Request.Context(), uint(id), &req)
	// 目标路径是客户端自由文本，审计只记录移动行为本身。
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypeAssetUpdate, fmt.Sprintf("%d", id),
		fmt.Sprintf("Move asset ID: %d", id), err)
	if err != nil {
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

	err := h.AssetService.BatchDelete(c.Request.Context(), req.IDs)
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypeAssetDelete, joinAuditIDs(req.IDs),
		fmt.Sprintf("Batch delete %d assets: %s", len(req.IDs), joinAuditIDs(req.IDs)), err)
	if err != nil {
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

	err := h.AssetService.BatchUpdateStatus(c.Request.Context(), req.IDs, req.Status)
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypeAssetUpdate, joinAuditIDs(req.IDs),
		fmt.Sprintf("Batch update %d assets status to %s: %s", len(req.IDs), req.Status, joinAuditIDs(req.IDs)), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessI18n(c, nil, response.InfBatchStatusUpdated)
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
