package permission

import "testing"

func deptPtr(v uint) *uint { return &v }

func TestAccessScope(t *testing.T) {
	t.Parallel()
	all := AccessScope{All: true}
	dept := AccessScope{UserID: 7, DepartmentIDs: NormalizeIDs([]uint{5, 3, 3})}
	self := AccessScope{UserID: 7}

	if !all.Contains(1, nil) || !all.ContainsDepartment(99) {
		t.Fatal("ALL must contain everyone")
	}
	if !dept.Contains(7, nil) || !self.Contains(7, deptPtr(9)) {
		t.Fatal("the caller is always in their own scope")
	}
	if !dept.Contains(8, deptPtr(3)) || dept.Contains(8, deptPtr(4)) || dept.Contains(8, nil) {
		t.Fatal("department membership decides visibility of other users")
	}
	if self.Contains(8, deptPtr(3)) {
		t.Fatal("SELF must not see other users")
	}

	if !dept.Within(all) || all.Within(dept) {
		t.Fatal("ALL is wider than any department set")
	}
	if !self.Within(dept) || !(AccessScope{DepartmentIDs: []uint{3}}).Within(dept) {
		t.Fatal("a subset of departments is within")
	}
	if (AccessScope{DepartmentIDs: []uint{3, 4}}).Within(dept) {
		t.Fatal("an extra department is not within")
	}
}

func TestDataScopeValid(t *testing.T) {
	t.Parallel()
	for _, s := range []DataScope{DataScopeAll, DataScopeDeptTree, DataScopeDept, DataScopeSelf, DataScopeCustom} {
		if !s.Valid() {
			t.Fatalf("%s must be valid", s)
		}
	}
	if DataScope("EVERYTHING").Valid() {
		t.Fatal("unknown scope must be invalid")
	}
}
