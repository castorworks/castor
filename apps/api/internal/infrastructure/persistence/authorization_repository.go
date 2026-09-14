package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type authorizationRepository struct {
	db *gorm.DB
}

func NewAuthorizationRepository(db *gorm.DB) permission.AuthorizationRepository {
	return &authorizationRepository{db: db}
}

func (r *authorizationRepository) q(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *authorizationRepository) WithTx(ctx context.Context, fn func(permission.AuthorizationRepository) error) error {
	return r.q(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&authorizationRepository{db: tx})
	})
}

func (r *authorizationRepository) LockUsers(ctx context.Context, userIDs ...uint) error {
	if len(userIDs) == 0 {
		return nil
	}
	var rows []models.UserModel
	return r.q(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", userIDs).Find(&rows).Error
}

func (r *authorizationRepository) LockRoles(ctx context.Context) error {
	var rows []models.RoleModel
	return r.q(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Order("id").Find(&rows).Error
}

func (r *authorizationRepository) LockSession(ctx context.Context, sessionID string) error {
	var row models.AuthorizationSessionModel
	return translateError(r.q(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", sessionID).First(&row).Error)
}

func (r *authorizationRepository) CreateResource(ctx context.Context, resource *permission.Resource) error {
	row := models.ResourceModelFromEntity(resource)
	if err := r.q(ctx).Create(row).Error; err != nil {
		return err
	}
	*resource = *row.ToEntity()
	return nil
}

func (r *authorizationRepository) UpdateResource(ctx context.Context, resource *permission.Resource) error {
	row := models.ResourceModelFromEntity(resource)
	if err := r.q(ctx).Save(row).Error; err != nil {
		return err
	}
	*resource = *row.ToEntity()
	return nil
}

func (r *authorizationRepository) DeleteResource(ctx context.Context, id uint) error {
	return r.q(ctx).Delete(&models.ResourceModel{}, id).Error
}

func (r *authorizationRepository) CreateRole(ctx context.Context, role *permission.Role) error {
	row := models.RoleModelFromEntity(role)
	if err := r.q(ctx).Create(row).Error; err != nil {
		return err
	}
	*role = *row.ToEntity()
	return nil
}

func (r *authorizationRepository) UpdateRole(ctx context.Context, role *permission.Role) error {
	row := models.RoleModelFromEntity(role)
	if err := r.q(ctx).Save(row).Error; err != nil {
		return err
	}
	*role = *row.ToEntity()
	return nil
}

func (r *authorizationRepository) DeleteRole(ctx context.Context, id uint) error {
	return r.q(ctx).Delete(&models.RoleModel{}, id).Error
}

func rolePermissionEntity(m models.RolePermissionModel) permission.RolePermission {
	return permission.RolePermission{ID: m.ID, CreatedAt: m.CreatedAt, CreatedBy: m.CreatedBy, RoleID: m.RoleID, ResourceID: m.ResourceID, Action: m.Action, IsSystem: m.IsSystem}
}

func (r *authorizationRepository) GetAllRolePermissions(ctx context.Context) ([]permission.RolePermission, error) {
	var rows []models.RolePermissionModel
	if err := r.q(ctx).Order("role_id, resource_id, action").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]permission.RolePermission, len(rows))
	for i := range rows {
		result[i] = rolePermissionEntity(rows[i])
	}
	return result, nil
}

func (r *authorizationRepository) GetRolePermissions(ctx context.Context, roleID uint) ([]permission.RolePermission, error) {
	var rows []models.RolePermissionModel
	if err := r.q(ctx).Where("role_id = ?", roleID).Order("resource_id, action").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]permission.RolePermission, len(rows))
	for i := range rows {
		result[i] = rolePermissionEntity(rows[i])
	}
	return result, nil
}

func (r *authorizationRepository) ReplaceRolePermissions(ctx context.Context, roleID uint, permissions []permission.RolePermission) error {
	return r.q(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&models.RolePermissionModel{}).Error; err != nil {
			return err
		}
		if len(permissions) == 0 {
			return nil
		}
		rows := make([]models.RolePermissionModel, len(permissions))
		for i := range permissions {
			rows[i] = *models.RolePermissionModelFromEntity(&permissions[i])
		}
		return tx.Create(&rows).Error
	})
}

