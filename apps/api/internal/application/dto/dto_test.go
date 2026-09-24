package dto

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

func validateStruct(s interface{}) []string {
	v := validator.New()
	v.SetTagName("binding")
	err := v.Struct(s)
	if err == nil {
		return nil
	}
	var fields []string
	for _, e := range err.(validator.ValidationErrors) {
		fields = append(fields, e.Field())
	}
	return fields
}

func TestUserRegisterReq_Validation(t *testing.T) {
	t.Run("valid request passes", func(t *testing.T) {
		req := &UserRegisterReq{
			Username:    "testuser",
			Password:    "password123",
			CaptchaId:   "captcha-id",
			CaptchaCode: "1234",
		}
		errs := validateStruct(req)
		if errs != nil {
			t.Errorf("expected no validation errors, got: %v", errs)
		}
	})

	t.Run("empty username fails", func(t *testing.T) {
		req := &UserRegisterReq{
			Password:    "password123",
			CaptchaId:   "captcha-id",
			CaptchaCode: "1234",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for empty username")
		}
	})

	t.Run("username too short fails", func(t *testing.T) {
		req := &UserRegisterReq{
			Username:    "ab",
			Password:    "password123",
			CaptchaId:   "captcha-id",
			CaptchaCode: "1234",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for short username")
		}
	})

	t.Run("username too long fails", func(t *testing.T) {
		req := &UserRegisterReq{
			Username:    string(make([]byte, 101)),
			Password:    "password123",
			CaptchaId:   "captcha-id",
			CaptchaCode: "1234",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for long username")
		}
	})

	t.Run("empty password fails", func(t *testing.T) {
		req := &UserRegisterReq{
			Username:    "testuser",
			CaptchaId:   "captcha-id",
			CaptchaCode: "1234",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for empty password")
		}
	})

	// 密码字段承载 RSA 密文，绑定层不再校验明文长度：明文最小长度由
	// SettingHelper.ValidatePasswordLength 在解密后按系统配置校验
	// （见 service.TestAuthService_ValidatePasswordLength）。
	// 绑定层只兜住明显异常的超长输入。
	t.Run("oversized password fails", func(t *testing.T) {
		req := &UserRegisterReq{
			Username:    "testuser",
			Password:    strings.Repeat("A", 513),
			CaptchaId:   "captcha-id",
			CaptchaCode: "1234",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for oversized password")
		}
	})

	t.Run("empty captcha id fails", func(t *testing.T) {
		req := &UserRegisterReq{
			Username:    "testuser",
			Password:    "password123",
			CaptchaCode: "1234",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for empty captcha id")
		}
	})

	t.Run("empty captcha code fails", func(t *testing.T) {
		req := &UserRegisterReq{
			Username:  "testuser",
			Password:  "password123",
			CaptchaId: "captcha-id",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for empty captcha code")
		}
	})
}

func TestLoginReq_Validation(t *testing.T) {
	t.Run("valid request passes", func(t *testing.T) {
		req := &LoginReq{
			Username:   "testuser",
			Credential: "password123",
		}
		errs := validateStruct(req)
		if errs != nil {
			t.Errorf("expected no validation errors, got: %v", errs)
		}
	})

	t.Run("empty username fails", func(t *testing.T) {
		req := &LoginReq{
			Credential: "password123",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for empty username")
		}
	})

	t.Run("empty credential fails", func(t *testing.T) {
		req := &LoginReq{
			Username: "testuser",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for empty credential")
		}
	})

	t.Run("username too short fails", func(t *testing.T) {
		req := &LoginReq{
			Username:   "ab",
			Credential: "password",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for short username")
		}
	})
}

func TestConfirmCodeReq_Validation(t *testing.T) {
	t.Run("valid EMAIL request passes", func(t *testing.T) {
		req := &ConfirmCodeReq{
			CodeType: "EMAIL",
			Username: "user@example.com",
		}
		errs := validateStruct(req)
		if errs != nil {
			t.Errorf("expected no validation errors, got: %v", errs)
		}
	})

	t.Run("valid MOBILE request passes", func(t *testing.T) {
		req := &ConfirmCodeReq{
			CodeType: "MOBILE",
			Username: "13812345678",
		}
		errs := validateStruct(req)
		if errs != nil {
			t.Errorf("expected no validation errors, got: %v", errs)
		}
	})

	// codeType 的取值合法性由服务层判定（大小写不敏感，未知值返回
	// ErrInvalidCodeType，见 service.TestAuthService_PostCode_CodeTypeIsCaseInsensitive）。
	// 绑定层刻意不用 oneof 限定大小写：与服务端常量不一致时 /auth/code 会完全不可用。
	t.Run("lowercase code type binds", func(t *testing.T) {
		req := &ConfirmCodeReq{
			CodeType: "email",
			Username: "user@example.com",
		}
		if errs := validateStruct(req); errs != nil {
			t.Errorf("lowercase codeType must bind, got: %v", errs)
		}
	})

	t.Run("oversized code type fails", func(t *testing.T) {
		req := &ConfirmCodeReq{
			CodeType: strings.Repeat("E", 17),
			Username: "user@example.com",
		}
		if errs := validateStruct(req); errs == nil {
			t.Error("expected validation error for oversized code type")
		}
	})

	t.Run("empty code type fails", func(t *testing.T) {
		req := &ConfirmCodeReq{
			Username: "user",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for empty code type")
		}
	})

	t.Run("empty username fails", func(t *testing.T) {
		req := &ConfirmCodeReq{
			CodeType: "EMAIL",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for empty username")
		}
	})
}

