package dictionary

import "testing"

// TestDefaultDictionaryTextIsComplete 防止字典种子退回单语言。
//
// 字典标签会渲染成徽章、下拉选项与审计日志类型名，覆盖面很广；
// 少填一种语言，那种语言的用户看到的就是另一种语言的字面量——
// 这正是这些标签长期只有英文时的样子。
func TestDefaultDictionaryTextIsComplete(t *testing.T) {
	if len(DefaultDictTypes) == 0 || len(DefaultDictItems) == 0 {
		t.Fatal("种子为空，断言会静默通过")
	}

	for _, dt := range DefaultDictTypes {
		if !dt.Name.IsComplete() {
			t.Errorf("字典类型 %q 的名称四语言不全: %+v", dt.Code, dt.Name)
		}
	}
	for _, di := range DefaultDictItems {
		if !di.Label.IsComplete() {
			t.Errorf("字典项 %s/%s 的标签四语言不全: %+v", di.TypeCode, di.Value, di.Label)
		}
	}
}
