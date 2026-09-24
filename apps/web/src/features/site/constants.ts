export const SITE_REPOSITORY_URL = 'https://github.com/castorworks/castor';

/** Public pages search engines should index; listed in sitemap.xml. */
export const SITE_INDEXABLE_PATHS = ['/', '/about'] as const;

/** Paths kept out of search engines; the app pages are also marked noindex by their layouts. */
export const SITE_PRIVATE_PATHS = ['/dashboard', '/auth', '/403', '/api/'] as const;
