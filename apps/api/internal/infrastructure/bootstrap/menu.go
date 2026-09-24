package bootstrap

import (
	"context"
	"fmt"

	"github.com/castorworks/castor/internal/domain/menu"
	"github.com/castorworks/castor/internal/domain/permission"
)

// initializeMenus 把 systemMenuTree 声明的系统导航对账进数据库。
// 新库与已安装的老库走同一条路径：缺失的节点被补齐，已存在的节点原样保留，
// 因此新模块上线后老库也能自动拿到导航入口，而运营人员在「菜单管理」中的
// 改名、排序、禁用、移动不会被覆盖。
func initializeMenus(ctx context.Context, repo menu.Repository, resourcesRepo permission.ResourceRepository) error {
	return repo.WithTx(ctx, func(tx menu.Repository) error {
		if err := tx.Lock(ctx); err != nil {
			return err
		}
		existing, err := tx.List(ctx)
		if err != nil {
			return err
		}
		resources, err := resourcesRepo.GetAll(ctx)
		if err != nil {
			return err
		}
		return createMissingMenus(ctx, tx, existing, systemMenuTree(resources))
	})
}

// createMissingMenus 只做新增：按声明顺序遍历期望树，Code 已存在的节点完全不动，
// 缺失的节点用声明值创建，父节点要么本就存在，要么在本次遍历中刚刚被创建。
func createMissingMenus(ctx context.Context, repo menu.Repository, existing []menu.Menu, desired []desiredMenu) error {
	ids := make(map[string]uint, len(existing)+len(desired))
	for _, item := range existing {
		ids[item.Code] = item.ID
	}
	for _, want := range desired {
		if _, ok := ids[want.Code]; ok {
			// 已存在的节点由运营人员掌管，标题、图标、排序、启停与位置一律保留。
			continue
		}
		node := want.Menu
		node.ID = 0
		node.ParentID = nil
		if want.ParentCode != "" {
			parentID, ok := ids[want.ParentCode]
			if !ok {
				return fmt.Errorf("menu %q declares unknown parent %q", want.Code, want.ParentCode)
			}
			node.ParentID = &parentID
		}
		if err := repo.Save(ctx, &node); err != nil {
			return err
		}
		ids[node.Code] = node.ID
	}
	return nil
}
