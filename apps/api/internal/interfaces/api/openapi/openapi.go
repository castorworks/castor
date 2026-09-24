// Package openapi 从路由目录与 Go 类型生成 OpenAPI 3.1 文档。
//
// 文档不靠注释维护：每个路由在 api 包的目录（openapi_catalog.go）里登记一条 Operation，
// 写明请求体、响应数据的 Go 类型；这里用反射把类型转成 JSON Schema，并套上统一的
// { code, data, message } 响应信封、分页、筛选排序参数与鉴权方式。
// 路由与目录的一致性由 api 包的测试把关：新增路由不登记，测试失败。
package openapi

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Auth 访问一个接口需要的身份
type Auth int

const (
	// Public 无需登录
	Public Auth = iota
	// Session 需要登录（任何已登录用户）
	Session
	// Admin 需要登录，并由 RBAC 按"路由模板:方法"授权
	Admin
)

func (a Auth) String() string {
	switch a {
	case Session:
		return "session"
	case Admin:
		return "admin"
	}
	return "public"
}

// Param 额外的 query 参数
type Param struct {
	Name        string
	Description string
	// Type 取 string / integer / boolean，缺省为 string
	Type     string
	Required bool
	Enum     []string
}

// FormField multipart/form-data 请求里的字段
type FormField struct {
	Name        string
	Description string
	File        bool
	Required    bool
}

// Operation 一个接口的文档。
type Operation struct {
	Method string
	// Path 是 Gin 路由模板（/api/v1/admin/users/:id），生成时改写成 {id}
	Path        string
	Tag         string
	Summary     string
	Description string
	Auth        Auth

	// Body JSON 请求体的类型（传零值，如 dto.UserPostReq{}）
	Body any
	// Form multipart/form-data 请求体
	Form []FormField
	// Query 额外的 query 参数
	Query []Param

	// Data 成功响应信封里 data 的类型；nil 表示 data 为 null
	Data any
	// List 分页列表的元素类型：data 为 { total, list, page, pageSize, totalPages }，
	// 并自动带上分页、搜索、筛选、排序参数
	List any
	// Filter 分页列表可筛选排序的模型类型（与 handler.GenericGets 的 modelType 一致）
	Filter any
	// File 成功时直接下发文件（导出、下载），而不是 JSON 信封
	File []string
	// Raw 成功响应不套信封（如 /health）
	Raw any
	// Redirect 响应是 302 跳转（浏览器导航的 OIDC 登录与回调），不返回 JSON
	Redirect string
	// Stream 响应是 Server-Sent Events 事件流；值为事件说明
	Stream string
}

// Info 文档的基本信息
type Info struct {
	Title       string
	Version     string
	Description string
}

// Options 生成选项
type Options struct {
	Info Info
	// FilterFields 返回模型类型允许筛选、排序的列（handler.AllowedQueryFields）
	FilterFields func(reflect.Type) []string
	// ErrorCodes 业务错误码说明（写进 ErrorResponse.errorCode 的描述）
	ErrorCodes map[int]string
	// TagDescriptions 分组说明；每个用到的分组都必须有
	TagDescriptions map[string]string
}

// Document 是生成的 OpenAPI 文档（直接序列化为 JSON）。
type Document map[string]any

var pathParam = regexp.MustCompile(`:([A-Za-z0-9_]+)`)

