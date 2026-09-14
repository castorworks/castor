import { mutationOptions } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { createUser, updateUser, deleteUser, addUserRole, removeUserRole } from './service';
import { userKeys } from './queries';
import { roleKeys } from '@/features/roles/api/queries';
import type { CreateUserPayload, UpdateUserPayload } from './types';

export const createUserMutation = mutationOptions({
  mutationFn: (data: CreateUserPayload) => createUser(data),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: userKeys.all });
  }
});

export const updateUserMutation = mutationOptions({
  mutationFn: ({ id, values }: { id: number; values: UpdateUserPayload }) => updateUser(id, values),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: userKeys.all });
  }
});

export const deleteUserMutation = mutationOptions({
  mutationFn: (id: number) => deleteUser(id),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: userKeys.all });
  }
});

export const addUserRoleMutation = mutationOptions({
  mutationFn: ({ username, role }: { username: string; role: string }) =>
    addUserRole(username, role),
  onSuccess: async (_, { username, role }) => {
    const qc = getQueryClient();
    await Promise.all([
      qc.invalidateQueries({ queryKey: userKeys.roles(username) }),
      qc.invalidateQueries({ queryKey: userKeys.permissions(username) }),
      qc.invalidateQueries({ queryKey: roleKeys.users(role) }),
      qc.invalidateQueries({ queryKey: userKeys.all })
    ]);
  }
});

export const removeUserRoleMutation = mutationOptions({
  mutationFn: ({ username, role }: { username: string; role: string }) =>
    removeUserRole(username, role),
  onSuccess: async (_, { username, role }) => {
    const qc = getQueryClient();
    await Promise.all([
      qc.invalidateQueries({ queryKey: userKeys.roles(username) }),
      qc.invalidateQueries({ queryKey: userKeys.permissions(username) }),
      qc.invalidateQueries({ queryKey: roleKeys.users(role) }),
      qc.invalidateQueries({ queryKey: userKeys.all })
    ]);
  }
});
