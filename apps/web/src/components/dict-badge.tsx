'use client';

import { Badge } from '@/components/ui/badge';
import { Icons } from '@/components/icons';
import { useDict } from '@/hooks/use-dict';
import type { DictTypeCode } from '@/lib/dict';
import { tagColorBg } from '@/lib/tag-color';
import { cn } from '@/lib/utils';

interface DictBadgeProps {
  type: DictTypeCode;
  value: string;
  /** Show the item's icon instead of its color dot. */
  showIcon?: boolean;
  className?: string;
}

/**
 * An enum value rendered from its dictionary item: localized label plus the
 * item's tag color as a dot.
 *
 * A dot on a neutral badge, rather than a filled badge, because tag colors are
 * data: all eight must stay distinguishable and legible in every theme and in
 * dark mode, which no fixed foreground color achieves on `--tag-yellow` and
 * `--tag-blue` alike. Badge variants (`success`, `destructive`, …) are for
 * states the code knows about, not for operator-chosen colors.
 */
export function DictBadge({ type, value, showIcon = false, className }: DictBadgeProps) {
  const dict = useDict();
  const color = tagColorBg(dict.color(type, value));
  const iconKey = dict.icon(type, value);
  const Icon = showIcon && iconKey ? Icons[iconKey as keyof typeof Icons] : undefined;

  return (
    <Badge variant='outline' className={className}>
      {Icon ? (
        <Icon />
      ) : (
        color && <span aria-hidden className={cn('size-2 shrink-0 rounded-full', color)} />
      )}
      {dict.label(type, value)}
    </Badge>
  );
}
