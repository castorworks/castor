package config

import (
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/castorworks/castor/internal/pkg/toml"
	"github.com/creasty/defaults"
	smtpmail "github.com/hyperits/gosuite/providers/smtp/mail"
	"github.com/pkg/errors"
)

var once sync.Once

// MustLoad 必须加载配置，失败则 panic
func MustLoad(filename string) {
	once.Do(func() {
		if err := Load(filename); err != nil {
			panic(err)
		}
		OverrideFromEnv()
		if err := Validate(); err != nil {
			panic(err)
		}
	})
}

// Validate 校验关键配置项，缺失则返回错误
func Validate() error {
	if len([]byte(C.General.JwtKey)) < 32 {
		return errors.New("config: General.JwtKey must be at least 32 bytes (set in config.toml or env CASTOR_JWT_KEY)")
	}
	if !instanceIDPattern.MatchString(C.General.InstanceID) {
		return errors.New("config: General.InstanceID must be 1-32 characters of lowercase letters, digits and '-', starting with a letter or digit (env CASTOR_INSTANCE_ID)")
	}
	if C.Postgres.Host == "" {
		return errors.New("config: Postgres.Host is required")
	}
	if C.Redis.Address == "" {
		return errors.New("config: Redis.Address is required")
	}
	if C.General.JwtTimeoutHours == 0 || C.General.JwtMaxRefreshHours == 0 {
		return errors.New("config: JWT timeout values must be greater than zero")
	}
	if C.General.JwtTimeoutHours > C.General.JwtMaxRefreshHours {
		return errors.New("config: General.JwtTimeoutHours must not exceed General.JwtMaxRefreshHours")
	}
	if C.General.MaxRequestBodyBytes <= 0 {
		return errors.New("config: General.MaxRequestBodyBytes must be greater than zero")
	}
	if C.VerifyCode.ExpireSeconds <= 0 || C.VerifyCode.MaxAttempts <= 0 {
		return errors.New("config: VerifyCode.ExpireSeconds and VerifyCode.MaxAttempts must be greater than zero")
	}
	if err := validateCORSOrigins(C.CORS.AllowOrigins, C.General.Development); err != nil {
		return err
	}
	if !C.General.Development {
		if err := validatePlaceholderSecrets(); err != nil {
			return err
		}
		if err := validateProductionSecret(C.Security.RSAPrivateKeySecret, "Security.RSAPrivateKeySecret"); err != nil {
			return err
		}
		if len(C.Security.DataEncryptionKey) < 32 {
			return errors.New("config: Security.DataEncryptionKey must be at least 32 bytes (env CASTOR_DATA_KEY)")
		}
	}
	if err := validatePublicURL(C.General.PublicURL, C.General.Development); err != nil {
		return err
	}
	if isExampleHost(C.Mail.Host) {
		// 示例域名永远连不上：当成已配置会让邮件入口出现、每封信都重试到失败。
		return errors.Errorf("config: Mail.Host %q is an example domain; set a real SMTP server or leave it empty to disable email (env CASTOR_MAIL_HOST)", C.Mail.Host)
	}
	return nil
}

// isExampleHost 判断主机名是否属于 RFC 2606 保留给文档示例的域名。
func isExampleHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	for _, reserved := range []string{"example.com", "example.net", "example.org", "example", "invalid"} {
		if host == reserved || strings.HasSuffix(host, "."+reserved) {
			return true
		}
	}
	return false
}

// validatePublicURL PublicURL 可以不配；配了就必须是不带路径的 http(s) 地址，非开发模式只允许 https。
func validatePublicURL(value string, development bool) error {
	if value == "" {
		return nil
	}
	u, err := url.Parse(value)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("config: General.PublicURL must be an origin such as https://castor.example.com (env CASTOR_PUBLIC_URL)")
	}
	if u.Scheme != "https" && !development {
		return errors.New("config: General.PublicURL must use https outside development mode")
	}
	return nil
}

// instanceIDPattern 限定实例标识的字符集：它会进入 Redis key 与 JWT，
// 不允许 ':'（key 分隔符）和 '{}'（Redis Cluster hash tag）。
var instanceIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

// placeholderMarkers identify values copied from example configs. Secrets containing any
// of them are rejected outside development mode.
var placeholderMarkers = []string{"change-me", "changeme", "change_me", "replace-me", "example", "development-only", "placeholder"}

// weakSecrets are exact example values shipped for local infrastructure.
var weakSecrets = map[string]bool{"dev": true, "dev123456": true, "password": true, "secret": true, "admin": true}

// validatePlaceholderSecrets rejects example or development secrets in non-development
// mode. Empty optional secrets (for unused integrations) are left to their consumers.
func validatePlaceholderSecrets() error {
	secrets := []struct {
		name     string
		value    string
		required bool
	}{
		{"General.JwtKey", C.General.JwtKey, true},
		{"Security.RSAPrivateKeySecret", C.Security.RSAPrivateKeySecret, true},
		{"Security.DataEncryptionKey", C.Security.DataEncryptionKey, true},
		{"Postgres.Password", C.Postgres.Password, false},
		{"Redis.Password", C.Redis.Password, false},
		{"S3.Secret", C.S3.Secret, false},
		{"Mail.Password", C.Mail.Password, false},
		{"AliyunSms.SecretKey", C.AliyunSms.SecretKey, false},
	}
	for _, secret := range secrets {
		if secret.value == "" && !secret.required {
			continue
		}
		if isPlaceholderSecret(secret.value) {
			return errors.Errorf("config: %s uses an example or development value; set a real secret (or env override) or enable General.Development for local use", secret.name)
		}
	}
	return nil
}

