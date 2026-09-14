package main

import (
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/infrastructure/config"
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

func provideRsaKeyConfig() service.RsaKeyConfig {
	return service.RsaKeyConfig{
		PrivateKeySecret: config.C.Security.RSAPrivateKeySecret,
		FallbackSecret:   config.C.General.JwtKey,
	}
}
