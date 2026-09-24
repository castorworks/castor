import { mutationOptions } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import {
  createAsset,
  updateAsset,
  updateAssetStatus,
  moveAsset,
  deleteAsset,
  batchDeleteAssets,
  batchUpdateAssetStatus
} from './service';
import { assetKeys } from './queries';
import type {
  UpdateAssetPayload,
  UpdateAssetStatusPayload,
  MoveAssetPayload,
  BatchDeletePayload,
  BatchStatusPayload
} from './types';

export const createAssetMutation = mutationOptions({
  mutationFn: (formData: FormData) => createAsset(formData),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: assetKeys.all });
  }
});

export const updateAssetMutation = mutationOptions({
  mutationFn: ({ id, values }: { id: number; values: UpdateAssetPayload }) =>
    updateAsset(id, values),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: assetKeys.all });
  }
});

export const updateAssetStatusMutation = mutationOptions({
  mutationFn: ({ id, values }: { id: number; values: UpdateAssetStatusPayload }) =>
    updateAssetStatus(id, values),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: assetKeys.all });
  }
});

export const moveAssetMutation = mutationOptions({
  mutationFn: ({ id, values }: { id: number; values: MoveAssetPayload }) => moveAsset(id, values),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: assetKeys.all });
  }
});

export const deleteAssetMutation = mutationOptions({
  mutationFn: (id: number) => deleteAsset(id),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: assetKeys.all });
  }
});

export const batchDeleteAssetsMutation = mutationOptions({
  mutationFn: (data: BatchDeletePayload) => batchDeleteAssets(data),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: assetKeys.all });
  }
});

export const batchUpdateAssetStatusMutation = mutationOptions({
  mutationFn: (data: BatchStatusPayload) => batchUpdateAssetStatus(data),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: assetKeys.all });
  }
});
