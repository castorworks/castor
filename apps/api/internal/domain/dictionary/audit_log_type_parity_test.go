package dictionary

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"testing"
)

// auditLogTypeDictCode 是承载审计日志类型标签的字典类型编码。
const auditLogTypeDictCode = "audit_log_type"

// TestAuditLogTypesHaveDictItems 防止审计日志类型与字典项漂移：
// 每个 domain/audit_log 中声明的 AuditLogType 常量都必须有对应的
// audit_log_type 字典项，否则前端只能显示原始枚举值。
// 反向也要成立，避免字典里残留已删除的类型。
func TestAuditLogTypesHaveDictItems(t *testing.T) {
	declared := parseAuditLogTypes(t)
	if len(declared) == 0 {
		t.Fatal("no AuditLogType constants were parsed; the guardrail would silently pass")
	}

	seeded := make(map[string]int, len(DefaultDictItems))
	for _, item := range DefaultDictItems {
		if item.TypeCode != auditLogTypeDictCode {
			continue
		}
		seeded[item.Value]++
	}

	for _, value := range declared {
		switch seeded[value] {
		case 1:
			continue
		case 0:
			t.Errorf("AuditLogType %q has no %s dict item; add it to DefaultDictItems", value, auditLogTypeDictCode)
		default:
			t.Errorf("AuditLogType %q has %d duplicate %s dict items", value, seeded[value], auditLogTypeDictCode)
		}
	}

	declaredSet := make(map[string]bool, len(declared))
	for _, value := range declared {
		declaredSet[value] = true
	}
	for value := range seeded {
		if !declaredSet[value] {
			t.Errorf("%s dict item %q has no AuditLogType constant; remove the stale item", auditLogTypeDictCode, value)
		}
	}
}

// parseAuditLogTypes 解析 domain/audit_log/entity.go，取出所有 AuditLogType
// 常量的字面量取值。用源码解析而非 reflect，是因为 domain 之间不互相导入。
func parseAuditLogTypes(t *testing.T) []string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate dictionary test source")
	}
	entityPath := filepath.Join(filepath.Dir(filepath.Dir(filename)), "audit_log", "entity.go")

	file, err := parser.ParseFile(token.NewFileSet(), entityPath, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	values := make([]string, 0)
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			typeIdent, ok := valueSpec.Type.(*ast.Ident)
			if !ok || typeIdent.Name != "AuditLogType" {
				continue
			}
			for _, expr := range valueSpec.Values {
				literal, ok := expr.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				value, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatal(err)
				}
				values = append(values, value)
			}
		}
	}

	sort.Strings(values)
	return values
}
