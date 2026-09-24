package dto

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// 契约测试：这些 DTO 的 binding tag 必须接受前端真实发送的载荷。
//
// 背景：服务层单测直接构造结构体、绕过 Gin 绑定，导致以下两类缺陷长期不被发现：
//  1. 密码字段承载 RSA-OAEP 密文（2048 位 → base64 后 344 字符），却被 max=128 拒绝，
//     使注册/改密/重置/管理员建用户在绑定层就 400；
//  2. codeType 绑定要求大写、服务端常量为小写，两侧永不匹配。
// 任何收紧这些字段的改动都必须先让本测试通过。

// rsaCiphertextSample 模拟 RSA-2048 OAEP 密文的 base64 形式（344 字符）。
func rsaCiphertextSample() string {
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0xAB}, 256))
}

func bindJSON(t *testing.T, target any, body any) error {
	t.Helper()
	gin.SetMode(gin.TestMode)
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	return c.ShouldBindJSON(target)
}

func TestPasswordFieldsAcceptRSACiphertext(t *testing.T) {
	cipher := rsaCiphertextSample()
	if len(cipher) != 344 {
		t.Fatalf("expected a 344-char RSA-2048 ciphertext sample, got %d", len(cipher))
	}

	t.Run("register", func(t *testing.T) {
		var req UserRegisterReq
		body := map[string]string{"username": "alice", "password": cipher, "captchaId": "cid", "captchaCode": "1234"}
		if err := bindJSON(t, &req, body); err != nil {
			t.Fatalf("register must accept an RSA ciphertext password: %v", err)
		}
	})

	t.Run("admin creates user", func(t *testing.T) {
		var req UserPostReq
		body := map[string]string{"username": "alice", "name": "Alice", "password": cipher}
		if err := bindJSON(t, &req, body); err != nil {
			t.Fatalf("admin user create must accept an RSA ciphertext password: %v", err)
		}
	})

	t.Run("change password", func(t *testing.T) {
		var req UserPasswordPutReq
		body := map[string]string{"currentPassword": cipher, "newPassword": cipher}
		if err := bindJSON(t, &req, body); err != nil {
			t.Fatalf("password change must accept RSA ciphertext: %v", err)
		}
	})

	t.Run("reset password", func(t *testing.T) {
		var req UserPasswordResetPutReq
		body := map[string]string{"username": "alice@example.com", "newPassword": cipher, "confirmCode": "123456"}
		if err := bindJSON(t, &req, body); err != nil {
			t.Fatalf("password reset must accept RSA ciphertext: %v", err)
		}
	})
}

func TestConfirmCodeReqAcceptsCodeTypeCaseInsensitively(t *testing.T) {
	// 前端发送大写；内部常量为小写。两种形式都必须能通过绑定，
	// 由服务层归一化后分发。
	for _, codeType := range []string{"EMAIL", "MOBILE", "email", "mobile"} {
		t.Run(codeType, func(t *testing.T) {
			var req ConfirmCodeReq
			body := map[string]string{"codeType": codeType, "username": "alice@example.com"}
			if err := bindJSON(t, &req, body); err != nil {
				t.Fatalf("codeType %q must bind: %v", codeType, err)
			}
			if !strings.EqualFold(req.CodeType, codeType) {
				t.Fatalf("codeType round-trip mismatch: got %q", req.CodeType)
			}
		})
	}
}

