# 安全策略

[English](SECURITY.md) | **简体中文** | [日本語](SECURITY.ja.md) | [한국어](SECURITY.ko.md)

## 支持的版本

只有最新发布版本和 `main` 分支会收到安全修复。

## 报告漏洞

请**不要**为安全问题创建公开 issue。

请通过 [GitHub Security Advisories](https://github.com/castorworks/castor/security/advisories/new) 私下报告，并附上受影响的版本或提交、复现步骤和影响范围。

我们力争在 3 个工作日内确认收到报告，并在 30 天内为已确认的高危问题发布修复或缓解措施。除非报告者另有要求，我们会在公告中致谢。

## 部署检查清单

- 设置 `Development = false`，并替换所有示例密钥（`JwtKey`、RSA 密钥、数据库 / Redis / S3 凭据）。
- 为 `init-db` 设置 `CASTOR_DEFAULT_ADMIN_PASSWORD`，首次登录后修改管理员密码。
- 在 API 与 Web 前终止 TLS，并按负载均衡器配置 `TrustedProxies`。
- 将 PostgreSQL、Redis 和 S3 置于私有网络中。
