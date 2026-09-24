import { queryOptions } from '@tanstack/react-query';
import { getMyIdentities, getOIDCProviders, getPublicOIDCProviders } from './service';

export const ssoKeys = {
  all: ['sso'] as const,
  providers: () => [...ssoKeys.all, 'providers'] as const,
  publicProviders: () => [...ssoKeys.all, 'public-providers'] as const,
  identities: () => [...ssoKeys.all, 'identities'] as const
};

export const oidcProvidersQueryOptions = (options?: RequestInit) =>
  queryOptions({ queryKey: ssoKeys.providers(), queryFn: () => getOIDCProviders(options) });

export const publicOIDCProvidersQueryOptions = () =>
  queryOptions({
    queryKey: ssoKeys.publicProviders(),
    queryFn: getPublicOIDCProviders,
    staleTime: 5 * 60 * 1000
  });

export const myIdentitiesQueryOptions = () =>
  queryOptions({ queryKey: ssoKeys.identities(), queryFn: getMyIdentities });
