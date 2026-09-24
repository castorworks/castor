// ============================================================
// API Client — castor backend adapter
// ============================================================
// Wraps fetch calls to the castor Go backend, handling:
// - Unified response format: { code, data, message }
// - Cookie-based auth (httpOnly jwt) with CSRF header injection
// - 401 auto-refresh token retry
// - Business error code mapping to user-friendly messages
// - Pagination format conversion: { total, list } → TanStack Table shape
// ============================================================

import { getCastorApiUrl } from '@/lib/server-config';

export interface CastorResponse<T = unknown> {
  code: number;
  data: T;
  message: string;
  /** Business error code on failures (e.g. 2002 captcha required); see CastorErrorCode. */
  errorCode?: number;
}

export interface CastorListResponse<T = unknown> {
  total: number;
  list: T[];
  page?: number;
  pageSize?: number;
  totalPages?: number;
}

/**
 * Business error codes the UI branches on (backend `response/errors.go`). The HTTP
 * status alone cannot tell "captcha required" from other 400s, or "password expired"
 * from other 403s.
 */
export const CastorErrorCode = {
  Unauthorized: 2000,
  InvalidCaptcha: 2002,
  CredentialExpired: 2009,
  /** Password (or code) was right; `data.challenge` must be completed with a TOTP code. */
  TOTPRequired: 2012,
  InvalidTOTP: 2013,
  /** The two-factor step expired or ran out of attempts: start the sign-in again. */
  MFAChallengeExpired: 2014
} as const;

/** Unified API error with castor error code */
export class CastorApiError extends Error {
  constructor(
    message: string,
    public code: number,
    public httpStatus: number,
    /** Business error code from the response body; 0 when the backend sent none. */
    public errorCode: number = 0,
    /** `data` of the error response, e.g. per-row problems of a rejected import. */
    public data: unknown = null
  ) {
    super(message);
    this.name = 'CastorApiError';
  }
}

/** Resolve the effective base URL (supports both direct and proxied paths) */
function resolveBaseUrl(): string {
  return process.env.NEXT_PUBLIC_API_URL ?? '';
}

/** Build full URL with optional API base prefix */
export function buildApiUrl(endpoint: string): string {
  const base = resolveBaseUrl();
  // If endpoint already starts with http, use as-is
  if (endpoint.startsWith('http://') || endpoint.startsWith('https://')) {
    return endpoint;
  }

  // If base URL is set, prepend it
  if (base) {
    const cleanBase = base.replace(/\/+$/, '');
    const cleanEndpoint = endpoint.startsWith('/') ? endpoint : `/${endpoint}`;
    return `${cleanBase}${cleanEndpoint}`;
  }

  // No base URL: assume proxied via Next.js rewrites, prefix with /api
  const cleanEndpoint = endpoint.startsWith('/') ? endpoint : `/${endpoint}`;
  if (typeof window === 'undefined') {
    return `${getCastorApiUrl()}/api${cleanEndpoint}`;
  }

  return `/api${cleanEndpoint}`;
}

function normalizeEndpointPath(endpoint: string): string {
  const path =
    endpoint.startsWith('http://') || endpoint.startsWith('https://')
      ? new URL(endpoint).pathname
      : endpoint.split('?')[0];

  return path.replace(/^\/api/, '');
}

export function getAuthRecoveryPolicy(endpoint: string): {
  refresh: boolean;
  redirectOnFailure: boolean;
} {
  const path = normalizeEndpointPath(endpoint);
  const isPublicAuthEndpoint =
    path === '/v1/auth/login' ||
    path === '/v1/auth/login/totp' ||
    path === '/v1/auth/oidc/providers' ||
    path === '/v1/auth/refresh-token' ||
    path === '/v1/auth/captcha' ||
    path === '/v1/auth/public-key' ||
    path === '/v1/auth/code' ||
    path === '/v1/auth/register' ||
    path === '/v1/auth/password/expired' ||
    path === '/v1/auth/logout' ||
    path === '/v1/account/password/reset';

  return {
    refresh: !isPublicAuthEndpoint,
    redirectOnFailure: !isPublicAuthEndpoint
  };
}

function getCsrfToken(): string | undefined {
  if (typeof document === 'undefined') return undefined;
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : undefined;
}

/**
 * Resolve the effective locale for API requests.
 * Priority: NEXT_LOCALE cookie > browser language > fallback 'zh'.
 */
