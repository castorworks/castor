package authorization

import (
	"context"
	"slices"

	"github.com/castorworks/castor/internal/domain/permission"
)

// InitializeDefaults reconciles system roles and resources during init-db only.
func InitializeDefaults(ctx context.Context, resourcesRepo permission.ResourceRepository, rolesRepo permission.RoleRepository, authzRepo permission.AuthorizationRepository) error {
	resources, err := resourcesRepo.GetAll(ctx)
	if err != nil {
		return err
	}
	resourceByCode := make(map[string]permission.Resource, len(resources))
	for _, resource := range resources {
		resourceByCode[resource.Code] = resource
	}
	defaultResourceCodes := make(map[string]struct{}, len(permission.DefaultResources))
	missingResources := make([]permission.Resource, 0)
	for _, spec := range permission.DefaultResources {
		defaultResourceCodes[spec.Code] = struct{}{}
		current, ok := resourceByCode[spec.Code]
		if !ok {
			missingResources = append(missingResources, spec)
			continue
		}
		spec.ID = current.ID
		spec.CreatedAt = current.CreatedAt
		spec.CreatedBy = current.CreatedBy
		spec.UpdatedAt = current.UpdatedAt
		spec.UpdatedBy = current.UpdatedBy
		spec.IsEnabled = current.IsEnabled
		if !sameResourceDefinition(current, spec) {
			if err := resourcesRepo.Update(ctx, &spec); err != nil {
				return err
			}
			if err := authzRepo.PruneResourcePermissions(ctx, spec.ID, []string(spec.Actions)); err != nil {
				return err
			}
		}
	}
	if err := resourcesRepo.BatchCreate(ctx, missingResources); err != nil {
		return err
	}
	for _, resource := range resources {
		if _, ok := defaultResourceCodes[resource.Code]; resource.IsSystem && !ok {
			if err := resourcesRepo.Delete(ctx, resource.ID); err != nil {
				return err
			}
		}
	}

	roles, err := rolesRepo.GetAll(ctx)
	if err != nil {
		return err
	}
	roleByCode := make(map[string]permission.Role, len(roles))
	for _, role := range roles {
		roleByCode[role.Code] = role
	}
	missingRoles := make([]permission.Role, 0)
	for _, spec := range permission.DefaultRoles {
		current, ok := roleByCode[spec.Code]
		if !ok {
			missingRoles = append(missingRoles, spec)
			continue
		}
		spec.ID = current.ID
		spec.CreatedAt = current.CreatedAt
		spec.CreatedBy = current.CreatedBy
		spec.UpdatedAt = current.UpdatedAt
		spec.UpdatedBy = current.UpdatedBy
		spec.IsEnabled = current.IsEnabled
		if !sameRoleDefinition(current, spec) {
			if err := rolesRepo.Update(ctx, &spec); err != nil {
				return err
			}
		}
	}
	if err := rolesRepo.BatchCreate(ctx, missingRoles); err != nil {
		return err
	}

	resources, err = resourcesRepo.GetAll(ctx)
	if err != nil {
		return err
	}
	roles, err = rolesRepo.GetAll(ctx)
	if err != nil {
		return err
	}
	roleByCode = make(map[string]permission.Role, len(roles))
	for _, role := range roles {
		roleByCode[role.Code] = role
	}
	for _, spec := range []struct {
		code  string
		allow func(permission.Resource, string) bool
	}{
		{permission.RoleAdmin, func(resource permission.Resource, _ string) bool { return resource.IsSystem }},
		{permission.RoleAuditor, func(resource permission.Resource, action string) bool {
			return resource.IsSystem && resource.Category == permission.CategoryAdmin && action == permission.ActionGET
		}},
		{permission.RoleUser, func(resource permission.Resource, _ string) bool {
			return resource.IsSystem && resource.Category == permission.CategoryUser
		}},
	} {
		role, ok := roleByCode[spec.code]
		if !ok {
			continue
		}
		desired := make([]permission.RolePermission, 0)
		for _, resource := range resources {
			for _, action := range resource.Actions {
				if !spec.allow(resource, action) {
					continue
				}
				desired = append(desired, permission.RolePermission{RoleID: role.ID, ResourceID: resource.ID, Action: action, IsSystem: true})
			}
		}
		if err := authzRepo.ReconcileSystemRolePermissions(ctx, role.ID, desired); err != nil {
			return err
		}
	}
	return nil
}

func sameResourceDefinition(left, right permission.Resource) bool {
	return left.Code == right.Code && left.Name == right.Name && left.Description == right.Description &&
		left.Path == right.Path && slices.Equal(left.Actions, right.Actions) && left.Category == right.Category &&
		left.Module == right.Module && left.SortOrder == right.SortOrder && left.IsSystem == right.IsSystem &&
		left.IsEnabled == right.IsEnabled
}

func sameRoleDefinition(left, right permission.Role) bool {
	return left.Code == right.Code && left.Name == right.Name && left.Description == right.Description &&
		left.IsSystem == right.IsSystem && left.IsEnabled == right.IsEnabled
}
