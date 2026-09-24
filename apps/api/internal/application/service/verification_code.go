package service

import (
	"context"
	"strings"
)

// One-time codes are scoped by purpose so a code issued for one intent can never be
// replayed against another: a login code must not reset a password, and neither may
// bind a contact to an account.
const (
	// VerificationPurposeAuth scopes codes used for verification-code login.
	VerificationPurposeAuth = "auth"
	// VerificationPurposeReset scopes codes used for password reset.
	VerificationPurposeReset = "reset"
	// VerificationPurposeBind scopes codes that prove control of a contact before it is
	// bound to the signed-in account. It is chosen by the server, never by the client.
	VerificationPurposeBind = "bind"
)

// NormalizeVerificationPurpose 归一化 POST /auth/code 的 purpose 参数，接受任意大小写
// 形式；空值默认为 VerificationPurposeAuth。只有客户端可自选的用途会被接受，
// VerificationPurposeBind 由服务端绑定流程自行指定，不在此列。
// 第二个返回值为 false 表示用途无法识别。
func NormalizeVerificationPurpose(purpose string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(purpose)) {
	case "":
		return VerificationPurposeAuth, true
	case VerificationPurposeAuth:
		return VerificationPurposeAuth, true
	case VerificationPurposeReset:
		return VerificationPurposeReset, true
	default:
		return "", false
	}
}

// VerificationCodeStore issues and checks one-time verification codes.
//
// Implementations must scope codes by (purpose, target), consume a code on successful
// verification, compare codes in constant time, and invalidate a code after a bounded
// number of failed attempts so it cannot be brute-forced.
type VerificationCodeStore interface {
	// Generate issues a fresh code for target, replacing any previous code and
	// resetting the failed-attempt counter.
	Generate(ctx context.Context, purpose, target string) (string, error)
	// Verify reports whether code matches the active code for target. A successful
	// verification consumes the code.
	Verify(ctx context.Context, purpose, target, code string) (bool, error)
}