function getLocale(): string {
  if (typeof document !== 'undefined') {
    const match = document.cookie.split('; ').find((row) => row.startsWith('NEXT_LOCALE='));
    if (match) {
      return match.split('=')[1];
    }
  }
  if (typeof navigator !== 'undefined') {
    return navigator.language?.split('-')[0] || 'zh';
  }
  return 'zh';
}

/**
 * Authentication is carried by the backend's httpOnly `jwt` cookie, which the
 * browser attaches automatically (`credentials: 'include'`). Server-side callers
 * forward auth explicitly via `server-auth-headers.ts`.
 */
export function buildDefaultHeaders(language: string = getLocale()): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'Accept-Language': language
  };

  const csrf = getCsrfToken();
  if (csrf) headers['X-CSRF-Token'] = csrf;

  return headers;
}

function buildUploadHeaders(language: string = getLocale()): Record<string, string> {
  const headers: Record<string, string> = {
    'Accept-Language': language
  };

  const csrf = getCsrfToken();
  if (csrf) headers['X-CSRF-Token'] = csrf;

  return headers;
}

function redirectToSignIn(): boolean {
  if (typeof window === 'undefined') return false;

  const currentPath = window.location.pathname + window.location.search;

  // An in-flight protected request may finish after the router has already
  // reached sign-in. Treat that as handled so stale 401s do not hit React.
  if (currentPath.startsWith('/auth/')) return true;

  // The httpOnly jwt cookie cannot be cleared from JS; proxy.ts clears it when
  // the sign-in redirect is issued for an invalid session.
  window.location.href = `/auth/sign-in?redirect=${encodeURIComponent(currentPath)}`;
  return true;
}

async function isAuthenticationFailure(res: Response): Promise<boolean> {
  if (res.status === 401) return true;
  if (res.ok) return false;

  try {
    const body = (await res.clone().json()) as Partial<CastorResponse<unknown>>;
    return body.errorCode === CastorErrorCode.Unauthorized || body.code === 401;
  } catch {
    return false;
  }
}

/** Map castor business error codes to user-friendly messages */
function getErrorMessage(code: number, message: string): string {
  // Castor uses HTTP status codes in the response
  // The message field already contains i18n text from the backend
  return message || 'Unknown error';
}

/** Handle a raw fetch response, unwrapping castor's { code, data, message } */
export async function readCastorResponse<T>(res: Response): Promise<T> {
  // Network / HTTP error (non-2xx)
  if (!res.ok) {
    // Try to parse castor error response
    let body: CastorResponse<unknown> | undefined;
    try {
      body = (await res.json()) as CastorResponse<unknown>;
    } catch {
      // Fall through to a generic HTTP error below.
    }

    if (body) {
      throw new CastorApiError(
        getErrorMessage(body.code, body.message),
        body.code,
        res.status,
        body.errorCode ?? 0,
        body.data ?? null
      );
    }

    throw new CastorApiError(`HTTP ${res.status} ${res.statusText}`, res.status, res.status);
  }

  const body = (await res.json()) as CastorResponse<T>;

  // castor uses code 200 for success
  if (body.code === 200) {
    return body.data;
  }

  // Business-level error with HTTP 200
  throw new CastorApiError(
    getErrorMessage(body.code, body.message),
    body.code,
    res.status,
    body.errorCode ?? 0
  );
}

/**
 * Handle 401 auth recovery: try token refresh once, then retry.
 * Extracts shared logic used by both apiClient and apiUpload.
 */
async function handle401Recovery(
  res: Response,
  authRecovery: ReturnType<typeof getAuthRecoveryPolicy>,
  headers: Record<string, string>,
  rebuildHeaders: () => Record<string, string>,
  retryFetch: (newHeaders: Record<string, string>) => Promise<Response>
): Promise<Response> {
  if (!authRecovery.refresh) return res;

  const authFailed = await isAuthenticationFailure(res);
  if (!authFailed) return res;

  const refreshed = await tryRefreshToken();
  if (!refreshed) {
    if (authRecovery.redirectOnFailure) {
      const redirected = redirectToSignIn();
      if (redirected) {
        return new Promise<Response>(() => {});
      }
    }
    return res;
  }

  // Rebuild headers (new token if available) and retry
  const newHeaders = { ...headers, ...rebuildHeaders() };
  return retryFetch(newHeaders);
}

/**
 * Core API client function.
 *
 * @param endpoint - API path (e.g. '/v1/auth/login'), auto-prefixed
 * @param options - Standard RequestInit options
 * @returns Parsed response data (the `data` field of castor response)
 */
