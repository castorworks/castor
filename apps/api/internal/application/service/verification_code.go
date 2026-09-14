package service

import "context"

// VerificationPurposeAuth scopes one-time codes sent by POST /auth/code. The same code
// proves control of an email address or phone number for both verification-code login
// and password reset, so both flows share this purpose.
const VerificationPurposeAuth = "auth"

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
