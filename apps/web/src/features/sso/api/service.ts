import { apiClient } from '@/lib/api-client';
import type { OIDCProvider, OIDCProviderPayload, PublicOIDCProvider, UserIdentity } from './types';

export async function getOIDCProviders(options?: RequestInit): Promise<OIDCProvider[]> {
  return (await apiClient<OIDCProvider[] | null>('/v1/admin/oidc-providers', options)) ?? [];
}

export function saveOIDCProvider(id: number | undefined, data: OIDCProviderPayload) {
  return apiClient<OIDCProvider>(
    id ? `/v1/admin/oidc-providers/${id}` : '/v1/admin/oidc-providers',
    {
      method: id ? 'PUT' : 'POST',
      body: JSON.stringify(data)
    }
  );
}

export async function deleteOIDCProvider(id: number): Promise<void> {
  await apiClient<void>(`/v1/admin/oidc-providers/${id}`, { method: 'DELETE' });
}

/** Enabled providers for the sign-in page (no sign-in needed). */
export async function getPublicOIDCProviders(): Promise<PublicOIDCProvider[]> {
  return (await apiClient<PublicOIDCProvider[] | null>('/v1/auth/oidc/providers')) ?? [];
}

/**
 * Browser URL that starts a provider sign-in. It is a navigation (the API answers
 * 302 to the provider), so it is a plain same-origin link, not a fetch.
 */
export function oidcAuthorizeUrl(code: string, redirect: string, remember: boolean): string {
  const params = new URLSearchParams({ redirect, remember: String(remember) });
  return `/api/v1/auth/oidc/${encodeURIComponent(code)}/authorize?${params}`;
}

export async function getMyIdentities(): Promise<UserIdentity[]> {
  return (await apiClient<UserIdentity[] | null>('/v1/account/identities')) ?? [];
}

/** Returns the provider URL to navigate to; the callback links the account. */
export async function startIdentityLink(code: string): Promise<string> {
  const res = await apiClient<{ authorizeUrl: string }>(
    `/v1/account/identities/${encodeURIComponent(code)}/link`,
    { method: 'POST' }
  );
  return res.authorizeUrl;
}

export async function unlinkIdentity(code: string): Promise<void> {
  await apiClient<void>(`/v1/account/identities/${encodeURIComponent(code)}`, { method: 'DELETE' });
}
