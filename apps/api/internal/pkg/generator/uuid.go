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
