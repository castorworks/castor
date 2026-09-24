package generator

import (
	"strings"

	"github.com/google/uuid"
)

// GenerateShortUUIDString 生成短 UUID 字符串（8位）
func GenerateShortUUIDString() string {
	str := uuid.NewString()
	parts := strings.Split(str, "-")
	return parts[0]
}

// GenerateCustomUUIDNoConfusingChars 生成不包含易混淆字符的 UUID
func GenerateCustomUUIDNoConfusingChars(length int) string {
	if length <= 0 {
		length = 8
	}
	if length > 32 {
		length = 32
	}

	confusingChars := "01oOiIlL"
	for {
		uuidStr := strings.ReplaceAll(uuid.NewString(), "-", "")
		for _, char := range confusingChars {
			uuidStr = strings.ReplaceAll(uuidStr, string(char), "")
		}
		if len(uuidStr) >= length {
			return uuidStr[:length]
		}
	}
}

// GenerateObjectKey 生成用于对象存储键的高熵标识（128 位，去连字符的十六进制）。
// 存储键必须使用完整熵，8 位短 UUID 在资产量增大时碰撞概率过高，碰撞会导致
// 覆盖或误删对象，因此不要用 GenerateShortUUIDString 作对象键。
func GenerateObjectKey() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}
