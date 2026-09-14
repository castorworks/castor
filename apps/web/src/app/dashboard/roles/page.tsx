import PageContainer from '@/components/layout/page-container';
import RoleListingPage from '@/features/roles/components/role-listing';
import type { SearchParams } from 'nuqs/server';
import { RoleFormSheetTrigger } from '@/features/roles/components/role-form-sheet';
import { serverHasPermission } from '@/lib/server-auth-headers';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations();
  return {
    title: t('nav.roles')
  };
}

type PageProps = {
  searchParams: Promise<SearchParams>;
};

export default async function RolesPage(props: PageProps) {
  await props.searchParams;
  const canViewRoles = await serverHasPermission('/api/v1/admin/roles:GET');
  const canManageRoles = await serverHasPermission('/api/v1/admin/roles:POST');
  const t = await getTranslations();

  return (
    <PageContainer
      pageTitle={t('nav.roles')}
      pageDescription={t('dashboard.rolesDescription')}
      access={canViewRoles}
      accessDeniedMessage={t('common.accessDenied')}
      pageHeaderAction={canManageRoles ? <RoleFormSheetTrigger /> : undefined}
    >
      <RoleListingPage />
    </PageContainer>
  );
}
