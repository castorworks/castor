package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/castorworks/castor/internal/infrastructure/database"
	"github.com/castorworks/castor/internal/interfaces/api"
	"github.com/gin-gonic/gin"
	"github.com/hyperits/gosuite/logger"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Server HTTP 服务器
type Server struct {
	engine          *gin.Engine
	apiRouter       *api.Router
	auditLogService service.AuditLogService
	db              *gorm.DB
	redis           redis.UniversalClient
}

// NewServer 创建服务器
func NewServer(engine *gin.Engine, apiRouter *api.Router, auditLogService service.AuditLogService, db *gorm.DB, rdb redis.UniversalClient) *Server {
	return &Server{
		engine:          engine,
		apiRouter:       apiRouter,
		auditLogService: auditLogService,
		db:              db,
		redis:           rdb,
	}
}

// Start 启动服务器，并在收到 SIGINT/SIGTERM 或监听失败时按顺序优雅关闭：
// 停止 HTTP → 等待异步审计日志落库 → 关闭 Redis → 关闭数据库。
func (s *Server) Start() {
	// 注册所有路由
	s.apiRouter.With(s.engine)

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:              config.C.General.Address,
		Handler:           s.engine,
		ReadHeaderTimeout: time.Duration(config.C.General.ReadHeaderTimeout) * time.Second,
		ReadTimeout:       time.Duration(config.C.General.ReadTimeout) * time.Second,
		WriteTimeout:      time.Duration(config.C.General.WriteTimeout) * time.Second,
		IdleTimeout:       time.Duration(config.C.General.IdleTimeout) * time.Second,
		MaxHeaderBytes:    config.C.General.MaxHeaderBytes,
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Infof("Server starting on %s", config.C.General.Address)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	exitCode := 0
	select {
	case sig := <-quit:
		logger.Infof("Received %s, shutting down server...", sig)
	case err := <-serveErr:
		logger.Errorf("Server listen error: %v", err)
		exitCode = 1
	}

	if !s.shutdown(srv) {
		exitCode = 1
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

// shutdown releases resources in dependency order and reports whether every step succeeded.
func (s *Server) shutdown(srv *http.Server) bool {
	ok := true

	shutdownTimeout := time.Duration(config.C.General.ShutdownTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// 1. 停止接收新请求并等待进行中的请求完成
	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("HTTP server did not shut down cleanly: %v", err)
		ok = false
	}

	// 2. 等待异步审计日志写入完成（依赖数据库，必须在关闭连接前完成）
	logger.Infof("Waiting for pending audit logs to complete...")
	s.auditLogService.Wait()

	// 3. 关闭 Redis
	if s.redis != nil {
		if err := s.redis.Close(); err != nil {
			logger.Errorf("Close redis: %v", err)
			ok = false
		}
	}

	// 4. 关闭数据库连接池
	if s.db != nil {
		if err := database.Close(s.db); err != nil {
			logger.Errorf("Close database: %v", err)
			ok = false
		}
	}

	if ok {
		logger.Infof("Server exited gracefully")
	}
	return ok
}