func TestConfirmCodeReqPurposeIsOptional(t *testing.T) {
	// purpose 可缺省（等价于 auth），也接受任意大小写的 auth/reset。
	// 前端按此契约实现「忘记密码」入口，收紧该字段前必须先让本测试通过。
	t.Run("absent", func(t *testing.T) {
		var req ConfirmCodeReq
		body := map[string]string{"codeType": "EMAIL", "username": "alice@example.com"}
		if err := bindJSON(t, &req, body); err != nil {
			t.Fatalf("purpose must be optional: %v", err)
		}
		if req.Purpose != "" {
			t.Fatalf("purpose = %q, want empty", req.Purpose)
		}
	})

	for _, purpose := range []string{"auth", "AUTH", "reset", "RESET"} {
		t.Run(purpose, func(t *testing.T) {
			var req ConfirmCodeReq
			body := map[string]string{"codeType": "EMAIL", "username": "alice@example.com", "purpose": purpose}
			if err := bindJSON(t, &req, body); err != nil {
				t.Fatalf("purpose %q must bind: %v", purpose, err)
			}
			if !strings.EqualFold(req.Purpose, purpose) {
				t.Fatalf("purpose round-trip mismatch: got %q", req.Purpose)
			}
		})
	}
}

func TestContactBindingRequests(t *testing.T) {
	// 联系方式绑定的两个接口：contactType 大小写不敏感，contact 接受邮箱与手机号。
	for _, contactType := range []string{"EMAIL", "email", "MOBILE", "mobile"} {
		t.Run("code/"+contactType, func(t *testing.T) {
			var req ContactCodePostReq
			body := map[string]string{"contactType": contactType, "contact": "alice@example.com"}
			if err := bindJSON(t, &req, body); err != nil {
				t.Fatalf("contactType %q must bind: %v", contactType, err)
			}
		})
		t.Run("bind/"+contactType, func(t *testing.T) {
			var req ContactPutReq
			body := map[string]string{"contactType": contactType, "contact": "13800138000", "code": "123456"}
			if err := bindJSON(t, &req, body); err != nil {
				t.Fatalf("contactType %q must bind: %v", contactType, err)
			}
			if req.Code != "123456" {
				t.Fatalf("code = %q, want 123456", req.Code)
			}
		})
	}
}

func TestAdminUserRequestsAcceptOptionalContacts(t *testing.T) {
	// 管理端创建/更新用户时，email/mobile 均为可选字段；
	// 更新请求用指针区分「不更新」(缺省) 与「解绑」(空串)。
	t.Run("create without contacts", func(t *testing.T) {
		var req UserPostReq
		body := map[string]string{"username": "alice", "password": rsaCiphertextSample()}
		if err := bindJSON(t, &req, body); err != nil {
			t.Fatalf("contacts must be optional on create: %v", err)
		}
		if req.Email != "" || req.Mobile != "" {
			t.Fatalf("email = %q mobile = %q, want empty", req.Email, req.Mobile)
		}
	})

	t.Run("create with contacts", func(t *testing.T) {
		var req UserPostReq
		body := map[string]string{
			"username": "alice", "password": rsaCiphertextSample(),
			"email": "alice@example.com", "mobile": "13800138000",
		}
		if err := bindJSON(t, &req, body); err != nil {
			t.Fatalf("create with contacts must bind: %v", err)
		}
		if req.Email != "alice@example.com" || req.Mobile != "13800138000" {
			t.Fatalf("email = %q mobile = %q", req.Email, req.Mobile)
		}
	})

	t.Run("update distinguishes absent from cleared", func(t *testing.T) {
		var absent UserPutReq
		if err := bindJSON(t, &absent, map[string]string{"name": "Alice"}); err != nil {
			t.Fatalf("update without contacts must bind: %v", err)
		}
		if absent.Email != nil || absent.Mobile != nil {
			t.Fatal("absent contacts must stay nil")
		}

		var cleared UserPutReq
		if err := bindJSON(t, &cleared, map[string]string{"email": "", "mobile": ""}); err != nil {
			t.Fatalf("clearing contacts must bind: %v", err)
		}
		if cleared.Email == nil || *cleared.Email != "" || cleared.Mobile == nil || *cleared.Mobile != "" {
			t.Fatal("cleared contacts must bind to a pointer to the empty string")
		}
		if !cleared.HasUpdatableFields() {
			t.Fatal("clearing a contact must count as an updatable field")
		}
	})
}
