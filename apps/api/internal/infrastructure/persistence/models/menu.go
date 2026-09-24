package models

import (
	"time"

	"github.com/castorworks/castor/internal/domain/menu"
	"github.com/castorworks/castor/internal/domain/shared"
	"gorm.io/gorm"
)

type MenuModel struct {
	ID          uint `gorm:"primarykey"`
	CreatedAt   time.Time
	CreatedBy   uint
	UpdatedAt   time.Time
	UpdatedBy   uint
	ParentID    *uint      `gorm:"index"`
	Parent      *MenuModel `gorm:"foreignKey:ParentID;constraint:OnDelete:RESTRICT"`
	Code        string     `gorm:"size:100;not null;uniqueIndex"`
	Kind        menu.Kind  `gorm:"size:16;not null;check:chk_menu_kind,kind IN ('directory','page','action')"`
	TitleEn     string     `gorm:"size:100;not null"`
	TitleZh     string     `gorm:"size:100;not null"`
	TitleJa     string     `gorm:"size:100;not null"`
	TitleKo     string     `gorm:"size:100;not null"`
	Path        string     `gorm:"size:200;uniqueIndex:idx_menu_page_path,where:kind = 'page'"`
	Icon        string     `gorm:"size:50"`
	SortOrder   int
	IsEnabled   bool                  `gorm:"not null"`
	AccessMode  string                `gorm:"size:16;not null;check:chk_menu_access,access_mode IN ('authenticated','permission')"`
	Permissions []MenuPermissionModel `gorm:"foreignKey:MenuID;constraint:OnDelete:CASCADE"`
}

func (MenuModel) TableName() string { return "rbac_menus" }
func (m *MenuModel) BeforeCreate(tx *gorm.DB) error {
	m.CreatedBy, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}
func (m *MenuModel) BeforeUpdate(tx *gorm.DB) error {
	_, m.UpdatedBy = ExtractUserIDFromContext(tx)
	return nil
}

type MenuPermissionModel struct {
	MenuID     uint          `gorm:"primaryKey;autoIncrement:false"`
	ResourceID uint          `gorm:"primaryKey;autoIncrement:false"`
	Action     string        `gorm:"primaryKey;size:16"`
	Resource   ResourceModel `gorm:"foreignKey:ResourceID;constraint:OnDelete:CASCADE"`
}

func (MenuPermissionModel) TableName() string { return "rbac_menu_permissions" }
func (m *MenuModel) ToEntity() menu.Menu {
	refs := make([]menu.Permission, 0, len(m.Permissions))
	for _, p := range m.Permissions {
		refs = append(refs, menu.Permission{ResourceID: p.ResourceID, Action: p.Action})
	}
	return menu.Menu{BaseModel: shared.BaseModel{ID: m.ID, CreatedAt: m.CreatedAt, CreatedBy: m.CreatedBy, UpdatedAt: m.UpdatedAt, UpdatedBy: m.UpdatedBy}, ParentID: m.ParentID, Code: m.Code, Kind: m.Kind, Titles: menu.Titles{En: m.TitleEn, Zh: m.TitleZh, Ja: m.TitleJa, Ko: m.TitleKo}, Path: m.Path, Icon: m.Icon, SortOrder: m.SortOrder, IsEnabled: m.IsEnabled, AccessMode: m.AccessMode, Permissions: refs}
}
func MenuModelFromEntity(m *menu.Menu) *MenuModel {
	return &MenuModel{ID: m.ID, CreatedAt: m.CreatedAt, CreatedBy: m.CreatedBy, UpdatedAt: m.UpdatedAt, UpdatedBy: m.UpdatedBy, ParentID: m.ParentID, Code: m.Code, Kind: m.Kind, TitleEn: m.Titles.En, TitleZh: m.Titles.Zh, TitleJa: m.Titles.Ja, TitleKo: m.Titles.Ko, Path: m.Path, Icon: m.Icon, SortOrder: m.SortOrder, IsEnabled: m.IsEnabled, AccessMode: m.AccessMode}
}
