import type { Column } from '@tanstack/react-table';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { DataTableSliderFilter } from './data-table-slider-filter';

vi.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string, values?: Record<string, unknown>) =>
    values ? `${key}:${JSON.stringify(values)}` : key
}));

vi.mock('@/components/icons', () => ({
  Icons: {
    plusCircle: (props: React.SVGProps<SVGSVGElement>) => (
      <svg data-icon='plus-circle' {...props} />
    ),
    xCircle: (props: React.SVGProps<SVGSVGElement>) => <svg data-icon='x-circle' {...props} />
  }
}));

describe('DataTableSliderFilter', () => {
  it('does not render a nested button when a selected range is visible in the trigger', () => {
    const column = {
      columnDef: { meta: { range: [0, 100] } },
      getFacetedMinMaxValues: () => [0, 100],
      getFilterValue: () => [10, 20],
      setFilterValue: vi.fn()
    } as unknown as Column<unknown, unknown>;

    const markup = renderToStaticMarkup(<DataTableSliderFilter column={column} title='Range' />);

    expect(markup).not.toMatch(/<button\b(?:(?!<\/button>).)*<button\b/s);
  });
});
