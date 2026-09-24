package config

import (
	"os"
	"strings"
	"testing"

	smtpmail "github.com/hyperits/gosuite/providers/smtp/mail"
)

const templatePath = "../../../../../deploy/config/api.config.example.toml"

// 模板不含任何密钥：原样使用必须被拒绝，只注入部署提供的密钥后即可通过校验。
func TestTemplateRequiresInjectedSecrets(t *testing.T) {
	original := C
	C = new(Config)
	t.Cleanup(func() {
		C = original
	})

	if err := Load(templatePath); err != nil {
		t.Fatalf("load template: %v", err)
	}
	if C.General.Development {
		t.Fatal("template must default to production mode")
	}
	if err := Validate(); err == nil {
		t.Fatal("template without injected secrets must be rejected")
	}

	t.Setenv("CASTOR_JWT_KEY", strings.Repeat("j", 32))
	t.Setenv("CASTOR_RSA_SECRET", strings.Repeat("r", 32))
	OverrideFromEnv()
	if err := Validate(); err == nil {
		t.Fatal("template without a data encryption key must be rejected")
	}
	t.Setenv("CASTOR_DATA_KEY", strings.Repeat("d", 32))
	OverrideFromEnv()
	if err := Validate(); err != nil {
		t.Fatalf("template with injected secrets rejected: %v", err)
	}
}

func TestValidateRejectsPlaceholderSecretsInProduction(t *testing.T) {
	base := func() {
		C = new(Config)
		C.General.InstanceID = "castor"
		C.General.JwtKey = "0123456789abcdef0123456789abcdef-jwt"
		C.General.JwtTimeoutHours = 2
		C.General.JwtMaxRefreshHours = 168
		C.General.MaxRequestBodyBytes = 1 << 20
		C.VerifyCode.ExpireSeconds = 600
		C.VerifyCode.MaxAttempts = 5
		C.Postgres.Host = "postgres"
		C.Redis.Address = "redis:6379"
		C.CORS.AllowOrigins = []string{"https://castor.example.com"}
		C.Security.RSAPrivateKeySecret = "0123456789abcdef0123456789abcdef"
		C.Security.DataEncryptionKey = "fedcba9876543210fedcba9876543210-data"
	}
	original := C
	t.Cleanup(func() { C = original })

	base()
	if err := Validate(); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}

	cases := map[string]func(){
		"jwt change-me":      func() { C.General.JwtKey = "development-only-jwt-key-change-me-32-bytes" },
		"jwt example":        func() { C.General.JwtKey = "an-example-jwt-key-that-is-32-bytes-long" },
		"rsa change-me":      func() { C.Security.RSAPrivateKeySecret = "change-me-32-byte-secret-0000000" },
		"postgres dev":       func() { C.Postgres.Password = "dev" },
		"s3 dev":             func() { C.S3.Secret = "dev123456" },
		"mail password":      func() { C.Mail.Password = "password" },
		"sms example":        func() { C.AliyunSms.SecretKey = "example" },
		"timeout > refresh":  func() { C.General.JwtTimeoutHours = 200 },
		"empty rsa":          func() { C.Security.RSAPrivateKeySecret = "" },
		"empty data key":     func() { C.Security.DataEncryptionKey = "" },
		"short data key":     func() { C.Security.DataEncryptionKey = "too-short" },
		"data key example":   func() { C.Security.DataEncryptionKey = "an-example-data-key-that-is-32-bytes" },
		"public url http":    func() { C.General.PublicURL = "http://castor.example.com" },
		"public url path":    func() { C.General.PublicURL = "https://castor.test/app" },
		"public url no host": func() { C.General.PublicURL = "castor.test" },
		"no attempts limit":  func() { C.VerifyCode.MaxAttempts = 0 },
		"no body size limit": func() { C.General.MaxRequestBodyBytes = 0 },
		"empty instance id":  func() { C.General.InstanceID = "" },
		"instance id colon":  func() { C.General.InstanceID = "tenant:a" },
		"instance id brace":  func() { C.General.InstanceID = "{tenant}" },
		"instance id upper":  func() { C.General.InstanceID = "TenantA" },
		"instance id dash":   func() { C.General.InstanceID = "-tenant" },
		"instance id long":   func() { C.General.InstanceID = strings.Repeat("a", 33) },
	}
	for name, mutate := range cases {
		base()
		mutate()
		if err := Validate(); err == nil {
			t.Errorf("%s: expected validation error", name)
		}
	}

	base()
	C.General.Development = true
	C.General.JwtKey = "development-only-jwt-key-change-me-32-bytes"
	C.Postgres.Password = "dev"
	if err := Validate(); err != nil {
		t.Fatalf("development mode must allow local placeholder secrets: %v", err)
	}
}

