import { parseRelativePath } from '@/lib/safe-redirect';

const DASHBOARD_PATH = '/dashboard';

export function getSafeNotificationHref(link: string | null | undefined): string | null {
  const parsed = parseRelativePath(link);
  if (!parsed) return null;

  const isDashboardPath =
    parsed.pathname === DASHBOARD_PATH || parsed.pathname.startsWith(`${DASHBOARD_PATH}/`);
  if (!isDashboardPath) return null;

  return `${parsed.pathname}${parsed.search}${parsed.hash}`;
}
