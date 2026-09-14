import PageContainer from '@/components/layout/page-container';
import { ConstraintFormSheetTrigger } from '@/features/roles/components/constraint-form-sheet';
import { ConstraintListing } from '@/features/roles/components/constraint-listing';
import { serverHasPermission } from '@/lib/server-auth-headers';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations('authorization');
  return { title: t('title') };
}

export default async function AuthorizationPage() {
  const t = await getTranslations('authorization');
  const canView = await serverHasPermission('/api/v1/admin/authorization/constraints:GET');
  const canCreate = await serverHasPermission('/api/v1/admin/authorization/constraints:POST');
  return (
    <PageContainer
      pageTitle={t('title')}
      pageDescription={t('description')}
      access={canView}
      accessDeniedMessage={t('accessDenied')}
      pageHeaderAction={canCreate ? <ConstraintFormSheetTrigger /> : undefined}
    >
      <ConstraintListing />
    </PageContainer>
  );
}
