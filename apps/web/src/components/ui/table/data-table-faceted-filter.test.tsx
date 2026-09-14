import type { Column } from '@tanstack/react-table';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { DataTableFacetedFilter } from './data-table-faceted-filter';

vi.mock('next-intl', () => ({
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

describe('DataTableFacetedFilter', () => {
  it('does not render a nested button when a selected value is visible in the trigger', () => {
    const column = {
      getFilterValue: () => ['system'],
      setFilterValue: vi.fn()
    } as unknown as Column<unknown, unknown>;

    const markup = renderToStaticMarkup(
      <DataTableFacetedFilter
        column={column}
        title='Type'
        options={[{ label: 'System', value: 'system' }]}
      />
    );

    expect(markup).not.toMatch(/<button\b(?:(?!<\/button>).)*<button\b/s);
  });
});
