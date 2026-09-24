import { describe, expect, it, vi } from 'vitest';
import NotificationsListingPage from './notifications-listing';

const redirect = vi.hoisted(() =>
  vi.fn((url: string) => {
    throw new Error(`NEXT_REDIRECT:${url}`);
  })
);

vi.mock('@tanstack/react-query', () => ({
  HydrationBoundary: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  dehydrate: vi.fn(() => ({}))
}));

vi.mock('next/navigation', () => ({
  redirect
}));

vi.mock('@/lib/query-client', () => ({
  getQueryClient: vi.fn(() => ({
    prefetchQuery: vi.fn().mockRejectedValue(new Error('unauthorized'))
  }))
}));

vi.mock('@/lib/server-auth-headers', () => ({
  getServerAuthHeaders: vi.fn(async () => ({})),
  hasServerAuthHeaders: vi.fn((headers: HeadersInit) => 'Authorization' in headers)
}));

vi.mock('../api/queries', () => ({
  userNotificationsQueryOptions: vi.fn(() => ({ queryKey: ['notifications', 'user'] })),
  unreadCountQueryOptions: vi.fn(() => ({ queryKey: ['notifications', 'unread-count'] }))
}));

vi.mock('./notifications-page', () => ({
  default: () => <div data-testid='notifications-page' />
}));

describe('NotificationsListingPage', () => {
  it('redirects to sign-in when no server auth headers are available', async () => {
    await expect(NotificationsListingPage()).rejects.toThrow(
      'NEXT_REDIRECT:/auth/sign-in?redirect=%2Fdashboard%2Fnotifications'
    );

    expect(redirect).toHaveBeenCalledWith('/auth/sign-in?redirect=%2Fdashboard%2Fnotifications');
  });
});
