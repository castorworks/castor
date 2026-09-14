import { redirect } from 'next/navigation';
import { getAuthToken } from '@/lib/auth';

export default async function Page() {
  const token = await getAuthToken();
  if (token) {
    redirect('/dashboard/overview');
  }
  redirect('/auth/sign-in');
}
