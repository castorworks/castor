package query

import (
	"reflect"
	"testing"
)

func TestUserScope(t *testing.T) {
	t.Parallel()
	if UserScope("user_id", true, 7, []uint{1}) != nil {
		t.Fatal("ALL scope must not filter")
	}
	cases := []struct {
		column string
		depts  []uint
		want   Option
	}{
		{"id", nil, Option{Condition: "id = ?", Args: []any{uint(7)}}},
		{"id", []uint{1, 2}, Option{Condition: "(id = ? OR department_id IN ?)", Args: []any{uint(7), []uint{1, 2}}}},
		{"user_id", []uint{1}, Option{Condition: "(user_id = ? OR user_id IN (SELECT id FROM users WHERE department_id IN ?))", Args: []any{uint(7), []uint{1}}}},
		{"operator_id", nil, Option{Condition: "operator_id = ?", Args: []any{uint(7)}}},
	}
	for _, tc := range cases {
		if got := UserScope(tc.column, false, 7, tc.depts); !reflect.DeepEqual(*got, tc.want) {
			t.Fatalf("UserScope(%s, %v) = %+v, want %+v", tc.column, tc.depts, *got, tc.want)
		}
	}
}
