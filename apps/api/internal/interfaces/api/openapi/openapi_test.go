package openapi

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/interfaces/api/openapi/testdata/other"
)

type base struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

type Widget struct {
	base
	Name     string            `json:"name" binding:"required,min=3,max=100"`
	Kind     string            `json:"kind" binding:"omitempty,oneof=A B"`
	Email    string            `json:"email" binding:"omitempty,email"`
	Parent   *uint             `json:"parentId"`
	Tags     []string          `json:"tags" binding:"max=5"`
	Labels   map[string]string `json:"labels"`
	Children []Widget          `json:"children"`
	Secret   string            `json:"-"`
	Raw      []byte            `json:"raw"`
	Count    int               `json:"count" binding:"gte=1,lte=9"`
	hidden   string
}

type WidgetPair struct {
	Mine   Widget       `json:"mine"`
	Theirs other.Widget `json:"theirs"`
}

func build(t *testing.T, ops ...Operation) map[string]any {
	t.Helper()
	doc, err := Build(ops, Options{
		Info:         Info{Title: "Test", Version: "v1"},
		FilterFields: func(reflect.Type) []string { return []string{"name", "created_at"} },
		ErrorCodes:   map[int]string{1000: "Bad request"},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := doc.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func dig(t *testing.T, v any, path ...string) any {
	t.Helper()
	for _, key := range path {
		m, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("%v: not an object at %q", path, key)
		}
		v = m[key]
	}
	return v
}

func TestSchemaFromGoTypes(t *testing.T) {
	doc := build(t, Operation{Method: "POST", Path: "/api/v1/admin/widgets", Tag: "Widgets", Summary: "Create", Auth: Admin, Body: Widget{}, Data: WidgetPair{}})
	widget := dig(t, doc, "components", "schemas", "Widget").(map[string]any)
	props := widget["properties"].(map[string]any)

	for _, name := range []string{"id", "createdAt", "name", "parentId", "children"} {
		if props[name] == nil {
			t.Errorf("property %q missing (embedded fields must be flattened)", name)
		}
	}
	for _, name := range []string{"Secret", "-", "hidden", "base"} {
		if props[name] != nil {
			t.Errorf("property %q must not be documented", name)
		}
	}
	if got := dig(t, props, "createdAt", "format"); got != "date-time" {
		t.Errorf("time format = %v", got)
	}
	name := props["name"].(map[string]any)
	if name["minLength"] != 3.0 || name["maxLength"] != 100.0 {
		t.Errorf("name constraints = %v", name)
	}
	if got := props["kind"].(map[string]any)["enum"]; !reflect.DeepEqual(got, []any{"A", "B"}) {
		t.Errorf("oneof → enum = %v", got)
	}
	if props["email"].(map[string]any)["format"] != "email" {
		t.Errorf("email format missing: %v", props["email"])
	}
	if props["tags"].(map[string]any)["maxItems"] != 5.0 {
		t.Errorf("slice max → maxItems: %v", props["tags"])
	}
	if c := props["count"].(map[string]any); c["minimum"] != 1.0 || c["maximum"] != 9.0 {
		t.Errorf("number bounds = %v", c)
	}
	if props["raw"].(map[string]any)["contentEncoding"] != "base64" {
		t.Errorf("[]byte should be a base64 string: %v", props["raw"])
	}
	parent := props["parentId"].(map[string]any)["anyOf"].([]any)
	if parent[1].(map[string]any)["type"] != "null" {
		t.Errorf("pointer must be nullable: %v", parent)
	}
	if ref := dig(t, props, "children", "items", "$ref"); ref != "#/components/schemas/Widget" {
		t.Errorf("recursive type must reference itself, got %v", ref)
	}
	if got := widget["required"]; !reflect.DeepEqual(got, []any{"name"}) {
		t.Errorf("required = %v", got)
	}

	pair := dig(t, doc, "components", "schemas", "WidgetPair", "properties").(map[string]any)
	if dig(t, pair, "theirs", "$ref") != "#/components/schemas/OtherWidget" {
		t.Errorf("same-named type from another package must be prefixed: %v", pair["theirs"])
	}
}

func TestOperationShape(t *testing.T) {
	doc := build(t,
		Operation{Method: "GET", Path: "/api/v1/admin/widgets", Tag: "Widgets", Summary: "List", Auth: Admin, List: Widget{}, Filter: Widget{}},
		Operation{Method: "DELETE", Path: "/api/v1/admin/widgets/:id", Tag: "Widgets", Summary: "Delete", Auth: Admin},
		Operation{Method: "GET", Path: "/api/v1/public/widgets/:slug/file", Tag: "Widgets", Summary: "Download", File: []string{"text/csv"}},
		Operation{Method: "GET", Path: "/api/v1/admin/widgets/export", Tag: "Widgets", Summary: "Export", Auth: Admin, File: []string{"text/csv"}, Filter: Widget{}},
		Operation{Method: "GET", Path: "/sso/start", Tag: "Auth", Summary: "Start", Redirect: "Goes to the identity provider."},
		Operation{Method: "GET", Path: "/events", Tag: "Auth", Summary: "Events", Auth: Session, Stream: "Event stream."},
		Operation{Method: "GET", Path: "/health", Tag: "System", Summary: "Health", Raw: struct {
			Status string `json:"status"`
		}{}},
	)
	paths := doc["paths"].(map[string]any)

	listOp := dig(t, paths, "/api/v1/admin/widgets", "get").(map[string]any)
	if listOp["operationId"] != "getAdminWidgets" {
		t.Errorf("operationId = %v", listOp["operationId"])
	}
	if !strings.Contains(listOp["description"].(string), "`/api/v1/admin/widgets:GET`") {
		t.Errorf("admin operations must name their permission: %v", listOp["description"])
	}
	var names []string
	for _, p := range listOp["parameters"].([]any) {
		param := p.(map[string]any)
		names = append(names, param["name"].(string))
		if param["name"] == "order" && !strings.Contains(param["description"].(string), "`created_at`") {
			t.Errorf("order must list the allowed fields: %v", param["description"])
		}
	}
	if strings.Join(names, ",") != "page,pageSize,order,searchText,searchFields,filters" {
		t.Errorf("list parameters = %v", names)
	}
	listData := dig(t, listOp, "responses", "200", "content", "application/json", "schema", "properties", "data", "properties").(map[string]any)
	if dig(t, listData, "list", "items", "$ref") != "#/components/schemas/Widget" || listData["totalPages"] == nil {
		t.Errorf("paginated envelope = %v", listData)
	}

	var exportParams []string
	for _, p := range dig(t, paths, "/api/v1/admin/widgets/export", "get", "parameters").([]any) {
		exportParams = append(exportParams, p.(map[string]any)["name"].(string))
	}
	if strings.Join(exportParams, ",") != "order,searchText,searchFields,filters" {
		t.Errorf("export takes the list's filters without paging, got %v", exportParams)
	}

	del := dig(t, paths, "/api/v1/admin/widgets/{id}", "delete").(map[string]any)
	if dig(t, del, "parameters").([]any)[0].(map[string]any)["schema"].(map[string]any)["type"] != "integer" {
		t.Errorf(":id must become an integer path parameter: %v", del["parameters"])
	}
	if dig(t, del, "responses", "200", "content", "application/json", "schema", "properties", "data", "type") != "null" {
		t.Errorf("no Data means data is null")
	}
	if dig(t, del, "responses", "4XX", "$ref") != "#/components/responses/Error" {
		t.Errorf("errors must reference the shared error envelope: %v", del["responses"])
	}
	if len(dig(t, del, "security").([]any)) != 2 {
		t.Errorf("admin operations accept cookie or bearer auth: %v", del["security"])
	}

	file := dig(t, paths, "/api/v1/public/widgets/{slug}/file", "get").(map[string]any)
	if dig(t, file, "responses", "200", "content", "text/csv") == nil || len(file["security"].([]any)) != 0 {
		t.Errorf("public file download = %v", file)
	}
	if dig(t, paths, "/sso/start", "get", "responses", "302", "headers", "Location") == nil || dig(t, paths, "/sso/start", "get", "responses", "200") != nil {
		t.Errorf("redirect operations answer 302 with a Location header only")
	}
	if dig(t, paths, "/events", "get", "responses", "200", "content", "text/event-stream") == nil {
		t.Error("stream operations answer text/event-stream")
	}
	health := dig(t, paths, "/health", "get").(map[string]any)
	if dig(t, health, "responses", "4XX") != nil || dig(t, health, "responses", "200", "content", "application/json", "schema", "properties", "status") == nil {
		t.Errorf("raw responses skip the envelope: %v", health["responses"])
	}
	if !strings.Contains(dig(t, doc, "components", "schemas", "ErrorResponse", "properties", "errorCode", "description").(string), "`1000` Bad request") {
		t.Error("error codes must be listed on ErrorResponse.errorCode")
	}
}

func TestBuildRejectsMistakes(t *testing.T) {
	op := Operation{Method: "GET", Path: "/x", Tag: "X", Summary: "X"}
	if _, err := Build([]Operation{op, op}, Options{}); err == nil {
		t.Error("duplicate operations must be rejected")
	}
	if _, err := Build([]Operation{{Method: "GET", Path: "/y"}}, Options{}); err == nil {
		t.Error("tag and summary are required")
	}
	if _, err := Build([]Operation{op}, Options{TagDescriptions: map[string]string{"Other": "x"}}); err == nil {
		t.Error("a tag without a description must be rejected once descriptions are given")
	}
	if _, err := Build([]Operation{{Method: "GET", Path: "/z", Tag: "Z", Summary: "Z", Data: make(chan int)}}, Options{}); err == nil {
		t.Error("unsupported types must be rejected")
	}
}
