package service

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/menu"
	"github.com/castorworks/castor/internal/domain/permission"
)

type MenuService interface {
	Catalog(context.Context) (*dto.MenuCatalog, error)
	Save(context.Context, *menu.Menu) error
	Delete(context.Context, uint) error
	Navigation(context.Context, []permission.EffectivePermission) (*dto.NavigationResp, error)
}
type menuService struct {
	repo      menu.Repository
	resources permission.ResourceRepository
}

func NewMenuService(repo menu.Repository, resources permission.ResourceRepository) MenuService {
	return &menuService{repo: repo, resources: resources}
}
func (s *menuService) Catalog(ctx context.Context) (*dto.MenuCatalog, error) {
	menus, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	resources, err := s.resources.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	result := &dto.MenuCatalog{Menus: make([]dto.MenuResp, 0, len(menus)), Resources: dto.ToResourceRespList(resources)}
	for _, m := range menus {
		var r dto.MenuResp
		r.FromEntity(&m)
		result.Menus = append(result.Menus, r)
	}
	return result, nil
}

var menuCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
var menuPathPattern = regexp.MustCompile(`^/dashboard(?:/[a-zA-Z0-9_-]+)+$`)
var menuIconPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`)

func validateMenu(m *menu.Menu, all []menu.Menu, resources []permission.Resource) error {
	if m == nil {
		return apperror.ErrInvalidMenu
	}
	m.Code = strings.TrimSpace(m.Code)
	m.Path = strings.TrimSpace(m.Path)
	m.Icon = strings.TrimSpace(m.Icon)
	if !menuCodePattern.MatchString(m.Code) || len(m.Code) > 100 || len(m.Path) > 200 || len(m.Icon) > 50 || (m.Icon != "" && !menuIconPattern.MatchString(m.Icon)) {
		return apperror.ErrInvalidMenu
	}
	for _, title := range []*string{&m.Titles.En, &m.Titles.Zh, &m.Titles.Ja, &m.Titles.Ko} {
		*title = strings.TrimSpace(*title)
		if utf8.RuneCountInString(*title) < 1 || utf8.RuneCountInString(*title) > 100 {
			return apperror.ErrInvalidMenu
		}
	}
	if m.Kind != menu.Directory && m.Kind != menu.Page && m.Kind != menu.Action {
		return apperror.ErrInvalidMenu
	}
	if m.Kind == menu.Page {
		if !menuPathPattern.MatchString(m.Path) {
			return apperror.ErrInvalidMenu
		}
	} else if m.Path != "" {
		return apperror.ErrInvalidMenu
	}
	if m.AccessMode != "authenticated" && m.AccessMode != "permission" {
		return apperror.ErrInvalidMenu
	}
	if m.Kind == menu.Directory && (m.AccessMode != "authenticated" || len(m.Permissions) != 0) {
		return apperror.ErrInvalidMenu
	}
	if m.Kind == menu.Action && m.AccessMode != "permission" {
		return apperror.ErrInvalidMenu
	}
	if (m.AccessMode == "permission" && len(m.Permissions) == 0) || (m.AccessMode == "authenticated" && len(m.Permissions) > 0) {
		return apperror.ErrInvalidMenu
	}
	byID := make(map[uint]menu.Menu, len(all))
	found := m.ID == 0
	for _, other := range all {
		byID[other.ID] = other
		if other.ID == m.ID {
			found = true
			continue
		}
		if other.Code == m.Code || (m.Kind == menu.Page && other.Kind == menu.Page && other.Path == m.Path) {
			return apperror.ErrMenuConflict
		}
	}
	if !found {
		return apperror.ErrMenuNotFound
	}
	if m.ParentID != nil {
		parent, ok := byID[*m.ParentID]
		if !ok || *m.ParentID == m.ID {
			return apperror.ErrMenuHierarchy
		}
		if (m.Kind == menu.Action && parent.Kind != menu.Page) || (m.Kind != menu.Action && parent.Kind != menu.Directory) {
			return apperror.ErrMenuHierarchy
		}
	} else if m.Kind == menu.Action {
		return apperror.ErrMenuHierarchy
	}
	// Validate the whole resulting tree, including children when a type changes.
	if m.ID != 0 {
		byID[m.ID] = *m
	}
	for _, node := range append(all, *m) {
		if node.ID == m.ID {
			node = *m
		}
		seen := map[uint]bool{}
		depth := 0
		for node.ParentID != nil {
			id := *node.ParentID
			depth++
			if seen[id] || depth > 8 || (m.ID != 0 && node.ID == id) {
				return apperror.ErrMenuHierarchy
			}
			seen[id] = true
			parent, ok := byID[id]
			if !ok {
				return apperror.ErrMenuHierarchy
			}
			if (node.Kind == menu.Action && parent.Kind != menu.Page) || (node.Kind != menu.Action && parent.Kind != menu.Directory) {
				return apperror.ErrMenuHierarchy
			}
			node = parent
		}
	}
	byResource := make(map[uint]permission.Resource, len(resources))
	for _, r := range resources {
		byResource[r.ID] = r
	}
	seen := make(map[menu.Permission]bool)
	refs := make([]menu.Permission, 0, len(m.Permissions))
	for _, p := range m.Permissions {
		r, ok := byResource[p.ResourceID]
		if !ok || !r.Actions.Contains(p.Action) {
			return apperror.ErrMenuPermission
		}
		if !seen[p] {
			refs = append(refs, p)
			seen[p] = true
		}
	}
	m.Permissions = refs
	return nil
}
func (s *menuService) Save(ctx context.Context, m *menu.Menu) error {
	return s.repo.WithTx(ctx, func(tx menu.Repository) error {
		if err := tx.Lock(ctx); err != nil {
			return err
		}
		all, err := tx.List(ctx)
		if err != nil {
			return err
		}
		resources, err := s.resources.GetAll(ctx)
		if err != nil {
			return err
		}
		if err = validateMenu(m, all, resources); err != nil {
			return err
		}
		for _, existing := range all {
			if existing.ID == m.ID {
				m.BaseModel = existing.BaseModel
				break
			}
		}
		return tx.Save(ctx, m)
	})
}
func (s *menuService) Delete(ctx context.Context, id uint) error {
	return s.repo.WithTx(ctx, func(tx menu.Repository) error {
		if err := tx.Lock(ctx); err != nil {
			return err
		}
		all, err := tx.List(ctx)
		if err != nil {
			return err
		}
		found := false
		for _, m := range all {
			if m.ID == id {
				found = true
			}
			if m.ParentID != nil && *m.ParentID == id {
				return apperror.ErrMenuHasChildren
			}
		}
		if !found {
			return apperror.ErrMenuNotFound
		}
		return tx.Delete(ctx, id)
	})
}
func (s *menuService) Navigation(ctx context.Context, grants []permission.EffectivePermission) (*dto.NavigationResp, error) {
	all, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return menuNavigation(all, grants), nil
}
func menuNavigation(all []menu.Menu, grants []permission.EffectivePermission) *dto.NavigationResp {
	result := &dto.NavigationResp{Items: []dto.MenuResp{}, Routes: []dto.MenuRoute{}}
	effective := make(map[menu.Permission]bool, len(grants))
	for _, p := range grants {
		effective[menu.Permission{ResourceID: p.ResourceID, Action: p.Action}] = true
	}
	byID := make(map[uint]menu.Menu, len(all))
	for _, m := range all {
		byID[m.ID] = m
	}
	visible := make(map[uint]bool)
	for _, m := range all {
		if m.Kind != menu.Page {
			continue
		}
		allowed := m.IsEnabled && (m.AccessMode == "authenticated")
		if m.IsEnabled && m.AccessMode == "permission" {
			for _, p := range m.Permissions {
				if effective[p] {
					allowed = true
					break
				}
			}
		}
		ancestors := []uint{}
		node := m
		seen := map[uint]bool{m.ID: true}
		for node.ParentID != nil {
			id := *node.ParentID
			parent, ok := byID[id]
			if !ok || seen[id] || !parent.IsEnabled || parent.Kind != menu.Directory {
				allowed = false
				break
			}
			ancestors = append(ancestors, id)
			seen[id] = true
			node = parent
		}
		result.Routes = append(result.Routes, dto.MenuRoute{Path: m.Path, Allowed: allowed})
		if allowed {
			visible[m.ID] = true
			for _, id := range ancestors {
				visible[id] = true
			}
		}
	}
	for _, m := range all {
		if visible[m.ID] {
			m.Permissions = []menu.Permission{}
			result.Items = append(result.Items, dto.MenuResp{Menu: m})
		}
	}
	return result
}
