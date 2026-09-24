import { cn } from '@/lib/utils';
import type * as React from 'react';

interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  action?: React.ReactNode;
  className?: string;
}

export function EmptyState({ icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div className={cn('flex flex-col items-center justify-center py-16 text-center', className)}>
      {icon && (
        <div className='text-muted-foreground/40 mb-3 [&>svg]:h-10 [&>svg]:w-10'>{icon}</div>
      )}
      <p className='text-base font-medium'>{title}</p>
      {description && <p className='text-muted-foreground mt-1 text-sm'>{description}</p>}
      {action && <div className='mt-4'>{action}</div>}
    </div>
  );
}