func (r *authorizationRepository) ReconcileSystemRolePermissions(ctx context.Context, roleID uint, permissions []permission.RolePermission) error {
	return r.q(ctx).Transaction(func(tx *gorm.DB) error {
		var role models.RoleModel
		if err := translateError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&role, roleID).Error); err != nil {
			return err
		}
		var existing []models.RolePermissionModel
		if err := tx.Where("role_id = ?", roleID).Find(&existing).Error; err != nil {
			return err
		}
		desired := make(map[string]permission.RolePermission, len(permissions))
		for _, row := range permissions {
			row.RoleID = roleID
			row.IsSystem = true
			desired[authorizationPermissionKey(row.ResourceID, row.Action)] = row
		}
		for _, row := range existing {
			key := authorizationPermissionKey(row.ResourceID, row.Action)
			if _, ok := desired[key]; ok {
				delete(desired, key)
				continue
			}
			if row.IsSystem {
				if err := tx.Delete(&row).Error; err != nil {
					return err
				}
			}
		}
		if len(desired) == 0 {
			return nil
		}
		rows := make([]models.RolePermissionModel, 0, len(desired))
		for _, item := range desired {
			rows = append(rows, *models.RolePermissionModelFromEntity(&item))
		}
		return tx.Create(&rows).Error
	})
}

func authorizationPermissionKey(resourceID uint, action string) string {
	return fmt.Sprintf("%d\x00%s", resourceID, action)
}

func (r *authorizationRepository) PruneResourcePermissions(ctx context.Context, resourceID uint, allowedActions []string) error {
	query := r.q(ctx).Where("resource_id = ?", resourceID)
	if len(allowedActions) > 0 {
		query = query.Where("action NOT IN ?", allowedActions)
	}
	return query.Delete(&models.RolePermissionModel{}).Error
}

func userRoleEntity(m models.UserRoleModel) permission.UserRole {
	return permission.UserRole{ID: m.ID, CreatedAt: m.CreatedAt, CreatedBy: m.CreatedBy, UserID: m.UserID, RoleID: m.RoleID, IsSystem: m.IsSystem}
}

func (r *authorizationRepository) GetAllUserRoles(ctx context.Context) ([]permission.UserRole, error) {
	var rows []models.UserRoleModel
	if err := r.q(ctx).Order("user_id, role_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]permission.UserRole, len(rows))
	for i := range rows {
		result[i] = userRoleEntity(rows[i])
	}
	return result, nil
}

func (r *authorizationRepository) GetUserRoles(ctx context.Context, userID uint) ([]permission.UserRole, error) {
	var rows []models.UserRoleModel
	if err := r.q(ctx).Where("user_id = ?", userID).Order("role_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]permission.UserRole, len(rows))
	for i := range rows {
		result[i] = userRoleEntity(rows[i])
	}
	return result, nil
}

func (r *authorizationRepository) GetRoleUserIDs(ctx context.Context, roleID uint) ([]uint, error) {
	var ids []uint
	return ids, r.q(ctx).Model(&models.UserRoleModel{}).Where("role_id = ?", roleID).Order("user_id").Pluck("user_id", &ids).Error
}

func (r *authorizationRepository) AddUserRole(ctx context.Context, assignment *permission.UserRole) error {
	row := &models.UserRoleModel{ID: assignment.ID, CreatedAt: assignment.CreatedAt, CreatedBy: assignment.CreatedBy, UserID: assignment.UserID, RoleID: assignment.RoleID, IsSystem: assignment.IsSystem}
	if err := r.q(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error; err != nil {
		return err
	}
	assignment.ID, assignment.CreatedAt, assignment.CreatedBy = row.ID, row.CreatedAt, row.CreatedBy
	return nil
}

func (r *authorizationRepository) DeleteUserRole(ctx context.Context, userID, roleID uint) error {
	return r.q(ctx).Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&models.UserRoleModel{}).Error
}

func (r *authorizationRepository) DeleteUserRoles(ctx context.Context, userID uint) error {
	return r.q(ctx).Where("user_id = ?", userID).Delete(&models.UserRoleModel{}).Error
}

func (r *authorizationRepository) GetRoleHierarchies(ctx context.Context) ([]permission.RoleHierarchy, error) {
	var rows []models.RoleHierarchyModel
	if err := r.q(ctx).Order("senior_role_id, junior_role_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]permission.RoleHierarchy, len(rows))
	for i, row := range rows {
		result[i] = permission.RoleHierarchy{ID: row.ID, CreatedAt: row.CreatedAt, CreatedBy: row.CreatedBy, SeniorRoleID: row.SeniorRoleID, JuniorRoleID: row.JuniorRoleID}
	}
	return result, nil
}

func (r *authorizationRepository) ReplaceRoleJuniors(ctx context.Context, seniorRoleID uint, juniorRoleIDs []uint) error {
	return r.q(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("senior_role_id = ?", seniorRoleID).Delete(&models.RoleHierarchyModel{}).Error; err != nil {
			return err
		}
		if len(juniorRoleIDs) == 0 {
			return nil
		}
		rows := make([]models.RoleHierarchyModel, len(juniorRoleIDs))
		for i, id := range juniorRoleIDs {
			rows[i] = models.RoleHierarchyModel{SeniorRoleID: seniorRoleID, JuniorRoleID: id}
		}
		return tx.Create(&rows).Error
	})
}

func (r *authorizationRepository) GetConstraints(ctx context.Context) ([]permission.SeparationConstraint, error) {
	var rows []models.SeparationConstraintModel
	if err := r.q(ctx).Order("type, code").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]permission.SeparationConstraint, len(rows))
	for i := range rows {
		result[i] = constraintEntity(rows[i])
		if err := r.q(ctx).Model(&models.ConstraintRoleModel{}).Where("constraint_id = ?", rows[i].ID).Order("role_id").Pluck("role_id", &result[i].RoleIDs).Error; err != nil {
			return nil, err
		}
	}
	return result, nil
}

