package service

// 应用层运行策略。application 不读取全局配置，由组合根（cmd/castor）从配置构造后注入，
// 测试可直接传入所需取值，无需修改全局状态。

// RuntimePolicy 认证与登录流程的运行策略
type RuntimePolicy struct {
	// Development 开发模式：跳过图形验证码与短信发送，放宽验证码发送频率限制
	Development bool
	// RegisterEnabled 静态注册开关，关闭时忽略系统设置中的动态开关
	RegisterEnabled bool
}

// AssetPolicy 资产上传策略
type AssetPolicy struct {
	// MaxUploadSize 单文件最大字节数，<=0 表示不限制
	MaxUploadSize int64
	// AllowedExtensions 允许的扩展名（含点号，不区分大小写），为空表示不限制
	AllowedExtensions []string
}

// RsaKeyConfig RSA 私钥在 Redis 中加密存储所用的密钥材料
type RsaKeyConfig struct {
	// PrivateKeySecret 专用加密密钥，优先使用
	PrivateKeySecret string
	// FallbackSecret 未配置专用密钥时的降级密钥（JWT 密钥）
	FallbackSecret string
}
