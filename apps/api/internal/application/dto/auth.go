package dto

//
// Request DTOs
//

// UserRegisterReq 用户注册请求
type UserRegisterReq struct {
	Username    string `json:"username" binding:"required,min=3,max=100"` // 用户名（必填，3-100字符）
	Name        string `json:"name" binding:"omitempty,max=100"`          // 显示名称（可选，最大100字符）
	Password    string `json:"password" binding:"required,max=512"`       // 密码（必填，RSA 密文；明文长度在解密后校验）
	CaptchaId   string `json:"captchaId" binding:"required,max=100"`      // 图形验证码 ID（必填）
	CaptchaCode string `json:"captchaCode" binding:"required,max=10"`     // 图形验证码（必填）
}

// ConfirmCodeReq 发送确认码请求
type ConfirmCodeReq struct {
	CodeType    string `json:"codeType" binding:"required,max=16"`        // 验证码类型（必填，EMAIL/MOBILE，大小写不敏感）
	Username    string `json:"username" binding:"required,min=3,max=100"` // 邮箱或手机号（必填）
	Purpose     string `json:"purpose" binding:"omitempty,max=16"`        // 验证码用途（可选，auth/reset，大小写不敏感；缺省为 auth）
	CaptchaId   string `json:"captchaId" binding:"omitempty,max=100"`     // 图形验证码 ID（可选）
	CaptchaCode string `json:"captchaCode" binding:"omitempty,max=10"`    // 图形验证码（可选）
}

// LoginReq 登录请求
type LoginReq struct {
	Username    string `json:"username" binding:"required,min=3,max=100"`   // 用户名（必填，3-100字符）
	Credential  string `json:"credential" binding:"required,min=1,max=256"` // 密码或验证码（必填）
	CaptchaId   string `json:"captchaId" binding:"omitempty,max=100"`       // 图形验证码 ID（可选）
	CaptchaCode string `json:"captchaCode" binding:"omitempty,max=10"`      // 图形验证码（可选）
}

//
// Response DTOs
//

// CaptchaResp 验证码响应
type CaptchaResp struct {
	ID  string `json:"id"`
	IMG string `json:"img"`
}