func constraintEntity(m models.SeparationConstraintModel) permission.SeparationConstraint {
	return permission.SeparationConstraint{ID: m.ID, CreatedAt: m.CreatedAt, CreatedBy: m.CreatedBy, UpdatedAt: m.UpdatedAt, UpdatedBy: m.UpdatedBy, Code: m.Code, Name: m.Name, Description: m.Description, Type: m.Type, Cardinality: m.Cardinality, IsEnabled: m.IsEnabled}
}

func (r *authorizationRepository) GetConstraint(ctx context.Context, id uint) (*permission.SeparationConstraint, error) {
	var row models.SeparationConstraintModel
	if err := translateError(r.q(ctx).First(&row, id).Error); err != nil {
		return nil, err
	}
	result := constraintEntity(row)
	if err := r.q(ctx).Model(&models.ConstraintRoleModel{}).Where("constraint_id = ?", id).Order("role_id").Pluck("role_id", &result.RoleIDs).Error; err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *authorizationRepository) CreateConstraint(ctx context.Context, constraint *permission.SeparationConstraint) error {
	row := models.SeparationConstraintModel{Code: constraint.Code, Name: constraint.Name, Description: constraint.Description, Type: constraint.Type, Cardinality: constraint.Cardinality, IsEnabled: constraint.IsEnabled}
	if err := r.q(ctx).Create(&row).Error; err != nil {
		return err
	}
	constraint.ID, constraint.CreatedAt, constraint.UpdatedAt = row.ID, row.CreatedAt, row.UpdatedAt
	return r.replaceConstraintRoles(ctx, row.ID, constraint.RoleIDs)
}

func (r *authorizationRepository) UpdateConstraint(ctx context.Context, constraint *permission.SeparationConstraint) error {
	row := models.SeparationConstraintModel{ID: constraint.ID, CreatedAt: constraint.CreatedAt, CreatedBy: constraint.CreatedBy, Code: constraint.Code, Name: constraint.Name, Description: constraint.Description, Type: constraint.Type, Cardinality: constraint.Cardinality, IsEnabled: constraint.IsEnabled}
	if err := r.q(ctx).Save(&row).Error; err != nil {
		return err
	}
	constraint.UpdatedAt, constraint.UpdatedBy = row.UpdatedAt, row.UpdatedBy
	return r.replaceConstraintRoles(ctx, constraint.ID, constraint.RoleIDs)
}

func (r *authorizationRepository) replaceConstraintRoles(ctx context.Context, constraintID uint, roleIDs []uint) error {
	if err := r.q(ctx).Where("constraint_id = ?", constraintID).Delete(&models.ConstraintRoleModel{}).Error; err != nil {
		return err
	}
	if len(roleIDs) == 0 {
		return nil
	}
	rows := make([]models.ConstraintRoleModel, len(roleIDs))
	for i, id := range roleIDs {
		rows[i] = models.ConstraintRoleModel{ConstraintID: constraintID, RoleID: id}
	}
	return r.q(ctx).Create(&rows).Error
}

func (r *authorizationRepository) DeleteConstraint(ctx context.Context, id uint) error {
	return r.q(ctx).Delete(&models.SeparationConstraintModel{}, id).Error
}

