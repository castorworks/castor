package config

import (
	"strings"
	"testing"
)

func TestExampleConfigLoads(t *testing.T) {
	original := C
	C = new(Config)
	t.Cleanup(func() {
		C = original
	})

	if err := Load("../../../configs/app/example.config.toml"); err != nil {
		t.Fatalf("load example config: %v", err)
	}
	if C.General.Development {
		t.Fatal("example config must default to production mode")
	}
	if err := Validate(); err == nil || !strings.Contains(err.Error(), "example or development value") {
		t.Fatalf("example placeholder secrets must be rejected in production mode, got %v", err)
	}

	C.General.Development = true
	C.CORS.AllowOrigins = []string{"*"}
	if err := Validate(); err != nil {
		t.Fatalf("validate example config in development mode: %v", err)
	}
}

func TestValidateRejectsPlaceholderSecretsInProduction(t *testing.T) {
	base := func() {
		C = new(Config)
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
		"no attempts limit":  func() { C.VerifyCode.MaxAttempts = 0 },
		"no body size limit": func() { C.General.MaxRequestBodyBytes = 0 },
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
	t.Setenv("CASTOR_S3_BUCKET", "deployment-bucket")
	t.Setenv("CASTOR_CORS_ORIGINS", "https://castor.example.com")
	OverrideFromEnv()
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
	if C.S3.Bucket != "deployment-bucket" {
		t.Fatal("bucket env override was ignored")
	}
	if C.Security.RSAPrivateKeySecret != "0123456789abcdef0123456789abcdef" {
		t.Fatal("RSA env override was ignored")
	}
}

func TestValidateRejectsWeakProductionSecurity(t *testing.T) {
	original := C
	C = new(Config)
	t.Cleanup(func() { C = original })
	C.General.Development = false
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
