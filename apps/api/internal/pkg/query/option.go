package query

// Option 查询选项
type Option struct {
	Condition string
	Args      []any
}

// NewOption 创建查询选项
func NewOption(condition string, args ...any) *Option {
	return &Option{Condition: condition, Args: args}
}
