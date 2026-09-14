package dto

import (
	"github.com/castorworks/castor/internal/domain/login_history"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/jinzhu/copier"
)

//
// Request DTOs
//

// LoginHistoryPostReq 创建登录历史请求
type LoginHistoryPostReq struct {
	UserID      uint   `json:"userId"`
	Username    string `json:"username" binding:"required"`
	IpAddr      string `json:"ipAddr" binding:"required"`
	UserAgent   string `json:"userAgent"`
	LoginMethod string `json:"loginMethod" binding:"required"`
	Success     bool   `json:"success"`
}

// LoginHistoryBatchDeleteReq 批量删除登录历史请求
type LoginHistoryBatchDeleteReq struct {
	Ids []uint `json:"ids" binding:"required,min=1,max=100"`
}

//
// Response DTOs
//

// LoginHistoryResp 登录历史响应
type LoginHistoryResp struct {
	shared.BaseModel
	UserID      uint   `json:"userId"`      // 用户ID
	Username    string `json:"username"`    // 用户名
	IpAddr      string `json:"ipAddr"`      // IP地址
	UserAgent   string `json:"userAgent"`   // 用户代理
	LoginMethod string `json:"loginMethod"` // 登录方式
	Success     bool   `json:"success"`     // 是否成功
}

// FromEntity 从登录历史实体转换
func (r *LoginHistoryResp) FromEntity(entity *login_history.LoginHistory) error {
	return copier.Copy(r, entity)
}
