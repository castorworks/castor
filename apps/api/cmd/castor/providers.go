package main

import (
	"time"

	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/castorworks/castor/internal/pkg/secretbox"
)

// 组合根：把基础设施配置转换为应用层策略，application 包不直接读取 config.C。

func provideRuntimePolicy() service.RuntimePolicy {
	return service.RuntimePolicy{
		Development:     config.C.General.Development,
		RegisterEnabled: config.C.General.RegisterEnabled,
	}
}

func provideAssetPolicy() service.AssetPolicy {
	return service.AssetPolicy{
		MaxUploadSize:     config.C.Asset.MaxUploadSize,
		AllowedExtensions: config.C.Asset.AllowedExtensions,
	}
}

func provideAssetSweepPolicy() service.AssetSweepPolicy {
	return service.AssetSweepPolicy{
		OrphanTTL: time.Duration(config.C.Asset.OrphanAttachmentTTLHours) * time.Hour,
	}
}

func provideRsaKeyConfig() service.RsaKeyConfig {
	return service.RsaKeyConfig{
		PrivateKeySecret: config.C.Security.RSAPrivateKeySecret,
		FallbackSecret:   config.C.General.JwtKey,
	}
}

// provideKeyring 敏感字段加密的主密钥。非开发模式由配置校验保证已设置；
// 开发环境未设置时从 JwtKey 派生，免得本地每个环境都要多配一个密钥。
func provideKeyring() (*secretbox.Keyring, error) {
	key := config.C.Security.DataEncryptionKey
	if key == "" && config.C.General.Development {
		key = "development-derived:" + config.C.General.JwtKey
	}
	return secretbox.NewKeyring(key)
}

func provideSSOPolicy() service.SSOPolicy {
	return service.SSOPolicy{
		PublicURL:           config.C.General.PublicURL,
		AllowInsecureIssuer: config.C.General.Development,
	}
}

func provideNotificationMailPolicy() service.NotificationMailPolicy {
	return service.NotificationMailPolicy{PublicURL: config.C.General.PublicURL}
}
