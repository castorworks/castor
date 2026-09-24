package query

// UserScope 把数据范围转成查询条件：column 是记录所属用户的 ID 列。
//   - all 为 true 时不加限制，返回 nil；
//   - 否则允许本人（selfID）以及归属于 departmentIDs 中任一部门的用户。
//
// 对 users 表本身，column 传 "id"，部门直接比较 department_id；
// 对其它按用户归属的表（登录日志、会话、审计），通过子查询取部门内的用户。
// column 只能是代码里的常量列名，不得来自请求。
func UserScope(column string, all bool, selfID uint, departmentIDs []uint) *Option {
	if all {
		return nil
	}
	if len(departmentIDs) == 0 {
		return NewOption(column+" = ?", selfID)
	}
	if column == "id" {
		return NewOption("(id = ? OR department_id IN ?)", selfID, departmentIDs)
	}
	return NewOption("("+column+" = ? OR "+column+" IN (SELECT id FROM users WHERE department_id IN ?))", selfID, departmentIDs)
}