// Build 生成 OpenAPI 3.1 文档。
func Build(ops []Operation, opts Options) (Document, error) {
	g := &generator{schemas: map[string]any{}, names: map[reflect.Type]string{}, opts: opts}
	paths := map[string]map[string]any{}
	tags := map[string]bool{}
	seen := map[string]bool{}

	for _, op := range ops {
		key := op.Method + " " + op.Path
		if seen[key] {
			return nil, fmt.Errorf("openapi: %s is declared twice", key)
		}
		seen[key] = true
		path := pathParam.ReplaceAllString(op.Path, "{$1}")
		if paths[path] == nil {
			paths[path] = map[string]any{}
		}
		operation, err := g.operation(op)
		if err != nil {
			return nil, fmt.Errorf("openapi: %s: %w", key, err)
		}
		paths[path][strings.ToLower(op.Method)] = operation
		tags[op.Tag] = true
	}

	tagList := make([]string, 0, len(tags))
	for tag := range tags {
		tagList = append(tagList, tag)
	}
	sort.Strings(tagList)
	tagObjects := make([]any, len(tagList))
	for i, tag := range tagList {
		description, ok := opts.TagDescriptions[tag]
		if !ok && opts.TagDescriptions != nil {
			return nil, fmt.Errorf("openapi: tag %q has no description", tag)
		}
		tagObjects[i] = map[string]any{"name": tag, "description": description}
	}

	g.schemas["ErrorResponse"] = g.errorSchema()
	return Document{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       opts.Info.Title,
			"version":     opts.Info.Version,
			"description": opts.Info.Description,
		},
		"servers": []any{map[string]any{"url": "/"}},
		"tags":    tagObjects,
		"paths":   paths,
		"components": map[string]any{
			"schemas": g.schemas,
			"securitySchemes": map[string]any{
				"cookieAuth": map[string]any{
					"type": "apiKey", "in": "cookie", "name": "jwt",
					"description": "The httpOnly `jwt` cookie set by `POST /api/v1/auth/login`. Unsafe methods (POST/PUT/PATCH/DELETE) must also send the `csrf_token` cookie's value in the `X-CSRF-Token` header.",
				},
				"bearerAuth": map[string]any{
					"type": "http", "scheme": "bearer", "bearerFormat": "JWT",
					"description": "The same token sent as `Authorization: Bearer <jwt>` (for non-browser clients). No CSRF header is needed when no `jwt` cookie is sent.",
				},
			},
			"responses": map[string]any{
				"Error": map[string]any{
					"description": "Error envelope; `message` is localized per `Accept-Language`.",
					"content":     jsonContent(ref("ErrorResponse")),
				},
			},
		},
	}, nil
}

// JSON 序列化文档（键有序、缩进两格，便于提交与审阅差异）。
func (d Document) JSON() ([]byte, error) {
	out, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

type generator struct {
	schemas map[string]any
	names   map[reflect.Type]string
	opts    Options
}

func (g *generator) operation(op Operation) (map[string]any, error) {
	if op.Tag == "" || op.Summary == "" {
		return nil, fmt.Errorf("tag and summary are required")
	}
	out := map[string]any{
		"tags":        []string{op.Tag},
		"summary":     op.Summary,
		"operationId": operationID(op),
	}
	description := op.Description
	switch op.Auth {
	case Session, Admin:
		out["security"] = []any{map[string]any{"cookieAuth": []string{}}, map[string]any{"bearerAuth": []string{}}}
		if op.Auth == Admin {
			description = strings.TrimSpace(description + "\n\nRequires the permission `" + op.Path + ":" + op.Method + "`.")
		}
	default:
		out["security"] = []any{}
	}
	if description != "" {
		out["description"] = description
	}

	params := []any{}
	for _, m := range pathParam.FindAllStringSubmatch(op.Path, -1) {
		schema := map[string]any{"type": "string"}
		if m[1] == "id" {
			schema = map[string]any{"type": "integer", "minimum": 1}
		}
		params = append(params, map[string]any{"name": m[1], "in": "path", "required": true, "schema": schema})
	}
	// 分页列表带全部列表参数；导出等"按列表条件取全量"的接口只带筛选、搜索与排序。
	if op.List != nil || op.Filter != nil {
		params = append(params, g.listParams(op, op.List != nil)...)
	}
	for _, p := range op.Query {
		schema := map[string]any{"type": defaultString(p.Type, "string")}
		if len(p.Enum) > 0 {
			schema["enum"] = p.Enum
		}
		param := map[string]any{"name": p.Name, "in": "query", "schema": schema, "required": p.Required}
		if p.Description != "" {
			param["description"] = p.Description
		}
		params = append(params, param)
	}
	if len(params) > 0 {
		out["parameters"] = params
	}

	switch {
	case op.Body != nil:
		schema, err := g.schemaOf(reflect.TypeOf(op.Body))
		if err != nil {
			return nil, err
		}
		out["requestBody"] = map[string]any{"required": true, "content": jsonContent(schema)}
	case len(op.Form) > 0:
		props := map[string]any{}
		var required []string
		for _, f := range op.Form {
			prop := map[string]any{"type": "string"}
			if f.File {
				prop["contentMediaType"] = "application/octet-stream"
			}
			if f.Description != "" {
				prop["description"] = f.Description
			}
			props[f.Name] = prop
			if f.Required {
				required = append(required, f.Name)
			}
		}
		schema := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			schema["required"] = required
		}
		out["requestBody"] = map[string]any{"required": true, "content": map[string]any{"multipart/form-data": map[string]any{"schema": schema}}}
	}

	if op.Stream != "" {
		out["responses"] = map[string]any{
			"200": map[string]any{
				"description": op.Stream,
				"content":     map[string]any{"text/event-stream": map[string]any{"schema": map[string]any{"type": "string"}}},
			},
			"4XX": map[string]any{"$ref": "#/components/responses/Error"},
		}
		return out, nil
	}
	if op.Redirect != "" {
		out["responses"] = map[string]any{"302": map[string]any{
			"description": op.Redirect,
			"headers":     map[string]any{"Location": map[string]any{"schema": map[string]any{"type": "string"}}},
		}}
		return out, nil
	}
	success, err := g.successResponse(op)
	if err != nil {
		return nil, err
	}
	responses := map[string]any{"200": success}
	if op.Raw == nil {
		responses["4XX"] = map[string]any{"$ref": "#/components/responses/Error"}
		responses["5XX"] = map[string]any{"$ref": "#/components/responses/Error"}
	}
	out["responses"] = responses
	return out, nil
}

