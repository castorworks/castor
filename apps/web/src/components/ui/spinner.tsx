import { Icons } from '@/components/icons';
import { useTranslations } from 'next-intl';

import { cn } from '@/lib/utils';

function Spinner({ className, ...props }: React.ComponentProps<'svg'>) {
  const t = useTranslations('common');
  return (
    <Icons.spinner
      role='status'
      aria-label={t('loading')}
      className={cn('size-4 animate-spin', className)}
      {...props}
    />
  );
}

export { Spinner };
