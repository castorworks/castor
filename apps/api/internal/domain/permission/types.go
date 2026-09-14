package permission

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"slices"
)

// ============================================
// 数据库类型定义
// ============================================

// StringSlice 是一个可以存储到数据库的字符串切片类型
type StringSlice []string

// Scan 实现 sql.Scanner 接口
func (s *StringSlice) Scan(value any) error {
	if value == nil {
		*s = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("invalid type for StringSlice")
	}

	if len(bytes) == 0 {
		*s = nil
		return nil
	}

	return json.Unmarshal(bytes, s)
}

// Value 实现 driver.Valuer 接口
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Contains 检查切片是否包含指定元素
func (s StringSlice) Contains(item string) bool {
	return slices.Contains(s, item)
}
