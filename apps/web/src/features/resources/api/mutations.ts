import { mutationOptions } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { createResource, updateResource, deleteResource } from './service';
import { resourceKeys } from './queries';
import type { CreateResourcePayload, UpdateResourcePayload } from './types';

export const createResourceMutation = mutationOptions({
  mutationFn: (data: CreateResourcePayload) => createResource(data),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: resourceKeys.all });
  }
});

export const updateResourceMutation = mutationOptions({
  mutationFn: ({ id, values }: { id: number; values: UpdateResourcePayload }) =>
    updateResource(id, values),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: resourceKeys.all });
  }
});

export const deleteResourceMutation = mutationOptions({
  mutationFn: (id: number) => deleteResource(id),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: resourceKeys.all });
  }
});
