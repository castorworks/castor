import KBar from '@/components/kbar';
import AppSidebar from '@/components/layout/app-sidebar';
import Header from '@/components/layout/header';
import { InfoSidebar } from '@/components/layout/info-sidebar';
import { InfobarProvider } from '@/components/ui/infobar';
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar';
import { DictProvider } from '@/components/layout/dict-provider';
import { enabledDictsQueryOptions } from '@/features/dictionaries/api/queries';
import { getQueryClient } from '@/lib/query-client';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';
import { HydrationBoundary, dehydrate } from '@tanstack/react-query';
import type { Metadata } from 'next';
import { cookies } from 'next/headers';

// 每个页面用 generateMetadata 给出当前语言的标题；这里只放与语言无关的项。
export const metadata: Metadata = {
  robots: {
    index: false,
    follow: false
  }
};

export default async function DashboardLayout({ children }: { children: React.ReactNode }) {
  // Persisting the sidebar state in the cookie.
  const cookieStore = await cookies();
  const defaultOpen = cookieStore.get('sidebar_state')?.value === 'true';

  // Dictionaries label enum values on nearly every dashboard page. Prefetching here
  // puts them in the first HTML; without it badges flash raw values like `ACTIVE`.
  const queryClient = getQueryClient();
  try {
    await queryClient.prefetchQuery(
      enabledDictsQueryOptions({ headers: await getServerAuthHeaders() })
    );
  } catch {
    // Prefetch failed — DictProvider retries on the client with a refreshed token
  }

  return (
    <KBar>
      <HydrationBoundary state={dehydrate(queryClient)}>
        <DictProvider>
          <SidebarProvider defaultOpen={defaultOpen}>
            <AppSidebar />
            <SidebarInset>
              <Header />
              <InfobarProvider defaultOpen={false}>
                {children}
                <InfoSidebar side='right' />
              </InfobarProvider>
            </SidebarInset>
          </SidebarProvider>
        </DictProvider>
      </HydrationBoundary>
    </KBar>
  );
}
