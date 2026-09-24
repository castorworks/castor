package fieldhandler

import (
	"fmt"
	"reflect"
	"strings"
)

// fieldHandler 实现 Handler 接口
type fieldHandler struct {
	fieldType  reflect.Type
	fieldName  string
	modelValue reflect.Value
}

func (h *fieldHandler) IsValid(v interface{}) bool {
	return reflect.TypeOf(v).ConvertibleTo(h.fieldType)
}

func (h *fieldHandler) ApplyValue(model interface{}, v interface{}) {
	reflect.ValueOf(model).Elem().FieldByName(h.fieldName).Set(
		reflect.ValueOf(v).Convert(h.fieldType),
	)
}

// GetFieldHandlers 获取模型的字段处理器映射
func GetFieldHandlers(model interface{}, skipFn SkipFieldFunc) HandlerMap {
	handlers := make(HandlerMap)
	modelValue := reflect.ValueOf(model).Elem()
	modelType := modelValue.Type()

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		fieldName := field.Name

		// 跳过不应该更新的字段
		if skipFn != nil && skipFn(fieldName) {
			continue
		}

		// 获取字段的 json 标签名
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = strings.ToLower(fieldName)
		}
		if idx := strings.Index(jsonTag, ","); idx != -1 {
			jsonTag = jsonTag[:idx]
		}

		handlers[jsonTag] = &fieldHandler{
			fieldType:  field.Type,
			fieldName:  fieldName,
			modelValue: modelValue,
		}
	}
	return handlers
}

// ApplyUpdates 应用字段更新
func ApplyUpdates(model interface{}, updates map[string]interface{}, handlers HandlerMap) error {
	for field, value := range updates {
		if handler, exists := handlers[field]; exists {
			if !handler.IsValid(value) {
				return fmt.Errorf("invalid value for field: %s", field)
			}
			handler.ApplyValue(model, value)
		}
	}
	return nil
}

// StructToMap 将结构体转换为 map
func StructToMap(req interface{}) map[string]interface{} {
	updates := make(map[string]interface{})
	reqValue := reflect.ValueOf(req).Elem()
	reqType := reqValue.Type()

	for i := 0; i < reqType.NumField(); i++ {
		field := reqType.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = strings.ToLower(field.Name)
		}
		if idx := strings.Index(jsonTag, ","); idx != -1 {
			jsonTag = jsonTag[:idx]
		}

		fieldValue := reqValue.Field(i).Interface()
		updates[jsonTag] = fieldValue
	}
	return updates
}
