import PageContainer from '@/components/layout/page-container';
import UserListingPage from '@/features/users/components/user-listing';
import { searchParamsCache } from '@/lib/searchparams';
import type { SearchParams } from 'nuqs/server';
import { getUsersInfoContent } from '@/features/users/info-content';
import { UserFormSheetTrigger } from '@/features/users/components/user-form-sheet';
import { serverHasPermission } from '@/lib/server-auth-headers';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations();
  return {
    title: t('nav.users')
  };
}

type PageProps = {
  searchParams: Promise<SearchParams>;
};

export default async function UsersPage(props: PageProps) {
  const searchParams = await props.searchParams;
  searchParamsCache.parse(searchParams);
  const canViewUsers = await serverHasPermission('/api/v1/admin/users:GET');
  const canManageUsers = await serverHasPermission('/api/v1/admin/users:POST');
  const t = await getTranslations();

  return (
    <PageContainer
      pageTitle={t('nav.users')}
      pageDescription={t('dashboard.usersDescription')}
      infoContent={getUsersInfoContent(t)}
      access={canViewUsers}
      accessDeniedMessage={t('common.accessDenied')}
      pageHeaderAction={canManageUsers ? <UserFormSheetTrigger /> : undefined}
    >
      <UserListingPage />
    </PageContainer>
  );
}
