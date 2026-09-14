package login_history

import (
	"github.com/castorworks/castor/internal/domain/shared"
)

// LoginHistory 登录历史实体（纯 Domain 模型，无 ORM 依赖）
type LoginHistory struct {
	shared.BaseModel
	UserID      uint   `json:"userId"`
	Username    string `json:"username"`
	IpAddr      string `json:"ipAddr"`
	UserAgent   string `json:"userAgent"`
	LoginMethod string `json:"loginMethod"`
	Success     bool   `json:"success"`
}
