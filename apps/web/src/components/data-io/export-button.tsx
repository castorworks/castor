'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu';
import { Icons } from '@/components/icons';
import type { DownloadedFile } from '@/lib/api-client';
import { saveFile, type TableFormat } from '@/lib/download';

interface ExportButtonProps {
  /** Download the list with its current filters in the chosen format. */
  onExport: (format: TableFormat) => Promise<DownloadedFile>;
}

/** Export the current list (filters and sort applied) as Excel or CSV. */
export function ExportButton({ onExport }: ExportButtonProps) {
  const t = useTranslations('dataIO');
  const [pending, setPending] = useState(false);

  const run = async (format: TableFormat) => {
    setPending(true);
    try {
      saveFile(await onExport(format));
    } catch (error) {
      toast.error(error instanceof Error && error.message ? error.message : t('exportFailed'));
    } finally {
      setPending(false);
    }
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant='outline' size='sm' className='h-8' isLoading={pending} disabled={pending}>
          <Icons.download /> {t('export')}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='end'>
        <DropdownMenuItem onSelect={() => run('xlsx')}>{t('excel')}</DropdownMenuItem>
        <DropdownMenuItem onSelect={() => run('csv')}>{t('csv')}</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
