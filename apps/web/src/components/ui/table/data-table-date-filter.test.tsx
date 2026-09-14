import type { Column } from '@tanstack/react-table';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { DataTableDateFilter } from './data-table-date-filter';

vi.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string, values?: Record<string, unknown>) =>
    values ? `${key}:${JSON.stringify(values)}` : key
}));

vi.mock('@/components/icons', () => ({
  Icons: {
    calendar: (props: React.SVGProps<SVGSVGElement>) => <svg data-icon='calendar' {...props} />,
    xCircle: (props: React.SVGProps<SVGSVGElement>) => <svg data-icon='x-circle' {...props} />
  }
}));

describe('DataTableDateFilter', () => {
  it('does not render a nested button when a selected date is visible in the trigger', () => {
    const column = {
      getFilterValue: () => Date.UTC(2026, 4, 20),
      setFilterValue: vi.fn()
    } as unknown as Column<unknown, unknown>;

    const markup = renderToStaticMarkup(<DataTableDateFilter column={column} title='Date' />);

    expect(markup).not.toMatch(/<button\b(?:(?!<\/button>).)*<button\b/s);
  });
});
