import { getCoreRowModel, getPaginationRowModel, useReactTable } from '@tanstack/react-table';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { DataTablePagination } from './data-table-pagination';

vi.mock('next-intl', () => ({
  useTranslations: () => (key: string, values?: Record<string, unknown>) =>
    values ? `${key}:${JSON.stringify(values)}` : key
}));

// 服务端分页只拿到当前页的行：页脚的总行数和页数必须来自接口返回的 total（rowCount），
// 不能数当前页有几行（否则 21 条、3 页会显示成"共 10 行、共 3 页"）。
function ServerPagedTable({ rowCount }: { rowCount: number }) {
  const table = useReactTable({
    data: Array.from({ length: 10 }, (_, id) => ({ id })),
    columns: [{ accessorKey: 'id' }],
    rowCount,
    state: { pagination: { pageIndex: 0, pageSize: 10 } },
    manualPagination: true,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel()
  });
  return <DataTablePagination table={table} />;
}

describe('DataTablePagination', () => {
  it('shows the server total rather than the rows on the current page', () => {
    const markup = renderToStaticMarkup(<ServerPagedTable rowCount={21} />);
    expect(markup).toContain('rowsTotal:{&quot;total&quot;:21}');
    expect(markup).toContain('pageOf:{&quot;current&quot;:1,&quot;total&quot;:3}');
  });
});
