package dto

import "github.com/castorworks/castor/internal/domain/menu"

type MenuRequest struct {
	ParentID    *uint             `json:"parentId"`
	Code        string            `json:"code" binding:"required,max=100"`
	Kind        menu.Kind         `json:"kind" binding:"required"`
	Titles      menu.Titles       `json:"titles"`
	Path        string            `json:"path"`
	Icon        string            `json:"icon"`
	SortOrder   int               `json:"sortOrder"`
	IsEnabled   bool              `json:"isEnabled"`
	AccessMode  string            `json:"accessMode"`
	Permissions []menu.Permission `json:"permissions"`
}

func (r MenuRequest) ToEntity() menu.Menu {
	return menu.Menu{ParentID: r.ParentID, Code: r.Code, Kind: r.Kind, Titles: r.Titles, Path: r.Path, Icon: r.Icon, SortOrder: r.SortOrder, IsEnabled: r.IsEnabled, AccessMode: r.AccessMode, Permissions: r.Permissions}
}

type MenuResp struct{ menu.Menu }

func (r *MenuResp) FromEntity(m *menu.Menu) { r.Menu = *m }

type MenuCatalog struct {
	Menus     []MenuResp     `json:"menus"`
	Resources []ResourceResp `json:"resources"`
}
type MenuRoute struct {
	Path    string `json:"path"`
	Allowed bool   `json:"allowed"`
}
type NavigationResp struct {
	Items  []MenuResp  `json:"items"`
	Routes []MenuRoute `json:"routes"`
}
