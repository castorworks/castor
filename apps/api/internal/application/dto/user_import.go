package dto

// 导入文件的字段标识：模板表头写成"用户名* (username)"，括号里就是这些标识，
// 与界面语言无关，任何语言的模板都能导入。
const (
	UserImportFieldUsername       = "username"
	UserImportFieldName           = "name"
	UserImportFieldEmail          = "email"
	UserImportFieldMobile         = "mobile"
	UserImportFieldDepartmentCode = "departmentCode"
)

// UserImportRow 导入文件里的一行，已按表头映射到字段。Line 是它在文件里的行号。
type UserImportRow struct {
	Line           int
	Username       string
	Name           string
	Email          string
	Mobile         string
	DepartmentCode string
}

// UserImportReq 批量导入用户：所有用户共用一个初始密码（RSA 密文），
// 首次登录时必须修改（凭证创建即过期）。
type UserImportReq struct {
	Password string
	Rows     []UserImportRow
}

// UserImportError 某一行某个字段的问题。Code 是 i18n key，Message 由接口层按请求语言填写。
type UserImportError struct {
	Line    int    `json:"line"`
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// UserImportResult 导入结果。有错误时要么什么都没导入（校验失败），
// 要么只导入了出错行之前的行（逐行创建时遇到并发冲突）。
type UserImportResult struct {
	Created    int               `json:"created"`
	Errors     []UserImportError `json:"errors"`
	CreatedIDs []uint            `json:"-"`
}
