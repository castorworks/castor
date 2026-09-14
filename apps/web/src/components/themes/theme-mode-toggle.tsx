'use client';

import { Icons } from '@/components/icons';
import * as React from 'react';

import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { Kbd } from '@/components/ui/kbd';
import { useTheme } from './theme-provider';
import { useTranslations } from 'next-intl';

export function ThemeModeToggle() {
  const { setTheme, resolvedTheme } = useTheme();
  const t = useTranslations('theme');

  const handleThemeToggle = React.useCallback(
    (e?: React.MouseEvent) => {
      const newMode = resolvedTheme === 'dark' ? 'light' : 'dark';
      const root = document.documentElement;

      if (!document.startViewTransition) {
        setTheme(newMode);
        return;
      }

      // Set coordinates from the click event
      if (e) {
        root.style.setProperty('--x', `${e.clientX}px`);
        root.style.setProperty('--y', `${e.clientY}px`);
      }

      document.startViewTransition(() => {
        setTheme(newMode);
      });
    },
    [resolvedTheme, setTheme]
  );

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          variant='secondary'
          size='icon'
          className='group/toggle size-8'
          onClick={handleThemeToggle}
        >
          <Icons.brightness />
          <span className='sr-only'>{t('toggleTheme')}</span>
        </Button>
      </TooltipTrigger>
      <TooltipContent>
        {t('toggleTheme')} <Kbd>D D</Kbd>
      </TooltipContent>
    </Tooltip>
  );
}
