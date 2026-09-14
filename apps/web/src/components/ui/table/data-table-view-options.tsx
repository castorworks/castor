'use client';

import type { Table } from '@tanstack/react-table';
import { useTranslations } from 'next-intl';
import { Icons } from '@/components/icons';

import { Button } from '@/components/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList
} from '@/components/ui/command';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { cn } from '@/lib/utils';
import * as React from 'react';
import { CheckIcon, CaretSortIcon } from '@radix-ui/react-icons';

interface DataTableViewOptionsProps<TData> {
  table: Table<TData>;
}

export function DataTableViewOptions<TData>({ table }: DataTableViewOptionsProps<TData>) {
  const t = useTranslations('common');
  const columns = React.useMemo(
    () =>
      table
        .getAllColumns()
        .filter((column) => typeof column.accessorFn !== 'undefined' && column.getCanHide()),
    [table]
  );

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          aria-label={t('toggleColumns')}
          variant='outline'
          size='sm'
          className='ml-auto hidden h-8 lg:flex'
        >
          <Icons.adjustments />
          {t('viewColumns')}
          <CaretSortIcon className='ml-auto opacity-50' />
        </Button>
      </PopoverTrigger>
      <PopoverContent align='end' className='w-44 p-0'>
        <Command>
          <CommandInput placeholder={t('searchColumns')} />
          <CommandList>
            <CommandEmpty>{t('noColumnsFound')}</CommandEmpty>
            <CommandGroup>
              {columns.map((column) => (
                <CommandItem
                  key={column.id}
                  onSelect={() => column.toggleVisibility(!column.getIsVisible())}
                >
                  <span className='truncate'>{column.columnDef.meta?.label ?? column.id}</span>
                  <CheckIcon
                    className={cn(
                      'ml-auto size-4 shrink-0',
                      column.getIsVisible() ? 'opacity-100' : 'opacity-0'
                    )}
                  />
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
