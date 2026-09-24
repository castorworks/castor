package validator

import (
	"regexp"
	"strings"
	"unicode"
)

// IsAlphaNumeric 检查字符串是否只包含字母、数字、下划线、点和连字符
func IsAlphaNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '.' && r != '-' {
			return false
		}
	}
	return true
}

// IsValidFilename 检查是否是有效的文件名
// 允许：字母、数字、中文、空格、下划线、点、连字符、括号等常见字符
// 禁止：路径分隔符、特殊控制字符
func IsValidFilename(s string) bool {
	if s == "" {
		return false
	}

	// 禁止的字符（文件系统不允许或有安全风险）
	forbidden := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\x00"}
	for _, f := range forbidden {
		if strings.Contains(s, f) {
			return false
		}
	}

	// 禁止以点开头（隐藏文件）或以空格开头/结尾
	if strings.HasPrefix(s, ".") || strings.HasPrefix(s, " ") || strings.HasSuffix(s, " ") {
		return false
	}

	// 禁止特殊的 Windows 保留名称
	reserved := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
	baseName := strings.ToUpper(strings.TrimSuffix(s, "."+getExtension(s)))
	for _, r := range reserved {
		if baseName == r {
			return false
		}
	}

	return true
}

// getExtension 获取文件扩展名
func getExtension(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx == -1 {
		return ""
	}
	return filename[idx+1:]
}

// IsMobile 检查是否是有效的手机号
func IsMobile(mobile string) bool {
	pattern := `^1[3-9]\d{9}$`
	matched, _ := regexp.MatchString(pattern, mobile)
	return matched
}

// IsEmail 检查是否是有效的邮箱
func IsEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}
