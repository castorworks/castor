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

export const createDictTypeMutation = mutationOptions({
  mutationFn: (data: CreateDictTypePayload) => createDictType(data),
  onSuccess: async () => {
    const qc = getQueryClient();
    await Promise.all([
      qc.invalidateQueries({ queryKey: dictKeys.all }),
      qc.invalidateQueries({ queryKey: dictKeys.public })
    ]);
  }
});

export const updateDictTypeMutation = mutationOptions({
  mutationFn: ({ id, values }: { id: number; values: UpdateDictTypePayload }) =>
    updateDictType(id, values),
  onSuccess: async () => {
    const qc = getQueryClient();
    await Promise.all([
      qc.invalidateQueries({ queryKey: dictKeys.all }),
      qc.invalidateQueries({ queryKey: dictKeys.public })
    ]);
  }
});

export const deleteDictTypeMutation = mutationOptions({
  mutationFn: (id: number) => deleteDictType(id),
  onSuccess: async () => {
    const qc = getQueryClient();
    await Promise.all([
      qc.invalidateQueries({ queryKey: dictKeys.all }),
      qc.invalidateQueries({ queryKey: dictKeys.public })
    ]);
  }
});

export const createDictItemMutation = mutationOptions({
  mutationFn: (data: CreateDictItemPayload) => createDictItem(data),
  onSuccess: async (_, variables) => {
    const qc = getQueryClient();
    await Promise.all([
      qc.invalidateQueries({ queryKey: dictKeys.items(variables.typeCode) }),
      qc.invalidateQueries({ queryKey: dictKeys.public })
    ]);
  }
});

export const updateDictItemMutation = mutationOptions({
  mutationFn: ({ id, values }: { id: number; values: UpdateDictItemPayload }) =>
    updateDictItem(id, values),
  onSuccess: async () => {
    const qc = getQueryClient();
    await Promise.all([
      qc.invalidateQueries({ queryKey: dictKeys.all }),
      qc.invalidateQueries({ queryKey: dictKeys.public })
    ]);
  }
});

export const deleteDictItemMutation = mutationOptions({
  mutationFn: ({ id }: { id: number; typeCode: string }) => deleteDictItem(id),
  onSuccess: async (_, variables) => {
    const qc = getQueryClient();
    await Promise.all([
      qc.invalidateQueries({ queryKey: dictKeys.items(variables.typeCode) }),
      qc.invalidateQueries({ queryKey: dictKeys.public })
    ]);
  }
});

export const toggleDictTypeMutation = mutationOptions({
  mutationFn: ({ id, enabled }: { id: number; enabled: boolean }) => toggleDictType(id, enabled),
  onSuccess: async () => {
    const qc = getQueryClient();
    await Promise.all([
      qc.invalidateQueries({ queryKey: dictKeys.all }),
      qc.invalidateQueries({ queryKey: dictKeys.public })
    ]);
  }
});

export const toggleDictItemMutation = mutationOptions({
  mutationFn: ({ id, enabled }: { id: number; enabled: boolean }) => toggleDictItem(id, enabled),
  onSuccess: async () => {
    const qc = getQueryClient();
    await Promise.all([
      qc.invalidateQueries({ queryKey: dictKeys.all }),
      qc.invalidateQueries({ queryKey: dictKeys.public })
    ]);
  }
});
