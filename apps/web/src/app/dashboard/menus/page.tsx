import PageContainer from '@/components/layout/page-container';
import { MenuListing } from '@/features/menus/components/menu-listing';
import { serverHasPermission } from '@/lib/server-auth-headers';
import { getTranslations } from 'next-intl/server';
export async function generateMetadata() {
  const t = await getTranslations('menus');
  return { title: t('title') };
}
export default async function MenusPage() {
  const [t, allowed] = await Promise.all([
    getTranslations(),
    serverHasPermission('/api/v1/admin/menus:GET')
  ]);
  return (
    <PageContainer
      pageTitle={t('menus.title')}
      pageDescription={t('menus.description')}
      access={allowed}
      accessDeniedMessage={t('common.accessDenied')}
    >
      <MenuListing />
    </PageContainer>
  );
}
