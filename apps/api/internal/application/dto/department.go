package dto

import "github.com/castorworks/castor/internal/domain/department"

// DepartmentRequest 创建或更新部门请求
type DepartmentRequest struct {
	ParentID  *uint  `json:"parentId"`
	Code      string `json:"code" binding:"required,max=64"`
	Name      string `json:"name" binding:"required,max=100"`
	SortOrder int    `json:"sortOrder"`
	IsEnabled bool   `json:"isEnabled"`
}

func (r DepartmentRequest) ToEntity() department.Department {
	return department.Department{ParentID: r.ParentID, Code: r.Code, Name: r.Name, SortOrder: r.SortOrder, IsEnabled: r.IsEnabled}
}

// DepartmentResp 部门响应。InScope 表示部门在调用者的数据范围内：界面据此决定能否选择或编辑。
type DepartmentResp struct {
	department.Department
	MemberCount int64 `json:"memberCount"`
	InScope     bool  `json:"inScope"`
}
