package database

import (
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/hyperits/gosuite/db/postgres"
)

// TestNewPostgresConnectsWithRegionalTimeZone 走生产同款路径（配置 → gosuite 客户端 → gorm）连一次真库。
//
// 其余集成测试都是拿 DSN 直接开 gorm，绕过了 gosuite 的 DSN 构造。带 "/" 的时区（Asia/Shanghai）
// 一旦在 DSN 里被 URL 编码成 %2F，gorm 的 postgres 驱动会报 unknown time zone，服务起不来。
func TestNewPostgresConnectsWithRegionalTimeZone(t *testing.T) {
	dsn := os.Getenv("CASTOR_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("CASTOR_TEST_POSTGRES_DSN is not configured")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Scheme == "" {
		t.Skip("CASTOR_TEST_POSTGRES_DSN is not in URL form")
	}
	port, _ := strconv.Atoi(u.Port())
	password, _ := u.User.Password()

	original := config.C
	config.C = new(config.Config)
	t.Cleanup(func() { config.C = original })
	config.C.Postgres = postgres.Config{
		Host:     u.Hostname(),
		Port:     port,
		Username: u.User.Username(),
		Password: password,
		DbName:   u.Path[1:],
		SSLMode:  u.Query().Get("sslmode"),
		TimeZone: "Asia/Shanghai",
	}

	db, err := NewPostgres()
	if err != nil {
		t.Fatalf("NewPostgres() error = %v", err)
	}
	t.Cleanup(func() { _ = Close(db) })

	var timeZone string
	if err := db.Raw("SHOW TimeZone").Scan(&timeZone).Error; err != nil {
		t.Fatal(err)
	}
	if timeZone != "Asia/Shanghai" {
		t.Fatalf("session TimeZone = %q, want Asia/Shanghai", timeZone)
	}
}
