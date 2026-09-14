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

// 账号来源常量
const (
	ACCOUNT_SOURCE_INTERNAL = "internal"
	// add your own

	CASTOR_RESERVED_USER_ROLE_NAME = "castor-internal"
)

// 登录方式常量
const (
	LOGIN_METHOD_EMAIL    = "email"
	LOGIN_METHOD_MOBILE   = "mobile"
	LOGIN_METHOD_PASSWORD = "password"
	// add your own
)

// RSA 相关常量
const (
	// RSA_CURRENT_KEY_PAIR_ID Redis 中存储当前 RSA 密钥对 ID 的键
	RSA_CURRENT_KEY_PAIR_ID = "rsa:current_keypair_id"
)