func (g *generator) successResponse(op Operation) (map[string]any, error) {
	if len(op.File) > 0 {
		content := map[string]any{}
		for _, mediaType := range op.File {
			content[mediaType] = map[string]any{"schema": map[string]any{"type": "string", "contentMediaType": mediaType}}
		}
		return map[string]any{"description": "File download (`Content-Disposition: attachment`).", "content": content}, nil
	}
	if op.Raw != nil {
		schema, err := g.schemaOf(reflect.TypeOf(op.Raw))
		if err != nil {
			return nil, err
		}
		return map[string]any{"description": "OK", "content": jsonContent(schema)}, nil
	}

	var data any = map[string]any{"type": "null"}
	switch {
	case op.List != nil:
		item, err := g.schemaOf(reflect.TypeOf(op.List))
		if err != nil {
			return nil, err
		}
		data = map[string]any{
			"type":     "object",
			"required": []string{"total", "list", "page", "pageSize", "totalPages"},
			"properties": map[string]any{
				"total":      map[string]any{"type": "integer"},
				"list":       map[string]any{"type": "array", "items": item},
				"page":       map[string]any{"type": "integer"},
				"pageSize":   map[string]any{"type": "integer"},
				"totalPages": map[string]any{"type": "integer"},
			},
		}
	case op.Data != nil:
		schema, err := g.schemaOf(reflect.TypeOf(op.Data))
		if err != nil {
			return nil, err
		}
		data = schema
	}
	return map[string]any{
		"description": "Success envelope (`code` is 200).",
		"content": jsonContent(map[string]any{
			"type":     "object",
			"required": []string{"code", "data", "message"},
			"properties": map[string]any{
				"code":    map[string]any{"type": "integer", "const": 200},
				"message": map[string]any{"type": "string"},
				"data":    data,
			},
		}),
	}, nil
}

