package api

import (
	"context"
	"net/http"
	"time"

	"github.com/castorworks/castor/internal/interfaces/api/handler"
	"github.com/castorworks/castor/internal/interfaces/api/middleware"
	"github.com/castorworks/castor/internal/pkg/ratelimit"
	"github.com/gin-gonic/gin"
	rds "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// AdminHandlers 管理端处理器组
type AdminHandlers struct {
	Menu          *handler.MenuHandler
	Asset         *handler.AdminAssetHandler
	AuditLog      *handler.AdminAuditLogHandler
	Authorization *handler.AdminAuthorizationHandler
	Dashboard     *handler.AdminDashboardHandler
	Department    *handler.AdminDepartmentHandler
	Dictionary    *handler.AdminDictionaryHandler
	Job           *handler.AdminJobHandler
	SSO           *handler.AdminSSOHandler
	LoginHistory  *handler.AdminLoginHistoryHandler
	Notification  *handler.AdminNotificationHandler
	Session       *handler.AdminSessionHandler
	Setting       *handler.AdminSettingHandler
	User          *handler.AdminUserHandler
}

// UserHandlers 用户端处理器组
type UserHandlers struct {
	Menu               *handler.MenuHandler
	Account            *handler.AccountHandler
	AccountMFA         *handler.AccountMFAHandler
	SSO                *handler.SSOHandler
	NotificationStream *handler.NotificationStreamHandler
	AccountPermission  *handler.AccountPermissionHandler
	Asset              *handler.AssetHandler
	Authentication     *handler.AuthenticationHandler
	Dictionary         *handler.DictionaryHandler
	LoginHistory       *handler.LoginHistoryHandler
	Notification       *handler.NotificationHandler
	Setting            *handler.SettingHandler
}

// Router API 路由
type Router struct {
	// handler groups
	admin *AdminHandlers
	user  *UserHandlers
	// middlewares
	jwtMiddleware       *middleware.JwtMiddleware
	adminRoleMiddleware *middleware.AdminRoleMiddleware
	// components
	rateLimiter *ratelimit.RateLimiter
	db          *gorm.DB
	redis       rds.UniversalClient
}

// NewRouter 创建路由
func NewRouter(
	admin *AdminHandlers,
	user *UserHandlers,

	jwtMiddleware *middleware.JwtMiddleware,
	adminRoleMiddleware *middleware.AdminRoleMiddleware,

	rateLimiter *ratelimit.RateLimiter,
	db *gorm.DB,
	redis rds.UniversalClient,
) *Router {
	return &Router{
		admin: admin,
		user:  user,

		jwtMiddleware:       jwtMiddleware,
		adminRoleMiddleware: adminRoleMiddleware,

		rateLimiter: rateLimiter,
		db:          db,
		redis:       redis,
	}
}

// With 注册路由到 Gin 引擎
func (r *Router) With(engine *gin.Engine) {
	// 健康检查接口
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	engine.GET("/ready", r.readinessCheck)

	public := engine.Group("/api/v1")
	private := engine.Group("/api/v1")

	// auth - 公开接口
	publicAuth := public.Group("/auth")
	{
		publicAuth.GET("/captcha",
			middleware.RemoteIpPathRateLimit(r.rateLimiter, 1*time.Minute, 30),
			r.user.Authentication.GetCaptcha,
		)
		publicAuth.GET("/public-key",
			middleware.RemoteIpPathRateLimit(r.rateLimiter, 1*time.Minute, 30),
			r.user.Authentication.GetRsaPublicKey,
		)
		publicAuth.POST("/login",
			middleware.RemoteIpPathRateLimit(r.rateLimiter, 5*time.Minute, 15),
			r.jwtMiddleware.AuthMiddleware.LoginHandler,
		)
		// 两步验证第二步：挑战令牌本身限定了用户与尝试次数，这里再按 IP 限流
		publicAuth.POST("/login/totp",
			middleware.RemoteIpPathRateLimit(r.rateLimiter, 5*time.Minute, 30),
			r.jwtMiddleware.LoginWithTOTP,
		)
		// OIDC 单点登录：浏览器导航（302），不是 XHR
		publicAuth.GET("/oidc/providers",
			middleware.RemoteIpPathRateLimit(r.rateLimiter, 1*time.Minute, 60),
			r.user.SSO.Providers,
		)
		publicAuth.GET("/oidc/:provider/authorize",
			middleware.RemoteIpPathRateLimit(r.rateLimiter, 1*time.Minute, 30),
			r.user.SSO.Authorize,
		)
		publicAuth.GET("/oidc/:provider/callback",
			middleware.RemoteIpPathRateLimit(r.rateLimiter, 1*time.Minute, 30),
			r.user.SSO.Callback,
		)
		publicAuth.POST("/refresh-token",
			middleware.RemoteIpPathRateLimit(r.rateLimiter, 5*time.Minute, 30),
			middleware.CSRFMiddleware(),
			r.jwtMiddleware.RefreshWithBlacklistCheck,
		)
		publicAuth.POST("/code",
			middleware.RemoteIpPathRateLimit(r.rateLimiter, 1*time.Minute, 5),
			r.user.Authentication.PostCode,
		)
		// 凭证过期的用户在此改密；服务层另与密码登录共用按用户名的失败计数
		publicAuth.PUT("/password/expired",
			middleware.RemoteIpPathRateLimit(r.rateLimiter, 5*time.Minute, 15),
			r.user.Authentication.PutExpiredPassword,
		)
		publicAuth.POST("/register",
			middleware.RemoteIpPathRateLimit(r.rateLimiter, 1*time.Hour, 10),
			r.user.Authentication.Register,
		)
	}

	// auth - 需要认证的接口
	privateAuth := private.Group("/auth")
	privateAuth.Use(r.jwtMiddleware.AuthMiddleware.MiddlewareFunc())
	privateAuth.Use(middleware.ActorContext())
	privateAuth.Use(middleware.CSRFMiddleware())
	{
		privateAuth.POST("/logout", r.jwtMiddleware.LogoutHandler)
	}

	// account
	privateAccount := private.Group("/account")
	privateAccount.Use(r.jwtMiddleware.AuthMiddleware.MiddlewareFunc())
	privateAccount.Use(middleware.ActorContext())
	privateAccount.Use(middleware.CSRFMiddleware())
	{
		privateAccount.GET("/info", r.user.Account.GetUserInfo)
		privateAccount.PUT("/password", r.user.Account.PutUserPassword)
		privateAccount.PUT("/name", r.user.Account.PutUserName)
		privateAccount.PUT("/notification-preferences", r.user.Account.PutNotificationPreferences)
		privateAccount.POST("/avatar", r.user.Account.PostUserAvatar)

		// 联系方式绑定（服务层另按调用者与目标联系方式限流）
		privateAccount.POST("/contact/code",
			middleware.RemoteIpPathRateLimit(r.rateLimiter, 1*time.Minute, 10),
			r.user.Account.PostContactCode,
		)
		privateAccount.PUT("/contact",
			middleware.RemoteIpPathRateLimit(r.rateLimiter, 1*time.Minute, 10),
			r.user.Account.PutContact,
		)

		privateAccount.GET("/login-histories", r.user.LoginHistory.Gets) // 用户查询自己的登录记录

		// 用户角色/权限（当前用户查询自身）
		privateAccount.GET("/roles", r.user.AccountPermission.GetMyRoles)
		privateAccount.GET("/permissions", r.user.AccountPermission.GetMyPermissions)
		privateAccount.GET("/navigation", r.user.Menu.Navigation)
		// 已登录界面渲染枚举标签用的全量启用字典；免认证的 /dictionaries 只返回公开类型
		privateAccount.GET("/dictionaries", r.user.Dictionary.GetEnabledDicts)
		privateAccount.PUT("/session/roles", r.user.AccountPermission.PutActiveRoles)

		// 两步验证（TOTP）
		privateAccount.GET("/totp", r.user.AccountMFA.Status)
		privateAccount.POST("/totp/setup", r.user.AccountMFA.BeginSetup)
		privateAccount.POST("/totp/enable", r.user.AccountMFA.Enable)
		privateAccount.POST("/totp/disable", r.user.AccountMFA.Disable)
		privateAccount.POST("/totp/recovery-codes", r.user.AccountMFA.RegenerateRecoveryCodes)

		// 外部身份（OIDC）关联
		privateAccount.GET("/identities", r.user.SSO.Identities)
		privateAccount.POST("/identities/:provider/link", r.user.SSO.Link)
		privateAccount.DELETE("/identities/:provider", r.user.SSO.Unlink)

		// 用户通知
		privateAccount.GET("/notifications", r.user.Notification.Gets)
		privateAccount.GET("/notifications/unread-count", r.user.Notification.GetUnreadCount)
		privateAccount.GET("/notifications/stream", r.user.NotificationStream.Stream)
		privateAccount.PUT("/notifications/:id/read", r.user.Notification.MarkAsRead)
		privateAccount.PUT("/notifications/batch-read", r.user.Notification.BatchMarkAsRead)
		privateAccount.PUT("/notifications/read-all", r.user.Notification.MarkAllAsRead)
		privateAccount.DELETE("/notifications/:id", r.user.Notification.Delete)
		privateAccount.GET("/notifications/:id/attachments/:objectKey", r.user.Notification.DownloadAttachment)
	}
	publicAccount := public.Group("/account")
	{
		publicAccount.PUT("/password/reset",
			middleware.RemoteIpPathRateLimit(r.rateLimiter, 1*time.Hour, 5),
			r.user.Account.PutUserPasswordByReset,
		)
	}

	// asset - 公开下载（仅 IsPublic + ACTIVE + 非高风险类型；头像等业务公开文件也走这里）。
	// 限流按 IP+路径计数，即每个文件各自 120 次/分钟。
	public.GET("/assets/download/:objectKey",
		middleware.RemoteIpPathRateLimit(r.rateLimiter, 1*time.Minute, 120),
		r.user.Asset.Download,
	)
	// asset - 公开资产列表
	public.GET("/assets/public",
		middleware.RemoteIpPathRateLimit(r.rateLimiter, 1*time.Minute, 60),
		r.user.Asset.GetPublic,
	)

	// settings - 公开配置
	public.GET("/settings/public",
		middleware.RemoteIpPathRateLimit(r.rateLimiter, 1*time.Minute, 60),
		r.user.Setting.GetPublicSettings,
	)

	// dictionaries - 公开字典（仅 IsPublic 的类型）
	publicDict := public.Group("/dictionaries")
	publicDict.Use(middleware.RemoteIpPathRateLimit(r.rateLimiter, 1*time.Minute, 60))
	{
		publicDict.GET("", r.user.Dictionary.GetAllPublicDicts)
		publicDict.GET("/:typeCode", r.user.Dictionary.GetPublicDictItems)
	}

	// admin
	adminGroup := private.Group("/admin")
	adminGroup.Use(r.jwtMiddleware.AuthMiddleware.MiddlewareFunc())
	adminGroup.Use(middleware.ActorContext())
	adminGroup.Use(middleware.CSRFMiddleware())
	adminGroup.Use(r.adminRoleMiddleware.MiddlewareFunc)
	{
		// 【管理】仪表盘统计
		adminGroup.GET("/dashboard/stats", r.admin.Dashboard.GetStats)

		// 【管理】资产
		adminAsset := adminGroup.Group("/assets")
		{
			adminAsset.GET("/download/:objectKey", r.admin.Asset.Download)
			adminAsset.GET("", r.admin.Asset.Gets)
			adminAsset.GET("/stats", r.admin.Asset.GetStats)
			adminAsset.GET("/:id", r.admin.Asset.Get)
			adminAsset.POST("", r.admin.Asset.Post)
			adminAsset.PUT("/:id", r.admin.Asset.Put)
			adminAsset.PUT("/:id/status", r.admin.Asset.UpdateStatus)
			adminAsset.PUT("/:id/move", r.admin.Asset.Move)
			adminAsset.DELETE("/:id", r.admin.Asset.Delete)
			adminAsset.POST("/batch/delete", r.admin.Asset.BatchDelete)
			adminAsset.PUT("/batch/status", r.admin.Asset.BatchUpdateStatus)
		}

		// 【管理】登录日志
		adminLoginHistory := adminGroup.Group("/login-histories")
		{
			adminLoginHistory.GET("", r.admin.LoginHistory.Gets)
			adminLoginHistory.GET("/export", r.admin.LoginHistory.Export)
			adminLoginHistory.DELETE("/:id", r.admin.LoginHistory.Delete)
			adminLoginHistory.POST("/batch/delete", r.admin.LoginHistory.BatchDelete)
		}

		// 【管理】在线会话
		adminSession := adminGroup.Group("/sessions")
		{
			adminSession.GET("", r.admin.Session.Gets)
			adminSession.DELETE("/:id", r.admin.Session.Revoke)
		}

		// 【管理】定时任务
		adminJobs := adminGroup.Group("/jobs")
		{
			adminJobs.GET("", r.admin.Job.List)
			adminJobs.PUT("/:key", r.admin.Job.Update)
			adminJobs.POST("/:key/run", r.admin.Job.Run)
		}
		adminGroup.GET("/job-runs", r.admin.Job.Runs)

		// 【管理】单点登录身份提供方
		adminOIDC := adminGroup.Group("/oidc-providers")
		{
			adminOIDC.GET("", r.admin.SSO.List)
			adminOIDC.POST("", r.admin.SSO.Create)
			adminOIDC.PUT("/:id", r.admin.SSO.Update)
			adminOIDC.DELETE("/:id", r.admin.SSO.Delete)
		}

		// 【管理】OpenAPI 文档（由路由目录 openapi_catalog.go 生成）
		adminGroup.GET("/openapi.json", r.serveOpenAPI)

		// 【管理】操作审计日志
		adminAuditLog := adminGroup.Group("/audit-logs")
		{
			adminAuditLog.GET("", r.admin.AuditLog.Gets)
			adminAuditLog.GET("/export", r.admin.AuditLog.Export)
			adminAuditLog.POST("/cleanup", r.admin.AuditLog.DeleteBefore)
		}

		// 【管理】系统配置
		adminSetting := adminGroup.Group("/settings")
		{
			adminSetting.GET("", r.admin.Setting.Gets)
			adminSetting.GET("/:id", r.admin.Setting.Get)
			adminSetting.GET("/key/:key", r.admin.Setting.GetByKey)
			adminSetting.POST("", r.admin.Setting.Post)
			adminSetting.PUT("/:id", r.admin.Setting.Put)
			adminSetting.PUT("/batch", r.admin.Setting.BatchUpdate)
			adminSetting.DELETE("/:id", r.admin.Setting.Delete)
		}

		// 【管理】数据字典
		adminDictType := adminGroup.Group("/dict-types")
		{
			adminDictType.GET("", r.admin.Dictionary.GetTypes)
			adminDictType.GET("/:id", r.admin.Dictionary.GetType)
			adminDictType.POST("", r.admin.Dictionary.PostType)
			adminDictType.PUT("/:id", r.admin.Dictionary.PutType)
			adminDictType.DELETE("/:id", r.admin.Dictionary.DeleteType)
		}

		adminDictItem := adminGroup.Group("/dict-items")
		{
			adminDictItem.GET("", r.admin.Dictionary.GetItems)
			adminDictItem.GET("/type/:typeCode", r.admin.Dictionary.GetItemsByTypeCode)
			adminDictItem.GET("/:id", r.admin.Dictionary.GetItem)
			adminDictItem.POST("", r.admin.Dictionary.PostItem)
			adminDictItem.PUT("/:id", r.admin.Dictionary.PutItem)
			adminDictItem.DELETE("/:id", r.admin.Dictionary.DeleteItem)
		}

		// 【管理】通知管理
		adminNotification := adminGroup.Group("/notifications")
		{
			adminNotification.GET("", r.admin.Notification.Gets)
			adminNotification.GET("/channels", r.admin.Notification.Channels)
			adminNotification.GET("/:id", r.admin.Notification.Get)
			adminNotification.GET("/:id/recipients", r.admin.Notification.GetRecipients)
			adminNotification.GET("/:id/attachments/:objectKey", r.admin.Notification.DownloadAttachment)
			adminNotification.POST("", r.admin.Notification.Post)
			adminNotification.PUT("/:id", r.admin.Notification.Put)
			adminNotification.DELETE("/:id", r.admin.Notification.Delete)
			adminNotification.POST("/batch/delete", r.admin.Notification.BatchDelete)
			adminNotification.POST("/attachments", r.admin.Notification.PostAttachment)
		}

		// 【管理】用户接口
		userGroup := adminGroup.Group("/users")
		{
			userGroup.GET("", r.admin.User.Gets)
			userGroup.GET("/export", r.admin.User.Export)
			userGroup.GET("/import-template", r.admin.User.ImportTemplate)
			userGroup.POST("/import", r.admin.User.Import)
			userGroup.GET("/:id", r.admin.User.Get)
			userGroup.POST("", r.admin.User.Post)
			userGroup.PUT("/:id", r.admin.User.Put)
			userGroup.DELETE("/:id", r.admin.User.Delete)
			userGroup.DELETE("/:id/totp", r.admin.User.ResetTOTP)
		}

		// 【管理】资源管理
		menusGroup := adminGroup.Group("/menus")
		{
			menusGroup.GET("", r.admin.Menu.List)
			menusGroup.POST("", r.admin.Menu.Create)
			menusGroup.PUT("/:id", r.admin.Menu.Update)
			menusGroup.DELETE("/:id", r.admin.Menu.Delete)
		}
		resourcesGroup := adminGroup.Group("/resources")
		{
			resourcesGroup.GET("", r.admin.Authorization.GetResources)
			resourcesGroup.GET("/modules", r.admin.Authorization.GetResourceModules)
			resourcesGroup.GET("/:id", r.admin.Authorization.GetResource)
			resourcesGroup.POST("", r.admin.Authorization.CreateResource)
			resourcesGroup.PUT("/:id", r.admin.Authorization.UpdateResource)
			resourcesGroup.DELETE("/:id", r.admin.Authorization.DeleteResource)
		}

		// 【管理】角色管理
		rolesGroup := adminGroup.Group("/roles")
		{
			rolesGroup.GET("", r.admin.Authorization.GetRoles)
			rolesGroup.GET("/:role", r.admin.Authorization.GetRole)
			rolesGroup.POST("", r.admin.Authorization.CreateRole)
			rolesGroup.PUT("/:role", r.admin.Authorization.UpdateRole)
			rolesGroup.DELETE("/:role", r.admin.Authorization.DeleteRole)
			rolesGroup.GET("/:role/permissions", r.admin.Authorization.GetRolePermissions)
			rolesGroup.PUT("/:role/permissions", r.admin.Authorization.SetRolePermissions)
			rolesGroup.GET("/:role/users", r.admin.Authorization.GetRoleUsers)
			rolesGroup.GET("/:role/hierarchy", r.admin.Authorization.GetRoleHierarchy)
			rolesGroup.PUT("/:role/hierarchy", r.admin.Authorization.SetRoleHierarchy)
			rolesGroup.PUT("/:role/data-scope", r.admin.Authorization.SetRoleDataScope)
		}

		// 【管理】部门
		departmentsGroup := adminGroup.Group("/departments")
		{
			departmentsGroup.GET("", r.admin.Department.List)
			departmentsGroup.POST("", r.admin.Department.Create)
			departmentsGroup.PUT("/:id", r.admin.Department.Update)
			departmentsGroup.DELETE("/:id", r.admin.Department.Delete)
		}

		constraintsGroup := adminGroup.Group("/authorization/constraints")
		{
			constraintsGroup.GET("", r.admin.Authorization.GetConstraints)
			constraintsGroup.POST("", r.admin.Authorization.CreateConstraint)
			constraintsGroup.PUT("/:id", r.admin.Authorization.UpdateConstraint)
			constraintsGroup.DELETE("/:id", r.admin.Authorization.DeleteConstraint)
		}

		// 【管理】授权 - 用户角色管理
		userRolesGroup := adminGroup.Group("/authorization/users/:username")
		{
			userRolesGroup.GET("/roles", r.admin.Authorization.GetUserRoles)
			userRolesGroup.POST("/roles", r.admin.Authorization.AddUserRole)
			userRolesGroup.DELETE("/roles/:role", r.admin.Authorization.DeleteUserRole)
			userRolesGroup.GET("/permissions", r.admin.Authorization.GetUserPermissions)
		}

		// 【管理】授权 - 元数据查询
		metadataGroup := adminGroup.Group("/authorization/metadata")
		{
			metadataGroup.GET("/subjects", r.admin.Authorization.GetSubjects)
			metadataGroup.GET("/objects", r.admin.Authorization.GetObjects)
			metadataGroup.GET("/actions", r.admin.Authorization.GetActions)
			metadataGroup.GET("/policies", r.admin.Authorization.GetPolicies)
		}
	}
}

// readinessCheck 就绪探针，检查数据库和 Redis 连通性
func (r *Router) readinessCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	ready := true

	// 检查数据库连通性
	if sqlDB, err := r.db.WithContext(ctx).DB(); err != nil {
		ready = false
	} else if err := sqlDB.PingContext(ctx); err != nil {
		ready = false
	}

	// 检查 Redis 连通性
	if err := r.redis.Ping(ctx).Err(); err != nil {
		ready = false
	}

	if ready {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
	}
}
