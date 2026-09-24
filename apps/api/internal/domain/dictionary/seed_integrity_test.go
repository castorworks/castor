package dictionary

import (
	"regexp"
	"testing"
)

var (
	dictTypeCodePattern = regexp.MustCompile(`^[a-z0-9_]+$`)
	dictIconPattern     = regexp.MustCompile(`^[a-zA-Z0-9]*$`)
)

// TestDefaultDictionarySeedsAreConsistent 防止种子内部自相矛盾。这些问题都不会报错，
// 只会让界面静默退化：孤儿字典项永远不出现在任何下拉里，重复的 (type, value) 让
// 对账按先到先得留下其中一条，未知颜色在徽章上失效，非系统项则失去取值保护。
func TestDefaultDictionarySeedsAreConsistent(t *testing.T) {
	if len(DefaultDictTypes) == 0 || len(DefaultDictItems) == 0 {
		t.Fatal("种子为空，断言会静默通过")
	}

	types := make(map[string]bool, len(DefaultDictTypes))
	for _, dt := range DefaultDictTypes {
		if !dictTypeCodePattern.MatchString(dt.Code) {
			t.Errorf("字典类型编码 %q 只能包含小写字母、数字与下划线", dt.Code)
		}
		if types[dt.Code] {
			t.Errorf("字典类型 %q 重复声明", dt.Code)
		}
		types[dt.Code] = true
		if !dt.IsSystem {
			t.Errorf("字典类型 %q 是种子，必须 IsSystem", dt.Code)
		}
	}

	type key struct{ typeCode, value string }
	seen := make(map[key]bool, len(DefaultDictItems))
	populated := make(map[string]bool, len(types))
	for _, di := range DefaultDictItems {
		id := di.TypeCode + "/" + di.Value
		if !types[di.TypeCode] {
			t.Errorf("字典项 %s 引用了未声明的字典类型", id)
		}
		k := key{di.TypeCode, di.Value}
		if seen[k] {
			t.Errorf("字典项 %s 重复声明", id)
		}
		seen[k] = true
		populated[di.TypeCode] = true

		if di.Value == "" {
			t.Errorf("字典类型 %q 下有取值为空的字典项", di.TypeCode)
		}
		if !di.IsSystem {
			t.Errorf("字典项 %s 是种子，必须 IsSystem，否则运营可以改掉它的取值", id)
		}
		if !IsValidColor(di.Color) {
			t.Errorf("字典项 %s 的颜色 %q 不在 Colors 中", id, di.Color)
		}
		if !dictIconPattern.MatchString(di.Icon) {
			t.Errorf("字典项 %s 的图标 %q 不是合法的 Icons key", id, di.Icon)
		}
	}

	for code := range types {
		if !populated[code] {
			t.Errorf("字典类型 %q 没有任何字典项", code)
		}
	}
}

func TestIsValidColor(t *testing.T) {
	for _, color := range append([]string{""}, Colors...) {
		if !IsValidColor(color) {
			t.Errorf("IsValidColor(%q) = false, want true", color)
		}
	}
	for _, color := range []string{"#ff0000", "Red", "teal", " red"} {
		if IsValidColor(color) {
			t.Errorf("IsValidColor(%q) = true, want false", color)
		}
	}
}
