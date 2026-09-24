package constant

// JWT 相关常量
const (
	JWT_IDENTITY_KEY           = "id"
	JWT_USERNAME               = "username"
	JWT_NAME                   = "name"
	JWT_ACCOUNT_SOURCE         = "accountSource"
	JWT_REFRESH_TOKEN_DURATION = "refreshTokenDuration"
	JWT_AUTHORIZATION_SESSION  = "authorizationSessionId"
)

// 账号来源常量：取值即字典 account_source 的字典项值（大小写一致，
// dictionary.TestAccountSourcesHaveDictItems 把关）。
const (
	ACCOUNT_SOURCE_INTERNAL = "INTERNAL"
	// ACCOUNT_SOURCE_OIDC 由 OIDC 登录自动注册的账号
	ACCOUNT_SOURCE_OIDC = "OIDC"
	// add your own

	CASTOR_RESERVED_USER_ROLE_NAME = "castor-internal"
)

// 登录方式常量
const (
	LOGIN_METHOD_EMAIL    = "email"
	LOGIN_METHOD_MOBILE   = "mobile"
	LOGIN_METHOD_PASSWORD = "password"
	// LOGIN_METHOD_OIDC 经外部身份提供方登录（只写登录历史，不是 /auth/login 的 method 参数）
	LOGIN_METHOD_OIDC = "oidc"
	// add your own
)

// RSA 相关常量
const (
	// RSA_CURRENT_KEY_PAIR_ID Redis 中存储当前 RSA 密钥对 ID 的键
	RSA_CURRENT_KEY_PAIR_ID = "rsa:current_keypair_id"
)

// 字典相关常量
const (
	// DICT_SNAPSHOT_CACHE_KEY Redis 中缓存字典读取快照的键
	DICT_SNAPSHOT_CACHE_KEY = "dict:snapshot"
)
