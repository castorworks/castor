import { describe, expect, it } from 'vitest';
import { resolveSiteInfo } from './site-info';

describe('resolveSiteInfo', () => {
  it('reads the brand and registration switch from public settings', () => {
    expect(
      resolveSiteInfo({
        'site.name': ' Acme Console ',
        'site.description': 'Internal tools for Acme',
        'feature.register.enabled': true
      })
    ).toEqual({
      name: 'Acme Console',
      description: 'Internal tools for Acme',
      registerEnabled: true
    });
  });

  it('falls back when settings are blank, malformed or unavailable', () => {
    const fallback = { name: 'Castor Admin', description: 'Castor Admin', registerEnabled: false };
    expect(resolveSiteInfo(null)).toEqual(fallback);
    expect(
      resolveSiteInfo({
        'site.name': '  ',
        'site.description': 42,
        'feature.register.enabled': 'true'
      })
    ).toEqual(fallback);
  });

  it('uses the site name as the description when none is set', () => {
    expect(resolveSiteInfo({ 'site.name': 'Acme' }).description).toBe('Acme');
  });
});
