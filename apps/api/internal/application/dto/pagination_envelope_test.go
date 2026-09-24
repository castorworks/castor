package dto

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// paginationMarkers 是分页信封的判定特征：`total` 是页面总数，
// 其余三个是分页元数据。同时出现就说明这个 DTO 在自己拼分页响应。
const totalMarker = "total"

var paginationMarkers = []string{"page", "pagesize", "totalpages"}

// TestNoHandRolledPaginationEnvelope 防止列表接口绕开统一分页契约：
// 分页响应只能由 response.SuccessListPaged 产出 `{total, list, page, pageSize, totalPages}`，
// DTO 层不得声明自己的分页信封：自造的信封（如返回 `items`、没有 totalPages、不经过 GenericGets）
// 会丢掉排序与通用筛选，还让前端多出一套只为它存在的解析分支。
func TestNoHandRolledPaginationEnvelope(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, 0)
	if err != nil {
		t.Fatalf("parse dto package: %v", err)
	}

	var offenders []string
	for _, pkg := range pkgs {
		for path, file := range pkg.Files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			ast.Inspect(file, func(n ast.Node) bool {
				spec, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				structType, ok := spec.Type.(*ast.StructType)
				if !ok {
					return true
				}
				if names := paginationFieldsOf(structType); len(names) > 0 {
					offenders = append(offenders,
						filepath.Base(path)+": "+spec.Name.Name+" declares "+strings.Join(names, ", "))
				}
				return true
			})
		}
	}

	sort.Strings(offenders)
	if len(offenders) > 0 {
		t.Errorf("DTO 不得自建分页信封，改用 response.SuccessListPaged（见 AGENTS.md 第 7 节）：\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

// paginationFieldsOf 返回该结构体命中的分页字段；未命中时返回 nil。
func paginationFieldsOf(structType *ast.StructType) []string {
	seen := make(map[string]string)
	for _, field := range structType.Fields.List {
		name := jsonFieldName(field)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if key == totalMarker {
			seen[totalMarker] = name
			continue
		}
		for _, marker := range paginationMarkers {
			if key == marker {
				seen[marker] = name
			}
		}
	}
	// 单独一个 total 可能是业务统计（例如 asset 的 totalCount），
	// 只有同时带上分页元数据才能确定这是一个分页信封。
	if _, ok := seen[totalMarker]; !ok {
		return nil
	}
	var hit []string
	for _, marker := range append([]string{totalMarker}, paginationMarkers...) {
		if name, ok := seen[marker]; ok {
			hit = append(hit, name)
		}
	}
	if len(hit) < 2 {
		return nil
	}
	return hit
}

// jsonFieldName 取字段的 json 名；匿名字段与 `json:"-"` 返回空串。
func jsonFieldName(field *ast.Field) string {
	if len(field.Names) == 0 || field.Tag == nil {
		return ""
	}
	raw, err := strconv.Unquote(field.Tag.Value)
	if err != nil {
		return ""
	}
	name := strings.Split(reflect.StructTag(raw).Get("json"), ",")[0]
	if name == "-" {
		return ""
	}
	return name
}