// listParams 分页列表共有的参数：分页、关键词搜索、字段筛选与排序（handler.GenericGets 的契约）。
func (g *generator) listParams(op Operation, paging bool) []any {
	var fields []string
	if op.Filter != nil && g.opts.FilterFields != nil {
		fields = g.opts.FilterFields(reflect.TypeOf(op.Filter))
	}
	fieldNote := ""
	if len(fields) > 0 {
		fieldNote = " Allowed fields: `" + strings.Join(fields, "`, `") + "`."
	}
	var params []any
	if paging {
		params = append(params,
			map[string]any{"name": "page", "in": "query", "schema": map[string]any{"type": "integer", "minimum": 1, "default": 1}},
			map[string]any{"name": "pageSize", "in": "query", "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": 500, "default": 10}},
		)
	}
	return append(params,
		map[string]any{"name": "order", "in": "query", "schema": map[string]any{"type": "string", "example": "created_at desc"},
			"description": "Up to 3 comma-separated `field asc|desc` items." + fieldNote},
		map[string]any{"name": "searchText", "in": "query", "schema": map[string]any{"type": "string"},
			"description": "Keyword matched (LIKE) against `searchFields`."},
		map[string]any{"name": "searchFields", "in": "query", "schema": map[string]any{"type": "string"},
			"description": "Comma-separated fields for `searchText`." + fieldNote},
		map[string]any{"name": "filters", "in": "query", "style": "form", "explode": true,
			"schema":      map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
			"description": "Field filters written as `{field}-{op}=value`, op one of `like|eq|ne|gt|gte|lt|lte|in` (`in` takes comma-separated values), e.g. `status-in=ACTIVE,LOCKED`." + fieldNote},
	)
}

func (g *generator) errorSchema() map[string]any {
	codes := make([]int, 0, len(g.opts.ErrorCodes))
	for code := range g.opts.ErrorCodes {
		codes = append(codes, code)
	}
	sort.Ints(codes)
	lines := make([]string, 0, len(codes))
	for _, code := range codes {
		lines = append(lines, fmt.Sprintf("- `%d` %s", code, g.opts.ErrorCodes[code]))
	}
	return map[string]any{
		"type":     "object",
		"required": []string{"code", "message"},
		"properties": map[string]any{
			"code":      map[string]any{"type": "integer", "description": "The HTTP status."},
			"message":   map[string]any{"type": "string", "description": "Localized message."},
			"data":      map[string]any{"description": "Details for some errors, e.g. per-row problems of a rejected import."},
			"errorCode": map[string]any{"type": "integer", "description": "Business error code:\n" + strings.Join(lines, "\n")},
		},
	}
}

var timeType = reflect.TypeOf(time.Time{})

type marshaler interface{ MarshalJSON() ([]byte, error) }

// schemaOf 把 Go 类型转成 JSON Schema；具名结构体放进 components 并返回 $ref。
func (g *generator) schemaOf(t reflect.Type) (map[string]any, error) {
	if t.Kind() == reflect.Pointer {
		inner, err := g.schemaOf(t.Elem())
		if err != nil {
			return nil, err
		}
		return nullable(inner), nil
	}
	if t == timeType {
		return map[string]any{"type": "string", "format": "date-time"}, nil
	}
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}, nil
	case reflect.Bool:
		return map[string]any{"type": "boolean"}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer", "minimum": 0}, nil
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}, nil
	case reflect.Interface:
		return map[string]any{}, nil
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return map[string]any{"type": "string", "contentEncoding": "base64"}, nil
		}
		items, err := g.schemaOf(t.Elem())
		if err != nil {
			return nil, err
		}
		return map[string]any{"type": "array", "items": items}, nil
	case reflect.Map:
		values, err := g.schemaOf(t.Elem())
		if err != nil {
			return nil, err
		}
		return map[string]any{"type": "object", "additionalProperties": values}, nil
	case reflect.Struct:
		if t.Name() == "" {
			return g.objectSchema(t)
		}
		if reflect.PointerTo(t).Implements(reflect.TypeOf((*marshaler)(nil)).Elem()) || t.Implements(reflect.TypeOf((*marshaler)(nil)).Elem()) {
			return nil, fmt.Errorf("type %s has a custom JSON encoding; describe it explicitly", t)
		}
		name := g.nameOf(t)
		if _, done := g.schemas[name]; !done {
			g.schemas[name] = map[string]any{} // 先占位，允许递归类型引用自身
			schema, err := g.objectSchema(t)
			if err != nil {
				return nil, err
			}
			g.schemas[name] = schema
		}
		return ref(name), nil
	}
	return nil, fmt.Errorf("unsupported type %s", t)
}

// nameOf 组件名取类型名；与已登记的另一个类型重名时加上包名（audit_log.Entry → AuditLogEntry）。
func (g *generator) nameOf(t reflect.Type) string {
	if name, ok := g.names[t]; ok {
		return name
	}
	name := t.Name()
	for other, used := range g.names {
		if used == name && other != t {
			name = pascal(t.PkgPath()[strings.LastIndex(t.PkgPath(), "/")+1:]) + name
			break
		}
	}
	g.names[t] = name
	return name
}

