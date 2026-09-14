package dto

import (
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

	t.Run("password too short fails", func(t *testing.T) {
		req := &UserRegisterReq{
			Username:    "testuser",
			Password:    "12345",
			CaptchaId:   "captcha-id",
			CaptchaCode: "1234",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for short password")
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

	t.Run("invalid code type fails", func(t *testing.T) {
		req := &ConfirmCodeReq{
			CodeType: "SMS",
			Username: "user",
		}
		errs := validateStruct(req)
		if errs == nil {
			t.Error("expected validation error for invalid code type")
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
			Name: "Color",
		}
		errs := validateStruct(req)
		if errs != nil {
			t.Errorf("expected no validation errors, got: %v", errs)
		}
	})

	t.Run("empty code fails", func(t *testing.T) {
		req := &DictTypePostReq{
			Name: "Color",
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
			Label:    "Male",
			Value:    "MALE",
		}
		errs := validateStruct(req)
		if errs != nil {
			t.Errorf("expected no validation errors, got: %v", errs)
		}
	})

	t.Run("empty type code fails", func(t *testing.T) {
		req := &DictItemPostReq{
			Label: "Male",
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
