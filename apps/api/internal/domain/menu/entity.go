package menu

import "github.com/castorworks/castor/internal/domain/shared"

type Kind string

const (
	Directory Kind = "directory"
	Page      Kind = "page"
	Action    Kind = "action"
)

// Titles 是菜单节点的四语言标题，与字典标签共用同一种文案表示。
type Titles = shared.I18nText
type Permission struct {
	ResourceID uint   `json:"resourceId"`
	Action     string `json:"action"`
}

// Menu organizes existing permissions. It does not grant permissions itself.
type Menu struct {
	shared.BaseModel
	ParentID    *uint        `json:"parentId"`
	Code        string       `json:"code"`
	Kind        Kind         `json:"kind"`
	Titles      Titles       `json:"titles"`
	Path        string       `json:"path"`
	Icon        string       `json:"icon"`
	SortOrder   int          `json:"sortOrder"`
	IsEnabled   bool         `json:"isEnabled"`
	AccessMode  string       `json:"accessMode"`
	Permissions []Permission `json:"permissions"`
}
