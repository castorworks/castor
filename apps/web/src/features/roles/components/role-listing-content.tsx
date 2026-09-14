'use client';

import { Icons } from '@/components/icons';
import { EmptyState } from '@/components/ui/empty-state';
import { useSuspenseQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { rolesQueryOptions } from '../api/queries';
import { RoleCard } from './role-card';

export function RoleListingContent() {
  const t = useTranslations('roles.empty');
  const { data: roles } = useSuspenseQuery(rolesQueryOptions());

  if (!roles?.length) {
    return <EmptyState icon={<Icons.shield />} title={t('title')} description={t('description')} />;
  }

  return (
    <div className='grid gap-4 md:grid-cols-2 lg:grid-cols-3'>
      {roles.map((role) => (
        <RoleCard key={role.code} role={role} />
      ))}
    </div>
  );
}
