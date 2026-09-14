'use client';

import { useEffect, useMemo, useState } from 'react';
import { useQuery, useSuspenseQuery } from '@tanstack/react-query';
import { dictTypesQueryOptions, dictItemsQueryOptions } from '../api/queries';
import { DictTypeList } from './dict-type-list';
import { DictItemList } from './dict-item-list';
import { Icons } from '@/components/icons';
import { useTranslations } from 'next-intl';
import { usePermission } from '@/hooks/use-permission';

export function DictionaryManager() {
  const t = useTranslations('dictionary.manager');
  const { data: dictTypes } = useSuspenseQuery(dictTypesQueryOptions());
  const [selectedTypeId, setSelectedTypeId] = useState<number | null>(dictTypes[0]?.id ?? null);
  const canCreateType = usePermission('/api/v1/admin/dict-types:POST');
  const canUpdateType = usePermission('/api/v1/admin/dict-types/:id:PUT');
  const canDeleteType = usePermission('/api/v1/admin/dict-types/:id:DELETE');
  const canCreateItem = usePermission('/api/v1/admin/dict-items:POST');
  const canUpdateItem = usePermission('/api/v1/admin/dict-items/:id:PUT');
  const canDeleteItem = usePermission('/api/v1/admin/dict-items/:id:DELETE');

  const selectedType = useMemo(
    () => dictTypes.find((type) => type.id === selectedTypeId) ?? null,
    [dictTypes, selectedTypeId]
  );

  useEffect(() => {
    if (selectedTypeId && dictTypes.some((type) => type.id === selectedTypeId)) {
      return;
    }
    setSelectedTypeId(dictTypes[0]?.id ?? null);
  }, [dictTypes, selectedTypeId]);

  const { data: dictItems } = useQuery({
    ...dictItemsQueryOptions(selectedType?.code ?? ''),
    enabled: !!selectedType?.code
  });

  return (
    <div className='grid h-[calc(100vh-12rem)] grid-cols-[280px_1fr] gap-6 lg:grid-cols-[320px_1fr]'>
      {/* Left panel: Dictionary types */}
      <DictTypeList
        types={dictTypes}
        selectedType={selectedType}
        onSelect={(type) => setSelectedTypeId(type?.id ?? null)}
        canCreate={canCreateType}
        canUpdate={canUpdateType}
        canDelete={canDeleteType}
      />

      {/* Right panel: Dictionary items */}
      {selectedType ? (
        <DictItemList
          type={selectedType}
          items={dictItems ?? []}
          canCreate={canCreateItem}
          canUpdate={canUpdateItem}
          canDelete={canDeleteItem}
        />
      ) : (
        <div className='flex flex-col items-center justify-center rounded-lg border border-dashed'>
          <Icons.code className='mb-3 h-10 w-10 text-muted-foreground/40' />
          <p className='text-muted-foreground text-sm'>{t('selectType')}</p>
        </div>
      )}
    </div>
  );
}
