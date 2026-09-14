package service

import (
	"errors"
	"testing"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/menu"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/shared"
)

func menuID(id uint) *uint { return &id }
func exampleMenu(id uint, kind menu.Kind, parent *uint) menu.Menu {
	m := menu.Menu{BaseModel: shared.BaseModel{ID: id}, Code: "item", Kind: kind, ParentID: parent, Titles: menu.Titles{En: "Item", Zh: "项目", Ja: "項目", Ko: "항목"}, IsEnabled: true, AccessMode: "authenticated", Permissions: []menu.Permission{}}
	if kind == menu.Page {
		m.Path = "/dashboard/reports"
		m.AccessMode = "permission"
		m.Permissions = []menu.Permission{{ResourceID: 1, Action: "GET"}}
	}
	if kind == menu.Action {
		m.AccessMode = "permission"
		m.Permissions = []menu.Permission{{ResourceID: 1, Action: "POST"}}
	}
	return m
}
func TestValidateMenuCatalog(t *testing.T) {
	resources := []permission.Resource{{ID: 1, Actions: permission.StringSlice{"GET", "POST"}}}
	root := exampleMenu(1, menu.Directory, nil)
	root.Code = "root"
	tests := []struct {
		name   string
		change func(*menu.Menu)
		all    []menu.Menu
		err    error
	}{
		{name: "page with permission", all: []menu.Menu{root}},
		{name: "title required in all languages", change: func(m *menu.Menu) { m.Titles.Ko = " " }, err: apperror.ErrInvalidMenu},
		{name: "external URL", change: func(m *menu.Menu) { m.Path = "https://example.org" }, err: apperror.ErrInvalidMenu},
		{name: "protocol relative URL", change: func(m *menu.Menu) { m.Path = "//example.org" }, err: apperror.ErrInvalidMenu},
		{name: "traversal path", change: func(m *menu.Menu) { m.Path = "/dashboard/../auth" }, err: apperror.ErrInvalidMenu},
		{name: "missing permissions", change: func(m *menu.Menu) { m.Permissions = nil }, err: apperror.ErrInvalidMenu},
		{name: "unknown resource", change: func(m *menu.Menu) { m.Permissions[0].ResourceID = 2 }, err: apperror.ErrMenuPermission},
		{name: "unsupported operation", change: func(m *menu.Menu) { m.Permissions[0].Action = "DELETE" }, err: apperror.ErrMenuPermission},
		{name: "missing parent", change: func(m *menu.Menu) { m.ParentID = menuID(8) }, err: apperror.ErrMenuHierarchy},
		{name: "duplicate code", all: []menu.Menu{exampleMenu(1, menu.Directory, nil)}, err: apperror.ErrMenuConflict},
		{name: "duplicate path", all: []menu.Menu{func() menu.Menu { m := exampleMenu(1, menu.Page, nil); m.Code = "another"; return m }()}, err: apperror.ErrMenuConflict},
		{name: "action cannot be root", change: func(m *menu.Menu) { m.Kind = menu.Action; m.Path = "" }, err: apperror.ErrMenuHierarchy},
		{name: "authenticated page cannot carry permissions", change: func(m *menu.Menu) { m.AccessMode = "authenticated" }, err: apperror.ErrInvalidMenu},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := exampleMenu(0, menu.Page, nil)
			if tt.change != nil {
				tt.change(&m)
			}
			if err := validateMenu(&m, tt.all, resources); !errors.Is(err, tt.err) {
				t.Fatalf("got %v want %v", err, tt.err)
			}
		})
	}
	child := exampleMenu(2, menu.Directory, menuID(1))
	child.Code = "child"
	root.ParentID = menuID(2)
	if err := validateMenu(&root, []menu.Menu{root, child}, resources); !errors.Is(err, apperror.ErrMenuHierarchy) {
		t.Fatalf("cycle accepted: %v", err)
	}
	root.ParentID = nil
	root.Kind = menu.Page
	root.Path = "/dashboard/root"
	if err := validateMenu(&root, []menu.Menu{root, child}, resources); !errors.Is(err, apperror.ErrMenuHierarchy) {
		t.Fatalf("invalid child type accepted: %v", err)
	}
}
func TestMenuNavigationUsesOnlyEffectivePermissions(t *testing.T) {
	root := exampleMenu(1, menu.Directory, nil)
	page := exampleMenu(2, menu.Page, menuID(1))
	action := exampleMenu(3, menu.Action, menuID(2))
	account := exampleMenu(4, menu.Page, nil)
	account.Path = "/dashboard/profile"
	account.AccessMode = "authenticated"
	account.Permissions = nil
	all := []menu.Menu{root, page, action, account}
	navigation := menuNavigation(all, nil)
	if len(navigation.Items) != 1 || navigation.Items[0].ID != 4 {
		t.Fatal("ungranted page or empty directory leaked")
	}
	grants := []permission.EffectivePermission{{ResourceID: 1, Action: "GET"}}
	navigation = menuNavigation(all, grants)
	if len(navigation.Items) != 3 || !navigation.Routes[0].Allowed {
		t.Fatalf("granted page hidden: %#v", navigation)
	}
	for _, m := range navigation.Items {
		if m.Kind == menu.Action || len(m.Permissions) > 0 {
			t.Fatal("operation node or permission catalog leaked into navigation")
		}
	}
	all[0].IsEnabled = false
	if navigation = menuNavigation(all, grants); navigation.Routes[0].Allowed || len(navigation.Items) != 1 {
		t.Fatal("disabled ancestor failed to hide descendants")
	}
	all[0].IsEnabled = true
	all[1].Permissions = nil
	if menuNavigation(all, grants).Routes[0].Allowed {
		t.Fatal("permission page without bindings must fail closed")
	}
}
