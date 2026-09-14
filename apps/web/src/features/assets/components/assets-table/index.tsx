'use client';

import { DataTable } from '@/components/ui/table/data-table';
import { DataTableToolbar } from '@/components/ui/table/data-table-toolbar';
import { useDataTable } from '@/hooks/use-data-table';
import { useMutation, useSuspenseQuery } from '@tanstack/react-query';
import { parseAsInteger, parseAsString, useQueryStates } from 'nuqs';
import { getSortingStateParser } from '@/lib/parsers';
import { assetsQueryOptions, assetStatsQueryOptions } from '../../api/queries';
import { batchDeleteAssetsMutation, batchUpdateAssetStatusMutation } from '../../api/mutations';
import { useAssetColumns, columnIds } from './columns';
import { Button } from '@/components/ui/button';
import { Icons } from '@/components/icons';
import { toast } from 'sonner';
import { AssetsStatsCards } from '../assets-stats-cards';
import type { Asset, AssetStatus } from '../../api/types';
import { useTranslations } from 'next-intl';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { useAllPermissions } from '@/hooks/use-permission';

export function AssetsTable() {
  const t = useTranslations('assets');
  const tc = useTranslations('common');
  const columns = useAssetColumns();
  const canBatchDelete = useAllPermissions(['/api/v1/admin/assets/batch/delete:POST']);
  const canBatchUpdateStatus = useAllPermissions(['/api/v1/admin/assets/batch/status:PUT']);
  const [params] = useQueryStates({
    page: parseAsInteger.withDefault(1),
    perPage: parseAsInteger.withDefault(10),
    asset: parseAsString,
    category: parseAsString,
    status: parseAsString,
    sort: getSortingStateParser(columnIds).withDefault([])
  });

  const filters = {
    page: params.page,
    pageSize: params.perPage,
    ...(params.asset && { search: params.asset }),
    ...(params.category && { category: params.category }),
    ...(params.status && { status: params.status }),
    ...(params.sort.length > 0 && { sort: JSON.stringify(params.sort) })
  };

  const { data } = useSuspenseQuery(assetsQueryOptions(filters));
  useSuspenseQuery(assetStatsQueryOptions());

  const pageCount = data.totalPages;

  const { table } = useDataTable<Asset>({
    data: data.list,
    columns,
    pageCount,
    shallow: true,
    debounceMs: 500,
    initialState: {
      columnPinning: { right: ['actions_col'] }
    }
  });

  // Batch delete mutation
  const batchDelete = useMutation({
    ...mergeMutationOptions(batchDeleteAssetsMutation, {
      onSuccess: () => {
        toast.success(t('messages.bulkDeleteSuccess'));
      },
      onError: () => {
        toast.error(t('messages.bulkDeleteFailed'));
      }
    })
  });

  // Batch status update mutation
  const batchStatus = useMutation({
    ...mergeMutationOptions(batchUpdateAssetStatusMutation, {
      onSuccess: () => {
        toast.success(t('messages.statusUpdated'));
      },
      onError: () => {
        toast.error(t('messages.statusUpdateFailed'));
      }
    })
  });

  const selectedCount = table.getFilteredSelectedRowModel().rows.length;

  const getSelectedIds = () => table.getFilteredSelectedRowModel().rows.map((r) => r.original.id);

  const handleBatchDelete = () => {
    const ids = getSelectedIds();
    if (ids.length === 0) return;
    batchDelete.mutate({ ids });
  };

  const handleBatchStatus = (status: AssetStatus) => {
    const ids = getSelectedIds();
    if (ids.length === 0) return;
    batchStatus.mutate({ ids, status });
  };

  const actionBar =
    canBatchDelete || canBatchUpdateStatus ? (
      <div className='flex items-center gap-2 rounded-lg border bg-muted/50 p-2'>
        <span className='text-muted-foreground text-sm'>
          {tc('selected', { count: selectedCount })}
        </span>
        <div className='flex items-center gap-1'>
          {canBatchUpdateStatus && (
            <>
              <Button
                variant='outline'
                size='sm'
                onClick={() => handleBatchStatus('ACTIVE')}
                disabled={batchStatus.isPending}
              >
                {t('table.setActive')}
              </Button>
              <Button
                variant='outline'
                size='sm'
                onClick={() => handleBatchStatus('ARCHIVED')}
                disabled={batchStatus.isPending}
              >
                {t('table.setArchived')}
              </Button>
            </>
          )}
          {canBatchDelete && (
            <Button
              variant='outline'
              size='sm'
              onClick={() => handleBatchDelete()}
              disabled={batchDelete.isPending}
              className='text-destructive'
            >
              <Icons.trash className='mr-1.5 h-3.5 w-3.5' />
              {tc('delete')}
            </Button>
          )}
        </div>
      </div>
    ) : undefined;

  return (
    <div className='flex flex-1 flex-col gap-4'>
      <AssetsStatsCards />
      <DataTable table={table} actionBar={actionBar}>
        <DataTableToolbar table={table} />
      </DataTable>
    </div>
  );
}

export function AssetsTableSkeleton() {
  return (
    <div className='flex flex-1 animate-pulse flex-col gap-4'>
      <div className='bg-muted h-10 w-full rounded' />
      <div className='bg-muted h-96 w-full rounded-lg' />
      <div className='bg-muted h-10 w-full rounded' />
    </div>
  );
}
