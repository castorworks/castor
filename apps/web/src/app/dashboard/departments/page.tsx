import PageContainer from '@/components/layout/page-container';
import { DepartmentListing } from '@/features/departments/components/department-listing';
import { serverHasPermission } from '@/lib/server-auth-headers';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations();
  return { title: t('nav.departments') };
}

export default async function DepartmentsPage() {
  const [t, allowed] = await Promise.all([
    getTranslations(),
    serverHasPermission('/api/v1/admin/departments:GET')
  ]);
  return (
    <PageContainer
      pageTitle={t('nav.departments')}
      pageDescription={t('dashboard.departmentsDescription')}
      access={allowed}
      accessDeniedMessage={t('common.accessDenied')}
    >
      <DepartmentListing />
    </PageContainer>
  );
}