func TestDictTypePostReq_Validation(t *testing.T) {
	t.Run("valid request passes", func(t *testing.T) {
		req := &DictTypePostReq{
			Code: "color",
			Name: I18nTextReq{En: "Color", Zh: "颜色", Ja: "色", Ko: "색상"},
		}
		errs := validateStruct(req)
		if errs != nil {
			t.Errorf("expected no validation errors, got: %v", errs)
		}
	})

	t.Run("empty code fails", func(t *testing.T) {
		req := &DictTypePostReq{
			Name: I18nTextReq{En: "Color", Zh: "颜色", Ja: "色", Ko: "색상"},
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for empty code")
		}
	})
}

func TestDictItemPostReq_Validation(t *testing.T) {
	t.Run("valid request passes", func(t *testing.T) {
		req := &DictItemPostReq{
			TypeCode: "gender",
			Label:    I18nTextReq{En: "Male", Zh: "男", Ja: "男性", Ko: "남성"},
			Value:    "MALE",
		}
		errs := validateStruct(req)
		if errs != nil {
			t.Errorf("expected no validation errors, got: %v", errs)
		}
	})

	t.Run("empty type code fails", func(t *testing.T) {
		req := &DictItemPostReq{
			Label: I18nTextReq{En: "Male", Zh: "男", Ja: "男性", Ko: "남성"},
			Value: "MALE",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for empty type code")
		}
	})
}

func TestSettingPostReq_Validation(t *testing.T) {
	t.Run("valid request passes", func(t *testing.T) {
		req := &SettingPostReq{
			Key:   "test.key",
			Value: "value",
			Type:  "STRING",
			Name:  "Test setting",
		}
		errs := validateStruct(req)
		if errs != nil {
			t.Errorf("expected no validation errors, got: %v", errs)
		}
	})

	t.Run("empty key fails", func(t *testing.T) {
		req := &SettingPostReq{
			Value: "value",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for empty key")
		}
	})
}

func TestNotificationPostReq_Validation(t *testing.T) {
	t.Run("valid request passes", func(t *testing.T) {
		req := &NotificationPostReq{
			Title:   "Notification",
			Content: "Content",
		}
		errs := validateStruct(req)
		if errs != nil {
			t.Errorf("expected no validation errors, got: %v", errs)
		}
	})

	t.Run("empty title fails", func(t *testing.T) {
		req := &NotificationPostReq{
			Content: "Content",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for empty title")
		}
	})
}

func TestLoginHistoryPostReq_Validation(t *testing.T) {
	t.Run("valid request passes", func(t *testing.T) {
		req := &LoginHistoryPostReq{
			Username:    "user",
			IpAddr:      "1.2.3.4",
			LoginMethod: "password",
		}
		errs := validateStruct(req)
		if errs != nil {
			t.Errorf("expected no validation errors, got: %v", errs)
		}
	})

	t.Run("missing required fields fails", func(t *testing.T) {
		req := &LoginHistoryPostReq{}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation errors for empty required fields")
		}
	})

	t.Run("missing username fails", func(t *testing.T) {
		req := &LoginHistoryPostReq{
			IpAddr:      "1.2.3.4",
			LoginMethod: "password",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for missing username")
		}
	})

	t.Run("missing ip fails", func(t *testing.T) {
		req := &LoginHistoryPostReq{
			Username:    "user",
			LoginMethod: "password",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for missing ipAddr")
		}
	})

	t.Run("missing login method fails", func(t *testing.T) {
		req := &LoginHistoryPostReq{
			Username: "user",
			IpAddr:   "1.2.3.4",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for missing loginMethod")
		}
	})
}

func TestAssetUploadReq_NoRequiredFields(t *testing.T) {
	t.Run("all fields optional", func(t *testing.T) {
		req := &AssetUploadReq{}
		errs := validateStruct(req)
		if errs != nil {
			t.Errorf("expected no validation errors, got: %v", errs)
		}
	})
}

func TestAssetUpdateReq_PointerFields(t *testing.T) {
	t.Run("all fields optional (pointers)", func(t *testing.T) {
		req := &AssetUpdateReq{}
		errs := validateStruct(req)
		if errs != nil {
			t.Errorf("expected no validation errors, got: %v", errs)
		}
	})
}

// 四语言缺一就会让那种语言的用户看到别的语言，所以这是必填校验而不是建议。
func TestI18nTextReq_RequiresEveryLanguage(t *testing.T) {
	full := I18nTextReq{En: "Male", Zh: "男", Ja: "男性", Ko: "남성"}
	for _, missing := range []string{"en", "zh", "ja", "ko"} {
		partial := full
		switch missing {
		case "en":
			partial.En = ""
		case "zh":
			partial.Zh = ""
		case "ja":
			partial.Ja = ""
		case "ko":
			partial.Ko = ""
		}
		req := &DictItemPostReq{TypeCode: "gender", Label: partial, Value: "MALE"}
		if errs := validateStruct(req); errs == nil {
			t.Errorf("缺少 %s 时应当校验失败", missing)
		}
	}
}