export async function apiClient<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const url = buildApiUrl(endpoint);
  const authRecovery = getAuthRecoveryPolicy(endpoint);

  const headers = buildDefaultHeaders();

  // Merge custom headers
  if (options?.headers) {
    Object.assign(headers, options.headers as Record<string, string>);
  }

  const credentials = options?.credentials ?? 'include';

  const res = await handle401Recovery(
    await fetch(url, { ...options, headers, credentials }),
    authRecovery,
    headers,
    buildDefaultHeaders,
    (newHeaders) => fetch(url, { ...options, headers: newHeaders, credentials })
  );

  return readCastorResponse<T>(res);
}

/**
 * File upload client (multipart/form-data).
 * Does not set Content-Type so the browser can set the boundary.
 */
export async function apiUpload<T>(
  endpoint: string,
  formData: FormData,
  options?: Omit<RequestInit, 'body' | 'headers'>
): Promise<T> {
  const url = buildApiUrl(endpoint);
  const authRecovery = getAuthRecoveryPolicy(endpoint);
  const headers = buildUploadHeaders();
  const method = options?.method ?? 'POST';
  const credentials = options?.credentials ?? 'include';

  const res = await handle401Recovery(
    await fetch(url, { ...options, method, body: formData, headers, credentials }),
    authRecovery,
    headers,
    buildUploadHeaders,
    (newHeaders) =>
      fetch(url, { ...options, method, body: formData, headers: newHeaders, credentials })
  );

  return readCastorResponse<T>(res);
}

/** A file returned by a download endpoint. */
export interface DownloadedFile {
  blob: Blob;
  filename: string;
}

/** The filename from `Content-Disposition` (RFC 5987 `filename*` first). */
export function filenameFromDisposition(header: string | null, fallback: string): string {
  if (!header) return fallback;
  const extended = header.match(/filename\*=UTF-8''([^;]+)/i);
  if (extended) {
    try {
      return decodeURIComponent(extended[1]);
    } catch {
      return fallback;
    }
  }
  const plain = header.match(/filename="?([^";]+)"?/i);
  return plain ? plain[1] : fallback;
}

/**
 * GET a file (export, template). Errors come back as the usual JSON envelope
 * and are thrown as `CastorApiError`, like `apiClient`.
 */
export async function apiDownload(endpoint: string, fallbackName: string): Promise<DownloadedFile> {
  const url = buildApiUrl(endpoint);
  const authRecovery = getAuthRecoveryPolicy(endpoint);
  const headers = buildUploadHeaders();
  const credentials = 'include';

  const res = await handle401Recovery(
    await fetch(url, { headers, credentials }),
    authRecovery,
    headers,
    buildUploadHeaders,
    (newHeaders) => fetch(url, { headers: newHeaders, credentials })
  );
  if (!res.ok) {
    await readCastorResponse<never>(res);
  }
  return {
    blob: await res.blob(),
    filename: filenameFromDisposition(res.headers.get('Content-Disposition'), fallbackName)
  };
}

/**
 * Attempt to refresh the JWT token via the castor refresh endpoint.
 * Returns true if successful (new token is stored in cookie).
 *
 * Concurrent calls share a single in-flight refresh promise so we never
 * fire multiple parallel /refresh-token calls when several requests hit
 * 401 at the same time (avoids server-side replay + back-end log churn).
 */
let inflightRefresh: Promise<boolean> | null = null;

async function tryRefreshToken(): Promise<boolean> {
  if (inflightRefresh) return inflightRefresh;

  inflightRefresh = (async () => {
    try {
      const url = buildApiUrl('/v1/auth/refresh-token');
      const res = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(getCsrfToken() ? { 'X-CSRF-Token': getCsrfToken() as string } : {})
        },
        credentials: 'include'
      });

      if (!res.ok) return false;

      const body = (await res.json()) as CastorResponse<{
        token?: string;
        expire: string;
      }>;

      // The backend rotates the httpOnly jwt cookie on the refresh response.
      return body.code === 200;
    } catch {
      return false;
    } finally {
      // Allow the next 401 wave to refresh again, but only after the
      // current attempt has fully resolved.
      inflightRefresh = null;
    }
  })();

  return inflightRefresh;
}

/**
 * Convert castor pagination format to TanStack Table shape.
 */
export function toTanStackTable<T>(paged: CastorListResponse<T>): {
  data: T[];
  totalCount: number;
  pageCount: number;
} {
  return {
    data: paged.list ?? [],
    totalCount: paged.total ?? 0,
    pageCount: paged.totalPages ?? Math.ceil((paged.total ?? 0) / (paged.pageSize ?? 10))
  };
}
