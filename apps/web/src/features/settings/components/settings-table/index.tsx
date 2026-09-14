'use client';

import { useSuspenseQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { settingsQueryOptions } from '../../api/queries';
import type { Setting } from '../../api/types';
import { SettingCard } from './setting-card';

const CATEGORY_ORDER: string[] = ['GENERAL', 'FEATURE', 'SECURITY', 'EMAIL', 'STORAGE', 'CUSTOM'];

function groupByCategory(settings: Setting[]): Map<string, Setting[]> {
  const groups = new Map<string, Setting[]>();
  for (const setting of settings) {
    const cat = setting.category || 'GENERAL';
    if (!groups.has(cat)) {
      groups.set(cat, []);
    }
    groups.get(cat)!.push(setting);
  }
  return groups;
}

export function SettingsTable() {
  const { data: settings } = useSuspenseQuery(settingsQueryOptions({}));

  const groupedSettings = useMemo(() => {
    const groups = groupByCategory(settings);
    const sorted = new Map<string, Setting[]>();

    // Add categories in defined order
    for (const cat of CATEGORY_ORDER) {
      if (groups.has(cat)) {
        sorted.set(cat, groups.get(cat)!);
        groups.delete(cat);
      }
    }

    // Add remaining categories
    for (const [cat, items] of groups) {
      sorted.set(cat, items);
    }

    return sorted;
  }, [settings]);

  return (
    <div className='flex flex-col gap-6'>
      {Array.from(groupedSettings.entries()).map(([category, items]) => (
        <div key={category} className='flex flex-col gap-3'>
          <h3 className='text-lg font-semibold'>{category}</h3>
          <div className='grid gap-4 md:grid-cols-2 lg:grid-cols-3'>
            {items.map((setting) => (
              <SettingCard key={setting.id} setting={setting} />
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

export function SettingsTableSkeleton() {
  return (
    <div className='flex flex-1 animate-pulse flex-col gap-4'>
      <div className='bg-muted h-10 w-full rounded' />
      <div className='bg-muted h-96 w-full rounded-lg' />
    </div>
  );
}
