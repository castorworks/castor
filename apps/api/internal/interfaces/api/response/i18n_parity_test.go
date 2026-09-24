package response

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"testing"

	"github.com/BurntSushi/toml"
)

var i18nLocales = []string{"zh", "en", "ja", "ko"}

// TestI18nLocalesHaveIdenticalKeys 保证四种语言的翻译文件 key 集合完全一致。
func TestI18nLocalesHaveIdenticalKeys(t *testing.T) {
	base := loadLocaleKeys(t, i18nLocales[0])
	for _, locale := range i18nLocales[1:] {
		keys := loadLocaleKeys(t, locale)
		if missing := diffKeys(base, keys); len(missing) > 0 {
			t.Errorf("%s.toml missing keys present in %s.toml: %v", locale, i18nLocales[0], missing)
		}
		if extra := diffKeys(keys, base); len(extra) > 0 {
			t.Errorf("%s.toml has keys absent from %s.toml: %v", locale, i18nLocales[0], extra)
		}
	}
}

// TestMessageConstantsExistInAllLocales 保证 messages.go 中声明的每个 i18n key 都有翻译。
func TestMessageConstantsExistInAllLocales(t *testing.T) {
	constants := messageConstants(t)
	for _, locale := range i18nLocales {
		keys := loadLocaleKeys(t, locale)
		var missing []string
		for _, key := range constants {
			if !keys[key] {
				missing = append(missing, key)
			}
		}
		if len(missing) > 0 {
			t.Errorf("%s.toml missing keys declared in messages.go: %v", locale, missing)
		}
	}
}

func i18nDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "configs", "i18n")
}

func loadLocaleKeys(t *testing.T, locale string) map[string]bool {
	t.Helper()
	var doc map[string]any
	path := filepath.Join(i18nDir(t), locale+".toml")
	if _, err := toml.DecodeFile(path, &doc); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	keys := make(map[string]bool)
	flattenKeys("", doc, keys)
	return keys
}

func flattenKeys(prefix string, node map[string]any, out map[string]bool) {
	for k, v := range node {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if child, ok := v.(map[string]any); ok {
			flattenKeys(key, child, out)
			continue
		}
		out[key] = true
	}
}

func messageConstants(t *testing.T) []string {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(filename), "messages.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	var values []string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			for _, v := range spec.(*ast.ValueSpec).Values {
				lit, ok := v.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				s, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("unquote %s: %v", lit.Value, err)
				}
				values = append(values, s)
			}
		}
	}
	if len(values) == 0 {
		t.Fatal("no i18n constants found in messages.go")
	}
	return values
}

func diffKeys(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
