package service

import (
	"errors"
	"reflect"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/permission"
)

func TestNormalizeResourceProducesCanonicalAtomicActions(t *testing.T) {
	resource := &permission.Resource{
		Code:     " Admin:Reports:Read ",
		Name:     " Reports ",
		Path:     " /api/v1/admin/reports/ ",
		Actions:  permission.StringSlice{"post", " GET ", "POST"},
		Category: permission.CategoryAdmin,
		Module:   " reports ",
	}
	if err := normalizeResource(resource); err != nil {
		t.Fatal(err)
	}
	if resource.Code != "admin:reports:read" || resource.Name != "Reports" || resource.Path != "/api/v1/admin/reports" || resource.Module != "reports" {
		t.Fatalf("resource was not normalized: %#v", resource)
	}
	if !reflect.DeepEqual(resource.Actions, permission.StringSlice{permission.ActionGET, permission.ActionPOST}) {
		t.Fatalf("actions were not normalized: %#v", resource.Actions)
	}
}

func TestNormalizeResourceRejectsUnsupportedAction(t *testing.T) {
	resource := &permission.Resource{
		Code:     "admin:reports:read",
		Name:     "Reports",
		Path:     "/api/v1/admin/reports",
		Actions:  permission.StringSlice{"EXECUTE"},
		Category: permission.CategoryAdmin,
	}
	if err := normalizeResource(resource); !errors.Is(err, apperror.ErrInvalidResource) {
		t.Fatalf("expected invalid resource error, got %v", err)
	}
}

func TestDefaultResourcesAreCanonicalAndUnambiguous(t *testing.T) {
	seen := make(map[string]string)
	for _, spec := range permission.DefaultResources {
		resource := spec
		if err := normalizeResource(&resource); err != nil {
			t.Fatalf("default resource %q is invalid: %v", spec.Code, err)
		}
		if !reflect.DeepEqual(resource, spec) {
			t.Fatalf("default resource %q is not canonical", spec.Code)
		}
		for _, action := range spec.Actions {
			key := spec.Path + "\x00" + action
			if previous, ok := seen[key]; ok {
				t.Fatalf("default resources %q and %q define the same path and action", previous, spec.Code)
			}
			seen[key] = spec.Code
		}
	}
}

func TestDefaultResourceTranslationsAreComplete(t *testing.T) {
	for _, language := range []string{"en", "zh", "ja", "ko"} {
		messages := make(map[string]any)
		if _, err := toml.DecodeFile("../../../configs/i18n/"+language+".toml", &messages); err != nil {
			t.Fatal(err)
		}
		for _, resource := range permission.DefaultResources {
			if _, ok := messages[resource.Name]; !ok {
				t.Errorf("%s translation is missing %q", language, resource.Name)
			}
		}
	}
}

func TestResourcePermissionConflictUsesPathAndAtomicAction(t *testing.T) {
	resources := []permission.Resource{{
		ID:       1,
		Path:     "/api/v1/admin/reports",
		Actions:  permission.StringSlice{permission.ActionGET},
		Category: permission.CategoryAdmin,
	}}
	if !resourcePermissionConflict(resources, &permission.Resource{ID: 2, Path: resources[0].Path, Actions: permission.StringSlice{permission.ActionGET}}) {
		t.Fatal("same path and action must conflict")
	}
	if resourcePermissionConflict(resources, &permission.Resource{ID: 2, Path: resources[0].Path, Actions: permission.StringSlice{permission.ActionPOST}}) {
		t.Fatal("different atomic actions may share one path")
	}
	if resourcePermissionConflict(resources, &permission.Resource{ID: 1, Path: resources[0].Path, Actions: permission.StringSlice{permission.ActionGET}}) {
		t.Fatal("an update must not conflict with itself")
	}
}

func TestNormalizeRole(t *testing.T) {
	role := &permission.Role{Code: " Report_Reviewer ", Name: " Reviewer "}
	if err := normalizeRole(role); err != nil {
		t.Fatal(err)
	}
	if role.Code != "report_reviewer" || role.Name != "Reviewer" {
		t.Fatalf("role was not normalized: %#v", role)
	}
	if err := normalizeRole(&permission.Role{Code: "invalid role", Name: "Role"}); !errors.Is(err, apperror.ErrInvalidRole) {
		t.Fatalf("expected invalid role error, got %v", err)
	}
}
