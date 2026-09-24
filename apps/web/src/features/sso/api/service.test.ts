import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/lib/api-client';
import { oidcAuthorizeUrl, saveOIDCProvider, startIdentityLink, unlinkIdentity } from './service';

vi.mock('@/lib/api-client', () => ({ apiClient: vi.fn() }));
const mocked = vi.mocked(apiClient);

describe('sso service', () => {
  beforeEach(() => mocked.mockReset());

  it('builds a same-origin authorize link with redirect and remember', () => {
    expect(oidcAuthorizeUrl('corp', '/dashboard/users?x=1', true)).toBe(
      '/api/v1/auth/oidc/corp/authorize?redirect=%2Fdashboard%2Fusers%3Fx%3D1&remember=true'
    );
  });

  it('creates with POST and updates with PUT', async () => {
    mocked.mockResolvedValue({} as never);
    const data = {
      code: 'corp',
      name: { en: 'Corp', zh: '公司', ja: '会社', ko: '회사' },
      issuer: 'https://idp.test',
      clientId: 'c',
      clientSecret: '',
      scopes: ['openid'],
      usernameClaim: '',
      autoRegister: false,
      isEnabled: true,
      sortOrder: 0
    };
    await saveOIDCProvider(undefined, data);
    await saveOIDCProvider(3, data);
    expect(mocked.mock.calls[0][0]).toBe('/v1/admin/oidc-providers');
    expect(mocked.mock.calls[0][1]?.method).toBe('POST');
    expect(mocked.mock.calls[1][0]).toBe('/v1/admin/oidc-providers/3');
    expect(mocked.mock.calls[1][1]?.method).toBe('PUT');
  });

  it('links and unlinks identities', async () => {
    mocked.mockResolvedValueOnce({ authorizeUrl: 'https://idp.test/authorize?x' });
    await expect(startIdentityLink('corp')).resolves.toBe('https://idp.test/authorize?x');
    expect(mocked.mock.calls[0]).toEqual(['/v1/account/identities/corp/link', { method: 'POST' }]);
    mocked.mockResolvedValueOnce(undefined as never);
    await unlinkIdentity('corp');
    expect(mocked.mock.calls[1]).toEqual(['/v1/account/identities/corp', { method: 'DELETE' }]);
  });
});