func (r *authorizationRepository) CreateSession(ctx context.Context, session *permission.AuthorizationSession, activeRoleIDs []uint) error {
	return r.q(ctx).Transaction(func(tx *gorm.DB) error {
		row := models.AuthorizationSessionModel{ID: session.ID, UserID: session.UserID, CreatedAt: session.CreatedAt, ExpiresAt: session.ExpiresAt, RevokedAt: session.RevokedAt, Version: session.Version}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if len(activeRoleIDs) > 0 {
			roles := make([]models.SessionRoleModel, len(activeRoleIDs))
			for i, id := range activeRoleIDs {
				roles[i] = models.SessionRoleModel{SessionID: session.ID, RoleID: id}
			}
			if err := tx.Create(&roles).Error; err != nil {
				return err
			}
		}
		session.CreatedAt = row.CreatedAt
		return nil
	})
}

func (r *authorizationRepository) GetSession(ctx context.Context, id string) (*permission.AuthorizationSession, error) {
	var row models.AuthorizationSessionModel
	if err := translateError(r.q(ctx).Where("id = ?", id).First(&row).Error); err != nil {
		return nil, err
	}
	return &permission.AuthorizationSession{ID: row.ID, UserID: row.UserID, CreatedAt: row.CreatedAt, ExpiresAt: row.ExpiresAt, RevokedAt: row.RevokedAt, Version: row.Version}, nil
}

func (r *authorizationRepository) GetSessionRoleIDs(ctx context.Context, sessionID string) ([]uint, error) {
	var ids []uint
	return ids, r.q(ctx).Model(&models.SessionRoleModel{}).Where("session_id = ?", sessionID).Order("role_id").Pluck("role_id", &ids).Error
}

func (r *authorizationRepository) ReplaceSessionRoles(ctx context.Context, sessionID string, roleIDs []uint) error {
	return r.q(ctx).Transaction(func(tx *gorm.DB) error {
		var session models.AuthorizationSessionModel
		if err := translateError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", sessionID).First(&session).Error); err != nil {
			return err
		}
		if err := tx.Where("session_id = ?", sessionID).Delete(&models.SessionRoleModel{}).Error; err != nil {
			return err
		}
		if len(roleIDs) > 0 {
			rows := make([]models.SessionRoleModel, len(roleIDs))
			for i, id := range roleIDs {
				rows[i] = models.SessionRoleModel{SessionID: sessionID, RoleID: id}
			}
			if err := tx.Create(&rows).Error; err != nil {
				return err
			}
		}
		return tx.Model(&session).UpdateColumn("version", gorm.Expr("version + 1")).Error
	})
}

func (r *authorizationRepository) GetActiveSessions(ctx context.Context) ([]permission.AuthorizationSession, error) {
	var rows []models.AuthorizationSessionModel
	if err := r.q(ctx).Where("revoked_at IS NULL AND expires_at > ?", time.Now()).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]permission.AuthorizationSession, len(rows))
	for i, row := range rows {
		result[i] = permission.AuthorizationSession{ID: row.ID, UserID: row.UserID, CreatedAt: row.CreatedAt, ExpiresAt: row.ExpiresAt, RevokedAt: row.RevokedAt, Version: row.Version}
	}
	return result, nil
}

func (r *authorizationRepository) RevokeSession(ctx context.Context, id string) error {
	now := time.Now()
	return r.q(ctx).Model(&models.AuthorizationSessionModel{}).Where("id = ? AND revoked_at IS NULL", id).Updates(map[string]any{"revoked_at": now, "version": gorm.Expr("version + 1")}).Error
}

func (r *authorizationRepository) RevokeUserSessions(ctx context.Context, userID uint) error {
	now := time.Now()
	return r.q(ctx).Model(&models.AuthorizationSessionModel{}).Where("user_id = ? AND revoked_at IS NULL", userID).Updates(map[string]any{"revoked_at": now, "version": gorm.Expr("version + 1")}).Error
}

func (r *authorizationRepository) RevokeSessionsWithRole(ctx context.Context, roleID uint) error {
	var sessionIDs []string
	if err := r.q(ctx).Model(&models.SessionRoleModel{}).Where("role_id = ?", roleID).Pluck("session_id", &sessionIDs).Error; err != nil {
		return err
	}
	if len(sessionIDs) == 0 {
		return nil
	}
	now := time.Now()
	return r.q(ctx).Model(&models.AuthorizationSessionModel{}).Where("id IN ? AND revoked_at IS NULL", sessionIDs).Updates(map[string]any{"revoked_at": now, "version": gorm.Expr("version + 1")}).Error
}
