package service

import (
	"slices"
	"strings"
	"time"
)

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
	// AllowedMimeTypes 允许的真实内容类型（按文件字节嗅探，而非客户端声明），为空表示不限制
	AllowedMimeTypes []string
}

// AssetSweepPolicy 孤儿业务附件的清扫策略
type AssetSweepPolicy struct {
	// OrphanTTL 无人引用的业务附件保留多久后回收；<=0 表示不清扫（不注册清扫任务）。
	// 清扫频率由定时任务 sweepOrphanAttachments 的 cron 决定。
	OrphanTTL time.Duration
}

// narrowedBy 返回在全局策略之内、再按业务字段 field 收紧后的策略：
// 业务只能把限制调严（更小的体积、更少的类型），不能突破全局上限。
func (p AssetPolicy) narrowedBy(field AssetPolicy) AssetPolicy {
	out := p
	if field.MaxUploadSize > 0 && (p.MaxUploadSize <= 0 || field.MaxUploadSize < p.MaxUploadSize) {
		out.MaxUploadSize = field.MaxUploadSize
	}
	if len(field.AllowedExtensions) > 0 {
		narrowed := make([]string, 0, len(field.AllowedExtensions))
		for _, ext := range field.AllowedExtensions {
			if p.allowsExtension(ext) {
				narrowed = append(narrowed, ext)
			}
		}
		// 交集为空时保留一个不可能命中的占位，避免空切片被解读成"不限制"。
		if len(narrowed) == 0 {
			narrowed = []string{""}
		}
		out.AllowedExtensions = narrowed
	}
	if len(field.AllowedMimeTypes) > 0 {
		out.AllowedMimeTypes = field.AllowedMimeTypes
	}
	return out
}

func (p AssetPolicy) allowsExtension(ext string) bool {
	if len(p.AllowedExtensions) == 0 {
		return true
	}
	return slices.ContainsFunc(p.AllowedExtensions, func(allowed string) bool {
		return allowed != "" && strings.EqualFold(ext, allowed)
	})
}

func (p AssetPolicy) allowsMimeType(detected string) bool {
	return len(p.AllowedMimeTypes) == 0 || slices.Contains(p.AllowedMimeTypes, detected)
}

// RsaKeyConfig RSA 私钥在 Redis 中加密存储所用的密钥材料
type RsaKeyConfig struct {
	// PrivateKeySecret 专用加密密钥，优先使用
	PrivateKeySecret string
	// FallbackSecret 未配置专用密钥时的降级密钥（JWT 密钥）
	FallbackSecret string
}

// SSOPolicy OIDC 登录的运行策略
type SSOPolicy struct {
	// PublicURL 站点对外地址（不带路径），回调地址 = PublicURL + /api/v1/auth/oidc/{code}/callback；
	// 为空时不能启用任何身份提供方
	PublicURL string
	// AllowInsecureIssuer 允许 http 的 issuer（仅开发模式，便于连本地的测试身份提供方）
	AllowInsecureIssuer bool
}
