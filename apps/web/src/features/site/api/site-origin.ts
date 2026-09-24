import { headers } from 'next/headers';

// Deployments do not bake their domain into the web image (see deploy/README.md),
// so absolute URLs are built from the request as the ingress forwarded it.
export function originFromHeaders(requestHeaders: Headers): string | null {
  const first = (name: string) => requestHeaders.get(name)?.split(',')[0]?.trim() || undefined;
  const host = first('x-forwarded-host') ?? first('host');
  if (!host || !/^[a-z0-9.-]+(:\d+)?$/i.test(host)) return null;
  const proto = first('x-forwarded-proto') === 'http' ? 'http' : 'https';
  return `${proto}://${host}`;
}

export async function getSiteOrigin(): Promise<string | null> {
  return originFromHeaders(await headers());
}
