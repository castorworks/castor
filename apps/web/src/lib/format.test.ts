import { describe, expect, it } from 'vitest';
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
} from './format';

describe('formatDate', () => {
  it('formats dates as YYYY/MM/DD regardless of locale', () => {
    expect(formatDate('2026-05-05T23:30:00Z', { locale: 'en' })).toBe('2026/05/06');
  });

  it('formats dates in zh locale', () => {
    const result = formatDate('2026-05-05T23:30:00Z', { locale: 'zh' });
    expect(result).toBe('2026/05/06');
  });

  it('returns empty string for undefined', () => {
    expect(formatDate(undefined)).toBe('');
  });

  it('uses Asia/Shanghai timezone by default', () => {
    // 23:30 UTC on May 5 is 07:30 on May 6 in Beijing time.
    const result = formatDate('2026-05-05T23:30:00Z', { locale: 'en' });
    expect(result).toBe('2026/05/06');
  });

  it('allows overriding timezone to UTC', () => {
    const result = formatDate('2026-05-05T23:30:00Z', { locale: 'en', timeZone: 'UTC' });
    expect(result).toBe('2026/05/05');
  });
});

describe('formatDateTime', () => {
  it('formats date + time', () => {
    const result = formatDateTime('2026-05-05T14:30:00Z', { locale: 'zh' });
    expect(result).toContain('2026');
    expect(result).toContain('22:30');
  });

  it('allows overriding timezone to UTC', () => {
    const result = formatDateTime('2026-05-05T14:30:00Z', { locale: 'zh', timeZone: 'UTC' });
    expect(result).toBe('2026/05/05 14:30');
  });

  it('returns empty string for undefined', () => {
    expect(formatDateTime(undefined)).toBe('');
  });
});

describe('formatTime', () => {
  it('formats time in 24h format', () => {
    const result = formatTime('2026-05-05T14:30:00Z', { locale: 'en' });
    expect(result).toBe('22:30');
  });

  it('allows overriding timezone to UTC', () => {
    const result = formatTime('2026-05-05T14:30:00Z', { locale: 'en', timeZone: 'UTC' });
    expect(result).toBe('14:30');
  });

  it('returns empty string for undefined', () => {
    expect(formatTime(undefined)).toBe('');
  });
});

describe('formatLongDate', () => {
  it('returns long date format in en', () => {
    const result = formatLongDate('2026-01-15T00:00:00Z', { locale: 'en' });
    expect(result).toContain('January');
    expect(result).toContain('2026');
    expect(result).toContain('15');
  });

  it('returns empty string for undefined', () => {
    expect(formatLongDate(undefined)).toBe('');
  });
});

describe('formatShortDate', () => {
  it('returns short date format in en', () => {
    const result = formatShortDate('2026-01-01T00:00:00Z', { locale: 'en' });
    expect(result).toContain('Jan');
    expect(result).toContain('2026');
  });

  it('returns empty string for undefined', () => {
    expect(formatShortDate(undefined)).toBe('');
  });
});

describe('formatMonthName', () => {
  it('returns month name in en', () => {
    const result = formatMonthName(new Date(2026, 0, 1), { locale: 'en' });
    expect(result).toBe('January');
  });

  it('returns short month name', () => {
    const result = formatMonthName(new Date(2026, 0, 1), { locale: 'en', style: 'short' });
    expect(result).toBe('Jan');
  });
});

describe('formatMonthYear', () => {
  it('returns month and year in en', () => {
    const result = formatMonthYear(new Date(2026, 0, 1), { locale: 'en' });
    expect(result).toBe('January 2026');
  });
});

describe('formatRelativeTime', () => {
  it('returns empty string for undefined', () => {
    expect(formatRelativeTime(undefined)).toBe('');
  });
});

describe('formatNumber', () => {
  it('formats numbers with locale grouping', () => {
    expect(formatNumber(1234, { locale: 'en' })).toBe('1,234');
  });

  it('formats large numbers', () => {
    expect(formatNumber(1234567, { locale: 'en' })).toBe('1,234,567');
  });

  it('returns empty string for undefined', () => {
    expect(formatNumber(undefined)).toBe('');
  });
});
