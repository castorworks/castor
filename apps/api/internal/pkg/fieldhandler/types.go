package fieldhandler

// Handler 定义字段处理器接口
type Handler interface {
	IsValid(interface{}) bool
	ApplyValue(interface{}, interface{})
}

// HandlerMap 字段处理器映射
type HandlerMap map[string]Handler

// SkipFieldFunc 定义跳过字段的判断函数类型
type SkipFieldFunc func(string) bool

// DefaultSkipFields 默认的需要跳过的字段（审计相关字段不应被手动更新）
var DefaultSkipFields = map[string]bool{
	"ID":        true,
	"CreatedAt": true,
	"CreatedBy": true,
	"UpdatedAt": true,
	"UpdatedBy": true,
}
