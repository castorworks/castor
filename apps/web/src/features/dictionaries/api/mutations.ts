import { mutationOptions } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import {
  createDictType,
  updateDictType,
  deleteDictType,
  toggleDictType,
  createDictItem,
  updateDictItem,
  deleteDictItem,
  toggleDictItem
} from './service';
import { dictKeys } from './queries';
import type {
  CreateDictTypePayload,
  UpdateDictTypePayload,
  CreateDictItemPayload,
  UpdateDictItemPayload
} from './types';

/**
 * `dictKeys.all` is a prefix of every dictionary key, so one invalidation covers
 * the admin lists and the `enabled` map that renders labels across the dashboard.
 */
async function invalidateDicts() {
  await getQueryClient().invalidateQueries({ queryKey: dictKeys.all });
}

export const createDictTypeMutation = mutationOptions({
  mutationFn: (data: CreateDictTypePayload) => createDictType(data),
  onSuccess: invalidateDicts
});

export const updateDictTypeMutation = mutationOptions({
  mutationFn: ({ id, values }: { id: number; values: UpdateDictTypePayload }) =>
    updateDictType(id, values),
  onSuccess: invalidateDicts
});

export const deleteDictTypeMutation = mutationOptions({
  mutationFn: (id: number) => deleteDictType(id),
  onSuccess: invalidateDicts
});

export const createDictItemMutation = mutationOptions({
  mutationFn: (data: CreateDictItemPayload) => createDictItem(data),
  onSuccess: invalidateDicts
});

export const updateDictItemMutation = mutationOptions({
  mutationFn: ({ id, values }: { id: number; values: UpdateDictItemPayload }) =>
    updateDictItem(id, values),
  onSuccess: invalidateDicts
});

export const deleteDictItemMutation = mutationOptions({
  mutationFn: ({ id }: { id: number; typeCode: string }) => deleteDictItem(id),
  onSuccess: invalidateDicts
});

export const toggleDictTypeMutation = mutationOptions({
  mutationFn: ({ id, enabled }: { id: number; enabled: boolean }) => toggleDictType(id, enabled),
  onSuccess: invalidateDicts
});

export const toggleDictItemMutation = mutationOptions({
  mutationFn: ({ id, enabled }: { id: number; enabled: boolean }) => toggleDictItem(id, enabled),
  onSuccess: invalidateDicts
});
