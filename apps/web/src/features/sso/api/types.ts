// ============================================================
// Single sign-on (OIDC) types — aligned with castor dto.OIDCProviderResp etc.
// ============================================================

import type { I18nText } from '@/lib/i18n-text';

/** An identity provider as admins see it; the client secret is never returned. */
export interface OIDCProvider {
  id: number;
  code: string;
  name: I18nText;
  issuer: string;
  clientId: string;
  scopes: string[];
  usernameClaim: string;
  autoRegister: boolean;
  isEnabled: boolean;
  sortOrder: number;
  createdAt: string;
  updatedAt: string;
  hasClientSecret: boolean;
  /** Redirect URI to register at the provider; empty until General.PublicURL is set. */
  callbackUrl: string;
}

export interface OIDCProviderPayload {
  code: string;
  name: I18nText;
  issuer: string;
  clientId: string;
  /** Empty on update keeps the stored secret. */
  clientSecret: string;
  scopes: string[];
  usernameClaim: string;
  autoRegister: boolean;
  isEnabled: boolean;
  sortOrder: number;
}

/** A sign-in button on the sign-in page. */
export interface PublicOIDCProvider {
  code: string;
  name: I18nText;
}

/** The signed-in user's link to one provider. */
export interface UserIdentity {
  providerCode: string;
  providerName: I18nText;
  providerEnabled: boolean;
  linked: boolean;
  email: string;
  linkedAt: string | null;
  lastLoginAt: string | null;
}
