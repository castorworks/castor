import { mutationOptions } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import {
  createRole,
  createConstraint,
  deleteConstraint,
  deleteRole,
  setRolePermissions,
  setRoleHierarchy,
  updateConstraint,
  updateRole,
  toggleRoleEnabled
} from './service';
import { roleKeys } from './queries';
import { useAuthStore } from '@/stores/auth-store';
import type {
  ConstraintPayload,
  CreateRolePayload,
  PermissionGrant,
  UpdateRolePayload
} from './types';

export const createRoleMutation = mutationOptions({
  mutationFn: (data: CreateRolePayload) => createRole(data),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: roleKeys.all });
  }
});

export const updateRoleMutation = mutationOptions({
  mutationFn: ({ code, data }: { code: string; data: UpdateRolePayload }) => updateRole(code, data),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: roleKeys.all });
  }
});

export const deleteRoleMutation = mutationOptions({
  mutationFn: (code: string) => deleteRole(code),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: roleKeys.all });
  }
});

export const toggleRoleEnabledMutation = mutationOptions({
  mutationFn: ({ code, isEnabled }: { code: string; isEnabled: boolean }) =>
    toggleRoleEnabled(code, isEnabled),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: roleKeys.all });
  }
});

export const setRolePermissionsMutation = mutationOptions({
  mutationFn: ({ code, grants }: { code: string; grants: PermissionGrant[] }) =>
    setRolePermissions(code, grants),
  onSuccess: async (_, variables) => {
    const qc = getQueryClient();
    await Promise.all([
      qc.invalidateQueries({ queryKey: roleKeys.all }),
      qc.invalidateQueries({ queryKey: roleKeys.permissions(variables.code) }),
      qc.invalidateQueries({ queryKey: ['account'] }),
      useAuthStore.getState().fetchUserInfo()
    ]);
  }
});

export const setRoleHierarchyMutation = mutationOptions({
  mutationFn: ({ code, juniorRoleCodes }: { code: string; juniorRoleCodes: string[] }) =>
    setRoleHierarchy(code, juniorRoleCodes),
  onSuccess: async (_, variables) => {
    const qc = getQueryClient();
    await Promise.all([
      qc.invalidateQueries({ queryKey: roleKeys.hierarchyAll() }),
      qc.invalidateQueries({ queryKey: roleKeys.hierarchy(variables.code) }),
      qc.invalidateQueries({ queryKey: ['account'] }),
      useAuthStore.getState().fetchUserInfo()
    ]);
  }
});

export const createConstraintMutation = mutationOptions({
  mutationFn: (data: ConstraintPayload) => createConstraint(data),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: roleKeys.constraints() });
  }
});

export const updateConstraintMutation = mutationOptions({
  mutationFn: ({ id, data }: { id: number; data: ConstraintPayload }) => updateConstraint(id, data),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: roleKeys.constraints() });
  }
});

export const deleteConstraintMutation = mutationOptions({
  mutationFn: (id: number) => deleteConstraint(id),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: roleKeys.constraints() });
  }
});
