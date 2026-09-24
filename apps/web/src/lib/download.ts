import type { DownloadedFile } from '@/lib/api-client';

/** Hand a downloaded file to the browser's save dialog. */
export function saveFile({ blob, filename }: DownloadedFile): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  // Give the browser a moment to start the download before revoking.
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

/** Spreadsheet formats the export and template endpoints produce. */
export type TableFormat = 'xlsx' | 'csv';

/** The viewer's IANA time zone, so exported times match what the page shows. */
export function browserTimeZone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone ?? '';
  } catch {
    return '';
  }
}

/**
 * Turn a list query into an export query: same filters and sort, no paging,
 * plus the format and the viewer's time zone.
 */
export function exportQuery(listQuery: URLSearchParams, format: TableFormat): string {
  const params = new URLSearchParams(listQuery);
  params.delete('page');
  params.delete('pageSize');
  params.set('format', format);
  const tz = browserTimeZone();
  if (tz) params.set('tz', tz);
  return params.toString();
}
