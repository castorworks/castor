package database

import (
	"fmt"

	"github.com/castorworks/castor/internal/domain/dictionary"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// SeedDictionary 把内置字典类型与字典项对账进数据库，由 init-db 在迁移之后每次执行。
//
// 与菜单一致：改完 DefaultDictTypes / DefaultDictItems，重跑 init-db 即可，新增默认项不需要迁移。
//
// 对账只增不改：按 code / (type_code, value) 判重，已存在的行一律保留，
// 运营在「字典管理」里改过的标签、颜色、图标、排序、启停、公开范围都不会被覆盖。
// 唯一的例外是 is_system——它不是运营可编辑的字段，而是"这一项被代码引用"的事实，
// 所以种子对应的行会被强制标记为系统项，从而受到取值不可改、不可删除的保护。
func SeedDictionary(db *gorm.DB) error {
	if err := seedDictionaryRows(db); err != nil {
		return err
	}
	return markSystemDictionaryRows(db)
}

// seedDictionaryRows 补建缺失的字典类型与字典项；初始结构建表后也调用它。
func seedDictionaryRows(db *gorm.DB) error {
	if err := seedDictTypes(db); err != nil {
		return fmt.Errorf("seed dict types: %w", err)
	}
	if err := seedDictItems(db); err != nil {
		return fmt.Errorf("seed dict items: %w", err)
	}
	return nil
}

func seedDictTypes(db *gorm.DB) error {
	for i := range dictionary.DefaultDictTypes {
		dt := &dictionary.DefaultDictTypes[i]
		// 用 Count 而不是 First：对账每次 init-db 都跑，First 未命中会刷一屏 record not found 日志
		var existing int64
		if err := db.Model(&models.DictTypeModel{}).Where("code = ?", dt.Code).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			continue
		}
		if err := db.Create(models.DictTypeModelFromEntity(dt)).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedDictItems(db *gorm.DB) error {
	for i := range dictionary.DefaultDictItems {
		di := &dictionary.DefaultDictItems[i]
		var existing int64
		if err := db.Model(&models.DictItemModel{}).
			Where("type_code = ? AND value = ?", di.TypeCode, di.Value).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			continue
		}
		if err := db.Create(models.DictItemModelFromEntity(di)).Error; err != nil {
			return err
		}
	}
	return nil
}

// markSystemDictionaryRows 把种子对应的已有行标记为系统项，
// 包括运营先手工建了某一项、后来代码才把它收编为种子的情况。
func markSystemDictionaryRows(db *gorm.DB) error {
	for _, dt := range dictionary.DefaultDictTypes {
		if err := db.Model(&models.DictTypeModel{}).
			Where("code = ? AND is_system = ?", dt.Code, false).
			UpdateColumn("is_system", true).Error; err != nil {
			return fmt.Errorf("mark dict type %q as system: %w", dt.Code, err)
		}
	}
	for _, di := range dictionary.DefaultDictItems {
		if err := db.Model(&models.DictItemModel{}).
			Where("type_code = ? AND value = ? AND is_system = ?", di.TypeCode, di.Value, false).
			UpdateColumn("is_system", true).Error; err != nil {
			return fmt.Errorf("mark dict item %s/%s as system: %w", di.TypeCode, di.Value, err)
		}
	}
	return nil
}
