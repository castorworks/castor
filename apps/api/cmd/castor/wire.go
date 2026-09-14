//go:build wireinject

package main

import (
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/infrastructure/cache"
	"github.com/castorworks/castor/internal/infrastructure/database"
	"github.com/castorworks/castor/internal/infrastructure/external"
	infraI18n "github.com/castorworks/castor/internal/infrastructure/i18n"
	"github.com/castorworks/castor/internal/infrastructure/persistence"
	"github.com/castorworks/castor/internal/infrastructure/storage"
	"github.com/castorworks/castor/internal/interfaces/api"
	"github.com/castorworks/castor/internal/interfaces/api/handler"
	"github.com/castorworks/castor/internal/interfaces/api/middleware"
	"github.com/castorworks/castor/internal/interfaces/api/server"
	"github.com/castorworks/castor/internal/pkg/ratelimit"
	"github.com/google/wire"
)

func InitServer() (*server.Server, error) {
	wire.Build(
		// gin engine
		server.NewGinEngine,

		// application policies (from config)
		provideRuntimePolicy,
		provideAssetPolicy,
		provideRsaKeyConfig,

		// infrastructure - database & cache
		database.NewPostgres,
		cache.NewRedis,
		storage.NewS3,

		// infrastructure - external services
		external.NewAliyunSmsClient,
		external.NewMailClient,
		external.NewCaptchaClient,
		external.NewVerificationCodeStore,
		wire.Bind(new(service.VerificationCodeStore), new(*external.VerificationCodeStore)),
		infraI18n.NewNotificationTemplateRenderer,

		// infrastructure - persistence (repositories)
		persistence.NewUserRepository,
		persistence.NewAssetRepository,
		persistence.NewLoginHistoryRepository,
		persistence.NewAuditLogRepository,
		// permission repositories
		persistence.NewResourceRepository,
		persistence.NewMenuRepository,
		persistence.NewRoleRepository,
		persistence.NewAuthorizationRepository,
		// setting repository
		persistence.NewSettingRepository,
		// dictionary repositories
		persistence.NewDictTypeRepository,
		persistence.NewDictItemRepository,
		// notification repositories
		persistence.NewNotificationRepository,
		persistence.NewUserNotificationRepository,

		// pkg - utilities
		ratelimit.NewRateLimiter,

		// middleware
		middleware.NewJwtMiddleware,
		middleware.NewAdminRoleMiddleware,

		// application - services
		service.NewSettingHelper,
		service.NewUserService,
		service.NewAccountService,
		service.NewAuthService,
		service.NewRsaService,
		service.NewTokenBlacklistService,
		service.NewLoginService,
		service.NewAssetService,
		service.NewUserAvatarService,
		service.NewLoginHistoryService,
		service.NewAuditLogService,
		service.NewPermissionService,
		service.NewMenuService,
		service.NewRBACService,
		service.NewSettingService,
		service.NewDictionaryService,
		service.NewNotificationService,
		service.NewNotificationPublisher,

		// interfaces - handlers
		handler.NewAccountHandler,
		handler.NewAccountPermissionHandler,
		handler.NewAdminAssetHandler,
		handler.NewAdminAuthorizationHandler,
		handler.NewMenuHandler,
		handler.NewAdminDashboardHandler,
		handler.NewAdminLoginHistoryHandler,
		handler.NewAdminUserHandler,
		handler.NewAssetHandler,
		handler.NewAuthenticationHandler,
		handler.NewLoginHistoryHandler,
		handler.NewAdminAuditLogHandler,
		handler.NewAdminSettingHandler,
		handler.NewSettingHandler,
		handler.NewAdminDictionaryHandler,
		handler.NewDictionaryHandler,
		handler.NewAdminNotificationHandler,
		handler.NewNotificationHandler,

		// interfaces - handler groups (wire.Struct auto-fills exported fields)
		wire.Struct(new(api.AdminHandlers), "*"),
		wire.Struct(new(api.UserHandlers), "*"),

		// interfaces - router & server
		api.NewRouter,
		server.NewServer,
	)

	return &server.Server{}, nil
}
