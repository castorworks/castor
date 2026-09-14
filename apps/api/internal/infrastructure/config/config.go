package config

import (
	"github.com/hyperits/gosuite/db/postgres"
	"github.com/hyperits/gosuite/db/redis"
	"github.com/hyperits/gosuite/logger"
	aliyunsms "github.com/hyperits/gosuite/providers/aliyun/sms"
	smtpmail "github.com/hyperits/gosuite/providers/smtp/mail"
	"github.com/hyperits/gosuite/storage/s3"
)

// C 全局配置实例
var C = new(Config)

// Config 应用配置
type Config struct {
	General    General
	Logger     LoggerConfig
	GinLogger  GinLoggerConfig
	Security   SecurityConfig
	CORS       CORSConfig
	I18n       I18nConfig
	Captcha    CaptchaConfig
	VerifyCode VerifyCodeConfig
	Postgres   postgres.Config
	Redis      redis.Config
	S3         s3.S3Config
	Mail       smtpmail.Config
	AliyunSms  aliyunsms.Config
	Module     ModuleConfig
	Asset      AssetConfig
}

// General 通用配置
type General struct {
	AppName            string `default:"castor"`
	Address            string `default:":1234"`
	Development        bool   `default:"false"`
	JwtKey             string `default:""`
	JwtTimeoutHours    uint   `default:"2"`
	JwtMaxRefreshHours uint   `default:"168"`
	RegisterEnabled    bool   `default:"false"`
	ShutdownTimeout    int    `default:"10"`
	ReadHeaderTimeout  int    `default:"10"`
	ReadTimeout        int    `default:"60"`
	WriteTimeout       int    `default:"60"`
	IdleTimeout        int    `default:"120"`
	MaxHeaderBytes     int    `default:"1048576"`
	// MaxRequestBodyBytes 非 multipart 请求体（JSON 等）的最大字节数；上传接口使用 Asset.MaxUploadSize
	MaxRequestBodyBytes int64 `default:"10485760"`
	// TrustedProxies 受信任的反向代理 IP/CIDR。仅当请求来自这些地址时才采信 X-Forwarded-For，
	// 否则 ClientIP 为直连地址；部署在负载均衡/Ingress 之后必须配置，否则所有客户端共享同一限流桶。
	TrustedProxies []string `default:"[]"`
}

// CORSConfig CORS 配置
type CORSConfig struct {
	AllowOrigins     []string `default:"[\"http://localhost:3000\",\"http://localhost:3003\"]"`
	AllowMethods     []string `default:"[\"GET\", \"POST\", \"PUT\", \"DELETE\", \"PATCH\", \"OPTIONS\"]"`
	AllowHeaders     []string `default:"[\"Content-Type\", \"Content-Length\", \"Accept-Encoding\", \"X-CSRF-Token\", \"Authorization\", \"accept\", \"origin\", \"Cache-Control\", \"X-Requested-With\"]"`
	AllowCredentials bool     `default:"true"`
}

// CaptchaConfig 验证码配置
type CaptchaConfig struct {
	ExpireSeconds int `default:"600"`
	Length        int `default:"4"`
	Width         int `default:"120"`
	Height        int `default:"40"`
}

// VerifyCodeConfig 验证码配置
type VerifyCodeConfig struct {
	ExpireSeconds int `default:"600"` // 验证码有效期（秒）
	Length        int `default:"6"`   // 验证码位数（6-10）
	MaxAttempts   int `default:"5"`   // 单个验证码允许的最大校验次数，超过后验证码作废
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	FilePath   string `default:"./logs/app.log"`
	MaxSize    int    `default:"32"`
	MaxBackups int    `default:"15"`
	MaxAge     int    `default:"15"`
	Compress   bool   `default:"true"`
	Level      string `default:"info"`
	Console    bool   `default:"true"`
	Caller     bool   `default:"false"`
}

// GinLoggerConfig Gin 框架日志配置
type GinLoggerConfig struct {
	FilePath   string `default:"./logs/gin.log"`
	MaxSize    int    `default:"32"`
	MaxBackups int    `default:"15"`
	MaxAge     int    `default:"15"`
	Compress   bool   `default:"true"`
	Console    bool   `default:"true"`
}

// ToLoggerConfig 转换为 logger.Config
func (c *LoggerConfig) ToLoggerConfig() *logger.Config {
	level := logger.InfoLevel
	switch c.Level {
	case "debug":
		level = logger.DebugLevel
	case "info":
		level = logger.InfoLevel
	case "warn":
		level = logger.WarnLevel
	case "error":
		level = logger.ErrorLevel
	}

	return &logger.Config{
		FilePath:   c.FilePath,
		MaxSize:    c.MaxSize,
		MaxBackups: c.MaxBackups,
		MaxAge:     c.MaxAge,
		Compress:   c.Compress,
		Level:      level,
		Console:    c.Console,
		Caller:     c.Caller,
	}
}

// ModuleConfig 模块配置
type ModuleConfig struct {
	FileEnable bool `default:"true"`
}

// AssetConfig 资产上传配置
type AssetConfig struct {
	MaxUploadSize     int64    `default:"104857600"`                                                                                                                                                                                                                                     // 最大上传大小（字节），默认 100MB
	AllowedExtensions []string `default:"[\".jpg\",\".jpeg\",\".png\",\".gif\",\".webp\",\".svg\",\".mp4\",\".avi\",\".mov\",\".mp3\",\".wav\",\".pdf\",\".doc\",\".docx\",\".xls\",\".xlsx\",\".ppt\",\".pptx\",\".txt\",\".md\",\".csv\",\".zip\",\".rar\",\".7z\",\".tar\",\".gz\"]"` // 允许的文件扩展名
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	RSAPrivateKeySecret string // AES-256-GCM 密钥，用于加密 Redis 中的 RSA 私钥
}

// I18nConfig 国际化配置
type I18nConfig struct {
	RootPath           string   `default:"configs/i18n"`
	DefaultLanguage    string   `default:"zh"`
	SupportedLanguages []string `default:"[\"zh\", \"en\", \"ja\", \"ko\"]"`
}
