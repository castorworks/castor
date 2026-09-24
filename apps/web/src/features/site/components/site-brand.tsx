import Link from 'next/link';
import { Icons } from '@/components/icons';
import { cn } from '@/lib/utils';

export function SiteBrand({
  name,
  className,
  nameClassName
}: {
  name: string;
  className?: string;
  nameClassName?: string;
}) {
  return (
    <Link href='/' className={cn('flex min-w-0 items-center gap-2 font-semibold', className)}>
      <Icons.logo className='text-primary size-8 shrink-0' />
      <span className={cn('truncate', nameClassName)}>{name}</span>
    </Link>
  );
}