// pascal 把 audit_log、login-history 一类名字转成 AuditLog、LoginHistory
func pascal(s string) string {
	var b strings.Builder
	for _, word := range strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' || r == '.' }) {
		b.WriteString(strings.ToUpper(word[:1]) + word[1:])
	}
	return b.String()
}

func (g *generator) objectSchema(t reflect.Type) (map[string]any, error) {
	props := map[string]any{}
	var required []string
	if err := g.collectFields(t, props, &required); err != nil {
		return nil, err
	}
	schema := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		sort.Strings(required)
		schema["required"] = required
	}
	return schema, nil
}

func (g *generator) collectFields(t reflect.Type, props map[string]any, required *[]string) error {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")
		name, _, _ := strings.Cut(tag, ",")
		if name == "-" {
			continue
		}
		if field.Anonymous && name == "" {
			embedded := field.Type
			if embedded.Kind() == reflect.Pointer {
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Struct {
				if err := g.collectFields(embedded, props, required); err != nil {
					return err
				}
				continue
			}
		}
		if !field.IsExported() {
			continue
		}
		if name == "" {
			name = field.Name
		}
		schema, err := g.schemaOf(field.Type)
		if err != nil {
			return fmt.Errorf("%s.%s: %w", t.Name(), field.Name, err)
		}
		isRequired := applyBinding(schema, field.Type, field.Tag.Get("binding"))
		if isRequired {
			*required = append(*required, name)
		}
		props[name] = schema
	}
	return nil
}

// applyBinding 把 gin binding 标签里能表达的约束写进 schema，返回字段是否必填。
func applyBinding(schema map[string]any, t reflect.Type, binding string) bool {
	if binding == "" {
		return false
	}
	target := schema
	if anyOf, ok := schema["anyOf"].([]any); ok && len(anyOf) > 0 {
		if first, ok := anyOf[0].(map[string]any); ok {
			target = first
		}
	}
	base := t
	for base.Kind() == reflect.Pointer {
		base = base.Elem()
	}
	required := false
	for _, rule := range strings.Split(binding, ",") {
		key, value, _ := strings.Cut(rule, "=")
		switch key {
		case "required":
			required = true
		case "email":
			target["format"] = "email"
		case "oneof":
			target["enum"] = strings.Fields(value)
		case "min", "max", "gte", "lte", "len":
			n, err := strconv.Atoi(value)
			if err != nil {
				continue
			}
			var keys []string
			switch base.Kind() {
			case reflect.String:
				keys = map[string][]string{"min": {"minLength"}, "gte": {"minLength"}, "max": {"maxLength"}, "lte": {"maxLength"}, "len": {"minLength", "maxLength"}}[key]
			case reflect.Slice, reflect.Array:
				keys = map[string][]string{"min": {"minItems"}, "gte": {"minItems"}, "max": {"maxItems"}, "lte": {"maxItems"}, "len": {"minItems", "maxItems"}}[key]
			default:
				keys = map[string][]string{"min": {"minimum"}, "gte": {"minimum"}, "max": {"maximum"}, "lte": {"maximum"}}[key]
			}
			for _, k := range keys {
				target[k] = n
			}
		}
	}
	return required
}

func nullable(schema map[string]any) map[string]any {
	return map[string]any{"anyOf": []any{schema, map[string]any{"type": "null"}}}
}

func ref(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

func jsonContent(schema any) map[string]any {
	return map[string]any{"application/json": map[string]any{"schema": schema}}
}

func defaultString(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// operationID 由方法与路径推导：GET /api/v1/admin/users/:id → getAdminUsersById
func operationID(op Operation) string {
	var b strings.Builder
	b.WriteString(strings.ToLower(op.Method))
	for _, segment := range strings.Split(strings.TrimPrefix(op.Path, "/api/v1"), "/") {
		if segment == "" {
			continue
		}
		if strings.HasPrefix(segment, ":") {
			b.WriteString("By")
			segment = segment[1:]
		}
		b.WriteString(pascal(segment))
	}
	return b.String()
}
