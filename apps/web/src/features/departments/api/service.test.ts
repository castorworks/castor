import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/lib/api-client';
import { deleteDepartment, getDepartments, saveDepartment } from './service';

vi.mock('@/lib/api-client', () => ({
  apiClient: vi.fn()
}));

const mockedApiClient = vi.mocked(apiClient);

describe('departments service', () => {
  beforeEach(() => {
    mockedApiClient.mockReset();
  });

  it('reads the flat department list and tolerates an empty tree', async () => {
    mockedApiClient.mockResolvedValueOnce([{ id: 1, name: 'HQ' }]);
    expect(await getDepartments()).toEqual([{ id: 1, name: 'HQ' }]);
    mockedApiClient.mockResolvedValueOnce(null);
    expect(await getDepartments()).toEqual([]);
    expect(mockedApiClient).toHaveBeenCalledWith('/v1/admin/departments', undefined);
  });

  it('creates with POST and updates with PUT', async () => {
    const data = { parentId: null, code: 'hq', name: 'HQ', sortOrder: 0, isEnabled: true };
    mockedApiClient.mockResolvedValue(undefined as never);
    await saveDepartment(undefined, data);
    expect(mockedApiClient).toHaveBeenLastCalledWith('/v1/admin/departments', {
      method: 'POST',
      body: JSON.stringify(data)
    });
    await saveDepartment(3, data);
    expect(mockedApiClient).toHaveBeenLastCalledWith('/v1/admin/departments/3', {
      method: 'PUT',
      body: JSON.stringify(data)
    });
    await deleteDepartment(3);
    expect(mockedApiClient).toHaveBeenLastCalledWith('/v1/admin/departments/3', {
      method: 'DELETE'
    });
  });
});
