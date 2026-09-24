package permission

import "slices"

// DataScope 是角色的数据范围：拥有该角色的用户能看到、能管理哪些用户的数据。
// 取值是与字典 role_data_scope 对应的枚举常量。
type DataScope string

const (
	DataScopeAll      DataScope = "ALL"               // 全部数据
	DataScopeDeptTree DataScope = "DEPT_AND_CHILDREN" // 本部门及下级部门
	DataScopeDept     DataScope = "DEPT"              // 仅本部门
	DataScopeSelf     DataScope = "SELF"              // 仅本人
	DataScopeCustom   DataScope = "CUSTOM"            // 指定部门（Role.DepartmentIDs）
)

// Valid 报告取值是否是已知的数据范围。
func (s DataScope) Valid() bool {
	switch s {
	case DataScopeAll, DataScopeDeptTree, DataScopeDept, DataScopeSelf, DataScopeCustom:
		return true
	}
	return false
}

// AccessScope 是一组角色的数据范围对某个用户展开后的结果：多个角色取并集，
// 任一角色为 ALL 即为全部。非全部时本人总是可见。
type AccessScope struct {
	All           bool
	UserID        uint
	DepartmentIDs []uint // 已排序、去重
}

// Contains 报告归属于 departmentID（可为空）的用户 userID 是否在范围内。
func (s AccessScope) Contains(userID uint, departmentID *uint) bool {
	if s.All || userID == s.UserID {
		return true
	}
	if departmentID == nil {
		return false
	}
	_, found := slices.BinarySearch(s.DepartmentIDs, *departmentID)
	return found
}

// ContainsDepartment 报告部门是否在范围内（ALL 范围包含所有部门）。
func (s AccessScope) ContainsDepartment(departmentID uint) bool {
	if s.All {
		return true
	}
	_, found := slices.BinarySearch(s.DepartmentIDs, departmentID)
	return found
}

// Within 报告 s 是否不超出 outer：用于防止授出或接管比自己更宽的数据范围。
// 两者的"本人"不参与比较——能不能看到目标用户本身由调用方另行检查。
func (s AccessScope) Within(outer AccessScope) bool {
	if outer.All {
		return true
	}
	if s.All {
		return false
	}
	for _, id := range s.DepartmentIDs {
		if !outer.ContainsDepartment(id) {
			return false
		}
	}
	return true
}

// NormalizeIDs 排序并去重，供 AccessScope.DepartmentIDs 使用。
func NormalizeIDs(ids []uint) []uint {
	out := slices.Clone(ids)
	slices.Sort(out)
	return slices.Compact(out)
}