func isPlaceholderSecret(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if weakSecrets[normalized] {
		return true
	}
	for _, marker := range placeholderMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func validateCORSOrigins(origins []string, development bool) error {
	if len(origins) == 0 {
		return errors.New("config: CORS.AllowOrigins must contain at least one origin")
	}
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "*" {
			if !development {
				return errors.New("config: CORS.AllowOrigins must not contain '*' outside development")
			}
			continue
		}
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return errors.Errorf("config: CORS.AllowOrigins contains invalid origin %q", origin)
		}
	}
	return nil
}

func validateProductionSecret(value, name string) error {
	if len(value) != 32 || !utf8.ValidString(value) {
		return errors.Errorf("config: %s must be exactly 32 UTF-8 bytes", name)
	}
	for _, r := range value {
		if r > 127 || r < 32 {
			return errors.Errorf("config: %s must contain printable ASCII characters only", name)
		}
	}
	return nil
}

// Load 加载配置文件
func Load(filename string) error {
	if err := defaults.Set(C); err != nil {
		return err
	}

	buf, err := os.ReadFile(filename)
	if err != nil {
		return errors.Wrapf(err, "failed to read config file %s", filename)
	}

	err = toml.Unmarshal(buf, C)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal config %s", filename)
	}

	return nil
}

// OverrideFromEnv reads environment variables and overrides sensitive config fields.
func OverrideFromEnv() {
	if v := os.Getenv("CASTOR_INSTANCE_ID"); v != "" {
		C.General.InstanceID = v
	}
	if v := os.Getenv("CASTOR_JWT_KEY"); v != "" {
		C.General.JwtKey = v
	}

	// Postgres
	if v := os.Getenv("CASTOR_DB_HOST"); v != "" {
		C.Postgres.Host = v
	}
	if v := os.Getenv("CASTOR_DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			C.Postgres.Port = port
		}
	}
	if v := os.Getenv("CASTOR_DB_USER"); v != "" {
		C.Postgres.Username = v
	}
	if v := os.Getenv("CASTOR_DB_PASSWORD"); v != "" {
		C.Postgres.Password = v
	}
	if v := os.Getenv("CASTOR_DB_NAME"); v != "" {
		C.Postgres.DbName = v
	}

	// Redis
	if v := os.Getenv("CASTOR_REDIS_ADDR"); v != "" {
		C.Redis.Address = v
	}
	if v := os.Getenv("CASTOR_REDIS_PASSWORD"); v != "" {
		C.Redis.Password = v
	}

	// S3
	if v := os.Getenv("CASTOR_S3_ENDPOINT"); v != "" {
		C.S3.Endpoint = v
	}
	if v := os.Getenv("CASTOR_S3_ACCESS_KEY"); v != "" {
		C.S3.AccessKey = v
	}
	if v := os.Getenv("CASTOR_S3_SECRET_KEY"); v != "" {
		C.S3.Secret = v
	}

	if v := os.Getenv("CASTOR_S3_BUCKET"); v != "" {
		C.S3.Bucket = v
	}

	// Mail
	if v := os.Getenv("CASTOR_MAIL_HOST"); v != "" {
		C.Mail.Host = v
	}
	if v := os.Getenv("CASTOR_MAIL_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			C.Mail.Port = port
		}
	}
	if v := os.Getenv("CASTOR_MAIL_USERNAME"); v != "" {
		C.Mail.Username = v
	}
	if v := os.Getenv("CASTOR_MAIL_PASSWORD"); v != "" {
		C.Mail.Password = v
	}
	if v := os.Getenv("CASTOR_MAIL_FROM"); v != "" {
		C.Mail.From = v
	}
	if v := os.Getenv("CASTOR_MAIL_TLS_POLICY"); v != "" {
		C.Mail.TLSPolicy = smtpmail.TLSPolicy(v)
	}

	// Aliyun SMS
	if v := os.Getenv("CASTOR_SMS_ACCESS_KEY"); v != "" {
		C.AliyunSms.AccessKey = v
	}
	if v := os.Getenv("CASTOR_SMS_SECRET_KEY"); v != "" {
		C.AliyunSms.SecretKey = v
	}

	// CORS
	if v := os.Getenv("CASTOR_CORS_ORIGINS"); v != "" {
		origins := strings.Split(v, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		C.CORS.AllowOrigins = origins
	}

	// RSA private key encryption secret
	if v := os.Getenv("CASTOR_RSA_SECRET"); v != "" {
		C.Security.RSAPrivateKeySecret = v
	}
	if v := os.Getenv("CASTOR_DATA_KEY"); v != "" {
		C.Security.DataEncryptionKey = v
	}
	if v := os.Getenv("CASTOR_PUBLIC_URL"); v != "" {
		C.General.PublicURL = strings.TrimRight(v, "/")
	}
}
