import { describe, expect, it, vi } from 'vitest';
import { renderToString } from 'react-dom/server';

const useQuery = vi.hoisted(() => vi.fn(() => ({ data: undefined })));
const useSuspenseQuery = vi.hoisted(() => vi.fn(() => ({ data: undefined })));
const useMutation = vi.hoisted(() => vi.fn(() => ({ mutate: vi.fn(), isPending: false })));

vi.mock('@tanstack/react-query', () => ({
  useQuery,
  useSuspenseQuery,
  useMutation,
  queryOptions: (options: unknown) => options,
  mutationOptions: (options: unknown) => options
}));

vi.mock('@/components/icons', () => ({
  Icons: {
    notification: (props: React.SVGProps<SVGSVGElement>) => <svg {...props} />,
    trash: (props: React.SVGProps<SVGSVGElement>) => <svg {...props} />
  }
}));

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn() })
}));

vi.mock('next-intl', () => ({
  useTranslations: () => (key: string, values?: Record<string, unknown>) =>
    values ? `${key}:${JSON.stringify(values)}` : key
}));

vi.mock('@/stores/auth-store', () => ({
  useAuthStore: (selector: (state: { user: null; isLoading: boolean }) => unknown) =>
    selector({ user: null, isLoading: true })
}));

vi.mock('@/components/layout/page-container', () => ({
  default: ({ children }: { children: React.ReactNode }) => <div>{children}</div>
}));

vi.mock('@/components/ui/button', () => ({
  Button: ({ children, ...props }: React.ComponentProps<'button'>) => (
    <button {...props}>{children}</button>
  )
}));

vi.mock('@/components/ui/notification-card', () => ({
  NotificationCard: () => <div />
}));

vi.mock('@/components/ui/tabs', () => ({
  Tabs: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsList: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsTrigger: ({ children }: { children: React.ReactNode }) => <button>{children}</button>
}));

vi.mock('@/components/ui/badge', () => ({
  Badge: ({ children }: { children: React.ReactNode }) => <span>{children}</span>
}));

vi.mock('@/components/modal/alert-modal', () => ({
  AlertModal: () => null
}));

vi.mock('../api/mutations', () => ({
  markNotificationReadMutation: { mutationFn: vi.fn() },
  markAllReadMutation: { mutationFn: vi.fn() },
  deleteNotificationMutation: { mutationFn: vi.fn() }
}));

vi.mock('@/lib/mutation-utils', () => ({
  mergeMutationOptions: (base: object, options: object) => ({ ...base, ...options })
}));

vi.mock('@/lib/dict', () => ({
  getDictLabel: (_type: string, value: string) => value,
  getDictVariant: () => 'secondary'
}));

import NotificationsPage from './notifications-page';

describe('NotificationsPage', () => {
  it('does not enable notification queries before auth bootstrap completes', () => {
    renderToString(<NotificationsPage />);

    expect(useQuery).toHaveBeenCalledTimes(2);
    expect(useSuspenseQuery).not.toHaveBeenCalled();
    const calls = useQuery.mock.calls as unknown as Array<[{ enabled: boolean }]>;
    expect(calls[0]?.[0]).toMatchObject({ enabled: false });
    expect(calls[1]?.[0]).toMatchObject({ enabled: false });
  });
});