func TestDeploymentConfigUsesEnvironmentSecrets(t *testing.T) {
	original := C
	C = new(Config)
	t.Cleanup(func() { C = original })
	if err := Load("../../../../../deploy/config/api.config.example.toml"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CASTOR_JWT_KEY", strings.Repeat("j", 32))
	t.Setenv("CASTOR_RSA_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("CASTOR_DATA_KEY", strings.Repeat("d", 40))
	t.Setenv("CASTOR_PUBLIC_URL", "https://castor.test/")
	t.Setenv("CASTOR_S3_BUCKET", "deployment-bucket")
	t.Setenv("CASTOR_INSTANCE_ID", "tenant-a")
	t.Setenv("CASTOR_CORS_ORIGINS", "https://castor.example.com")
	OverrideFromEnv()
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
	if C.General.InstanceID != "tenant-a" {
		t.Fatal("instance id env override was ignored")
	}
	if C.S3.Bucket != "deployment-bucket" {
		t.Fatal("bucket env override was ignored")
	}
	if C.Security.RSAPrivateKeySecret != "0123456789abcdef0123456789abcdef" {
		t.Fatal("RSA env override was ignored")
	}
	if C.Security.DataEncryptionKey != strings.Repeat("d", 40) || C.General.PublicURL != "https://castor.test" {
		t.Fatal("data key / public URL env overrides were ignored")
	}
}

func TestValidateRejectsWeakProductionSecurity(t *testing.T) {
	original := C
	C = new(Config)
	t.Cleanup(func() { C = original })
	C.General.Development = false
	C.General.InstanceID = "castor"
	C.General.JwtKey = strings.Repeat("j", 32)
	C.General.JwtTimeoutHours = 1
	C.General.JwtMaxRefreshHours = 1
	C.General.MaxRequestBodyBytes = 1 << 20
	C.VerifyCode.ExpireSeconds = 600
	C.VerifyCode.MaxAttempts = 5
	C.Postgres.Host = "postgres"
	C.Redis.Address = "redis:6379"
	C.CORS.AllowOrigins = []string{"*"}
	C.Security.RSAPrivateKeySecret = strings.Repeat("r", 32)

	if err := Validate(); err == nil {
		t.Fatal("expected wildcard CORS to be rejected in production")
	}

	C.CORS.AllowOrigins = []string{"https://castor.example.com"}
	C.Security.RSAPrivateKeySecret = "too-short"
	if err := Validate(); err == nil {
		t.Fatal("expected weak RSA secret to be rejected in production")
	}

	C.Security.RSAPrivateKeySecret = strings.Repeat("r", 32)
	C.General.JwtKey = strings.Repeat("j", 31)
	if err := Validate(); err == nil {
		t.Fatal("expected weak JWT secret to be rejected")
	}

	C.General.JwtKey = strings.Repeat("j", 32)
	C.CORS.AllowOrigins = []string{"https://castor.example.com/path"}
	if err := Validate(); err == nil {
		t.Fatal("expected CORS origin with a path to be rejected")
	}
}

// gosuite v1 在创建客户端时校验配置，校验不过服务直接起不来。
// 运维照着抄的模板必须能通过这些校验，模板里也不能留着结构体上不存在的死键。
func TestConfigTemplateSatisfiesGosuiteValidation(t *testing.T) {
	original := C
	C = new(Config)
	t.Cleanup(func() { C = original })
	if err := Load(templatePath); err != nil {
		t.Fatal(err)
	}
	if C.Mail.Host != "" {
		t.Errorf("the template must leave email disabled, got Mail.Host %q", C.Mail.Host)
	}
	t.Setenv("CASTOR_S3_ACCESS_KEY", "access")
	t.Setenv("CASTOR_S3_SECRET_KEY", "secret")
	// 运维启用邮件时注入的配置
	t.Setenv("CASTOR_MAIL_HOST", "smtp.castor.test")
	t.Setenv("CASTOR_MAIL_FROM", "noreply@castor.example.com")
	t.Setenv("CASTOR_MAIL_TLS_POLICY", "starttls")
	OverrideFromEnv()

	if err := C.Mail.Validate(); err != nil {
		t.Errorf("Mail: %v", err)
	}
	if C.Mail.From != "noreply@castor.example.com" || C.Mail.TLSPolicy != smtpmail.TLSPolicyStartTLS {
		t.Errorf("mail env overrides ignored: From=%q TLSPolicy=%q", C.Mail.From, C.Mail.TLSPolicy)
	}
	if err := C.S3.Validate(); err != nil {
		t.Errorf("S3: %v", err)
	}
	if err := C.Redis.Validate(); err != nil {
		t.Errorf("Redis: %v", err)
	}

	raw, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, deadKey := range []string{"SentinelMasterName", "\nSSL =", "AppName"} {
		if strings.Contains(string(raw), deadKey) {
			t.Errorf("template still documents %q, which no config struct reads", strings.TrimSpace(deadKey))
		}
	}
}

func TestMailTLSPolicyTypoFailsValidation(t *testing.T) {
	cfg := smtpmail.Config{Host: "smtp.example.com", Port: 587, TLSPolicy: "startls"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("an unknown TLS policy must be rejected instead of silently falling back")
	}
}

// 示例域名的 SMTP 永远连不上：当成已配置会让通知的"发送邮件"出现、每封信都重试到失败。
// 开发模式同样拒绝；dev.py prepare 会把旧模板留下的占位地址清空。
func TestValidateRejectsExampleMailHost(t *testing.T) {
	original := C
	t.Cleanup(func() { C = original })
	for _, development := range []bool{true, false} {
		for host, ok := range map[string]bool{
			"":                  true,
			"smtp.gmail.com":    true,
			"smtp.example.com":  false,
			"SMTP.Example.Org.": false,
			"mail.example":      false,
			"relay.invalid":     false,
			"example.com.cn":    true,
		} {
			C = new(Config)
			C.General.InstanceID = "castor"
			C.General.Development = development
			C.General.JwtKey = "0123456789abcdef0123456789abcdef-jwt"
			C.General.JwtTimeoutHours = 2
			C.General.JwtMaxRefreshHours = 168
			C.General.MaxRequestBodyBytes = 1 << 20
			C.VerifyCode.ExpireSeconds = 600
			C.VerifyCode.MaxAttempts = 5
			C.Postgres.Host = "postgres"
			C.Redis.Address = "redis:6379"
			C.CORS.AllowOrigins = []string{"https://castor.example.com"}
			C.Security.RSAPrivateKeySecret = "0123456789abcdef0123456789abcdef"
			C.Security.DataEncryptionKey = "fedcba9876543210fedcba9876543210-data"
			C.Mail.Host = host
			err := Validate()
			if ok && err != nil {
				t.Errorf("development=%t host %q rejected: %v", development, host, err)
			}
			if !ok && (err == nil || !strings.Contains(err.Error(), "Mail.Host")) {
				t.Errorf("development=%t host %q accepted (err %v)", development, host, err)
			}
		}
	}
}
