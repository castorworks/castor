'use client';
import { Badge } from '@/components/ui/badge';
import { DataTableColumnHeader } from '@/components/ui/table/data-table-column-header';
import type { User } from '../../api/types';
import { Column, ColumnDef } from '@tanstack/react-table';
import { Icons } from '@/components/icons';
import { formatDateTime } from '@/lib/format';
import { CellAction } from './cell-action';
import { useDict } from '@/hooks/use-dict';
import { useTranslations, useLocale } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { usePermission } from '@/hooks/use-permission';
import { departmentsQueryOptions } from '@/features/departments/api/queries';
import { flattenDepartments, indentedLabel } from '@/features/departments/tree';

export const userColumnIds = [
  'search',
  'contact',
  'department',
  'accountSource',
  'status',
  'createdAt',
  'actions'
];

/** One contact line: value plus a verified/unverified marker */
function ContactLine({
  icon: Icon,
  value,
  verified,
  verifiedLabel,
  unverifiedLabel
}: {
  icon: (typeof Icons)['mail'];
  value: string;
  verified: boolean;
  verifiedLabel: string;
  unverifiedLabel: string;
}) {
  return (
    <span className='flex items-center gap-1.5'>
      <Icon className='text-muted-foreground size-3.5 shrink-0' />
      <span className='truncate'>{value}</span>
      {verified ? (
        <Icons.circleCheck className='text-success size-3.5 shrink-0' aria-label={verifiedLabel} />
      ) : (
        <Icons.alertCircle
          className='text-warning size-3.5 shrink-0'
          aria-label={unverifiedLabel}
        />
      )}
    </span>
  );
}

export function useUserColumns(): ColumnDef<User>[] {
  const canListDepartments = usePermission('/api/v1/admin/departments:GET');
  const { data: departments = [] } = useQuery({
    ...departmentsQueryOptions(),
    enabled: canListDepartments
  });
  const departmentNames = new Map(departments.map((d) => [d.id, d.name]));
  const departmentOptions = flattenDepartments(departments).map(({ department, depth }) => ({
    value: String(department.id),
    label: indentedLabel(department.name, depth)
  }));
  const t = useTranslations('users');
  const locale = useLocale();
  const dict = useDict();

  return [
    {
      id: 'search',
      accessorFn: (row) => row.name || row.username,
      header: ({ column }: { column: Column<User, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('table.user')} />
      ),
      cell: ({ row }) => (
        <div className='flex items-center gap-3'>
          <div className='flex flex-col'>
            <span className='font-medium'>{row.original.name || row.original.username}</span>
            <span className='text-muted-foreground flex items-center gap-1 text-xs'>
              @{row.original.username}
              {row.original.totpEnabled && (
                <Badge variant='outline' className='px-1 py-0' title={t('twoFactor.enabledHint')}>
                  2FA
                </Badge>
              )}
            </span>
          </div>
        </div>
      ),
      meta: {
        label: t('table.user'),
        placeholder: t('table.searchUsers'),
        variant: 'text' as const,
        icon: Icons.text
      },
      enableColumnFilter: true
    },
    {
      id: 'contact',
      header: ({ column }: { column: Column<User, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('table.contact')} />
      ),
      cell: ({ row }) => {
        const { email, mobile, emailVerified, mobileVerified } = row.original;
        if (!email && !mobile) {
          return <span className='text-muted-foreground text-sm'>{t('table.noContact')}</span>;
        }
        return (
          <div className='flex max-w-56 flex-col gap-0.5 text-sm'>
            {email && (
              <ContactLine
                icon={Icons.mail}
                value={email}
                verified={emailVerified}
                verifiedLabel={t('table.verified')}
                unverifiedLabel={t('table.unverified')}
              />
            )}
            {mobile && (
              <ContactLine
                icon={Icons.phone}
                value={mobile}
                verified={mobileVerified}
                verifiedLabel={t('table.verified')}
                unverifiedLabel={t('table.unverified')}
              />
            )}
          </div>
        );
      }
    },
    {
      id: 'department',
      accessorKey: 'departmentId',
      header: t('table.department'),
      cell: ({ row }) => {
        const id = row.original.departmentId;
        return (
          <span className='text-sm'>
            {id === null ? '—' : (departmentNames.get(id) ?? `#${id}`)}
          </span>
        );
      },
      enableSorting: false,
      enableColumnFilter: canListDepartments,
      meta: {
        label: t('table.department'),
        variant: 'select' as const,
        options: departmentOptions
      }
    },
    {
      id: 'accountSource',
      accessorKey: 'accountSource',
      header: ({ column }: { column: Column<User, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('table.source')} />
      ),
      cell: ({ row }) => {
        const source = row.original.accountSource;
        return <span className='text-sm'>{dict.label('account_source', source)}</span>;
      },
      enableColumnFilter: true,
      meta: {
        label: t('table.source'),
        variant: 'multiSelect' as const,
        get options() {
          return dict.options('account_source');
        }
      }
    },
    {
      id: 'status',
      header: ({ column }: { column: Column<User, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('table.status')} />
      ),
      cell: ({ row }) => {
        const { enable, locked } = row.original;
        if (!enable) {
          return <Badge variant='destructive'>{t('table.disabled')}</Badge>;
        }
        if (locked) {
          return <Badge variant='secondary'>{t('table.locked')}</Badge>;
        }
        return <Badge variant='default'>{t('table.active')}</Badge>;
      },
      enableColumnFilter: true,
      meta: {
        label: t('table.status'),
        variant: 'select' as const,
        options: [
          { value: 'active', label: t('table.active') },
          { value: 'disabled', label: t('table.disabled') },
          { value: 'locked', label: t('table.locked') }
        ]
      }
    },
    {
      accessorKey: 'createdAt',
      header: ({ column }: { column: Column<User, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('table.created')} />
      ),
      cell: ({ row }) => {
        const date = formatDateTime(row.original.createdAt, { locale });
        return (
          <span className='text-sm' suppressHydrationWarning>
            {date}
          </span>
        );
      }
    },
    {
      id: 'actions',
      cell: ({ row }) => <CellAction data={row.original} />
    }
  ];
}
