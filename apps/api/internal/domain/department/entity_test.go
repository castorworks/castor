package department

import (
	"slices"
	"testing"
)

func ptr(v uint) *uint { return &v }

func TestSubtree(t *testing.T) {
	t.Parallel()
	all := []Department{
		{ParentID: nil}, {ParentID: ptr(1)}, {ParentID: ptr(1)}, {ParentID: ptr(2)}, {ParentID: nil},
	}
	for i := range all {
		all[i].ID = uint(i + 1)
	}
	got := Subtree(all, 1)
	slices.Sort(got)
	if !slices.Equal(got, []uint{1, 2, 3, 4}) {
		t.Fatalf("Subtree(1) = %v", got)
	}
	if got := Subtree(all, 4); !slices.Equal(got, []uint{4}) {
		t.Fatalf("Subtree(leaf) = %v", got)
	}
	if got := Subtree(all, 99); got != nil {
		t.Fatalf("Subtree(missing) = %v, want nil", got)
	}
}
