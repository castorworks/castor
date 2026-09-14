package server

import (
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/castorworks/castor/internal/interfaces/api/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/natefinch/lumberjack"
)

// multipartOverheadBytes leaves room for multipart boundaries and form fields around the file.
const multipartOverheadBytes = 1 << 20

// NewGinEngine 创建 Gin 引擎
func NewGinEngine() (*gin.Engine, error) {
	engine := gin.Default()
	// gin.Context 作为 context.Context 传入下层时，Done/Deadline/Value 回落到请求 context
	engine.ContextWithFallback = true

	err := initEngine(engine)
	if err != nil {
		return nil, err
	}

	// 自定义时间 CustomTime
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterCustomTypeFunc(ValidateJSONDateType, shared.CustomTime{})
	}

	return engine, nil
}

func initEngine(engine *gin.Engine) error {
	if err := engine.SetTrustedProxies(config.C.General.TrustedProxies); err != nil {
		return err
	}
	if !config.C.General.Development {
		gin.SetMode("release")
	}

	// gin 框架日志（仅记录路由注册、启动信息等框架级别日志）
	// 请求日志统一由 middleware.LogRequests() 结构化记录到 app.log
	ginCfg := config.C.GinLogger
	ginLogFile := &lumberjack.Logger{
		Filename:   ginCfg.FilePath,
		MaxSize:    ginCfg.MaxSize,
		MaxBackups: ginCfg.MaxBackups,
		MaxAge:     ginCfg.MaxAge,
		Compress:   ginCfg.Compress,
	}
	writers := []io.Writer{ginLogFile}
	if ginCfg.Console {
		writers = append(writers, os.Stdout)
	}
	gin.DefaultWriter = io.MultiWriter(writers...)

	// CORS
	engine.Use(CORSMiddleware())

	// Browser security headers
	engine.Use(SecurityHeaders())

	// Request ID（用于日志追踪）
	engine.Use(middleware.RequestID())

	// 请求体大小限制：JSON 等普通请求使用 General.MaxRequestBodyBytes，multipart 上传使用 Asset.MaxUploadSize（附加表单开销）
	engine.Use(middleware.BodyLimit(config.C.General.MaxRequestBodyBytes, config.C.Asset.MaxUploadSize+multipartOverheadBytes))

	// i18n
	engine.Use(middleware.GinI18nLocalize())

	// API 访问日志（结构化输出，带 requestId 便于链路追踪）
	engine.Use(middleware.LogRequests())

	return nil
}

// ValidateJSONDateType 验证 JSON 日期类型
func ValidateJSONDateType(field reflect.Value) interface{} {
	if field.Type() == reflect.TypeOf(shared.CustomTime{}) {
		timeStr := field.Interface().(shared.CustomTime).String()
		if timeStr == "0001-01-01 00:00:00" {
			return nil
		}
		return timeStr
	}
	return nil
}

// CORSMiddleware CORS 中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			c.Next()
			return
		}

		allowedOrigins := config.C.CORS.AllowOrigins
		originAllowed := false
		c.Header("Vary", "Origin")
		for _, allowed := range allowedOrigins {
			if allowed == "*" || allowed == origin {
				originAllowed = true
				break
			}
		}

		if originAllowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			if config.C.CORS.AllowCredentials {
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(config.C.CORS.AllowHeaders, ", "))
			c.Writer.Header().Set("Access-Control-Allow-Methods", strings.Join(config.C.CORS.AllowMethods, ", "))
			c.Writer.Header().Set("Access-Control-Expose-Headers", "X-Request-Id, Content-Disposition")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
