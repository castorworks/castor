'use client';

import { useLocale } from 'next-intl';
import { useCallback, useMemo } from 'react';
import {
  formatDate,
  formatDateTime,
  formatTime,
  formatLongDate,
  formatShortDate,
  formatMonthName,
  formatMonthYear,
  formatRelativeTime,
  formatNumber
} from '@/lib/format';

/**
 * Hook that provides locale-aware date/number formatting functions.
 * Automatically uses the current locale from next-intl.
 *
 * Usage:
 *   const fmt = useFormat()
 *   fmt.date(createdAt)       // "2026/05/07"
 *   fmt.dateTime(createdAt)   // "2026/05/07 14:30"
 *   fmt.time(createdAt)       // "14:30"
 *   fmt.longDate(createdAt)   // "May 7, 2026"
 *   fmt.shortDate(createdAt)  // "May 07, 2026"
 *   fmt.relativeTime(date)    // "3 minutes ago"
 *   fmt.number(1234)          // "1,234"
 */
export function useFormat() {
  const locale = useLocale();

  const date = useCallback(
    (value: Date | string | number | undefined) => formatDate(value, { locale }),
    [locale]
  );

  const dateTime = useCallback(
    (value: Date | string | number | undefined) => formatDateTime(value, { locale }),
    [locale]
  );

  const time = useCallback(
    (value: Date | string | number | undefined) => formatTime(value, { locale }),
    [locale]
  );

  const longDate = useCallback(
    (value: Date | string | number | undefined) => formatLongDate(value, { locale }),
    [locale]
  );

  const shortDate = useCallback(
    (value: Date | string | number | undefined) => formatShortDate(value, { locale }),
    [locale]
  );

  const monthName = useCallback(
    (value: Date | string | number, style?: 'long' | 'short' | 'narrow') =>
      formatMonthName(value, { locale, style }),
    [locale]
  );

  const monthYear = useCallback(
    (value: Date | string | number) => formatMonthYear(value, { locale }),
    [locale]
  );

  const relativeTime = useCallback(
    (value: Date | string | number | undefined) => formatRelativeTime(value, { locale }),
    [locale]
  );

  const number = useCallback(
    (value: number | undefined, opts?: Intl.NumberFormatOptions) =>
      formatNumber(value, { ...opts, locale }),
    [locale]
  );

  return useMemo(
    () => ({
      date,
      dateTime,
      time,
      longDate,
      shortDate,
      monthName,
      monthYear,
      relativeTime,
      number
    }),
    [date, dateTime, time, longDate, shortDate, monthName, monthYear, relativeTime, number]
  );
}
