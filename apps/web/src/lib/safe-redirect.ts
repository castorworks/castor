export const DEFAULT_REDIRECT_PATH = '/dashboard/overview';

export interface RelativePathParts {
  pathname: string;
  search: string;
  hash: string;
}

// Control characters and whitespace are stripped or reinterpreted by URL
// parsers (e.g. "/\t/evil.com" becomes "//evil.com"), so reject them outright.
// oxlint-disable-next-line no-control-regex
const UNSAFE_CHARS = /[\u0000-\u001F\u007F\\]/;
const DOT_SEGMENT = /(^|\/)\.{1,2}(\/|$)/;

/**
 * Split a same-origin, root-relative path into its parts without resolving it
 * against any origin. Returns null for anything that could leave the origin:
 * absolute URLs, schemes (javascript:, data:), protocol-relative "//host",
 * backslash tricks, control characters and dot segments.
 */
export function parseRelativePath(value: string | null | undefined): RelativePathParts | null {
  if (typeof value !== 'string' || !value.startsWith('/') || value.startsWith('//')) return null;
  if (UNSAFE_CHARS.test(value)) return null;

  const hashIndex = value.indexOf('#');
  const beforeHash = hashIndex === -1 ? value : value.slice(0, hashIndex);
  const hash = hashIndex === -1 ? '' : value.slice(hashIndex);
  const searchIndex = beforeHash.indexOf('?');
  const pathname = searchIndex === -1 ? beforeHash : beforeHash.slice(0, searchIndex);
  const search = searchIndex === -1 ? '' : beforeHash.slice(searchIndex);

  if (DOT_SEGMENT.test(pathname)) return null;

  return { pathname, search, hash };
}

/** Return `value` if it is a safe same-origin path, otherwise `fallback`. */
export function safeRedirectPath(
  value: string | null | undefined,
  fallback: string = DEFAULT_REDIRECT_PATH
): string {
  return parseRelativePath(value) ? (value as string) : fallback;
}
