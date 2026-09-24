import { describe, expect, it } from 'vitest';
import { assetUrl } from './asset-url';

describe('assetUrl', () => {
  it('defaults to the unauthenticated public download route', () => {
    expect(assetUrl('abc123.png')).toBe('/api/v1/assets/download/abc123.png');
  });

  it('uses the RBAC-protected route for admin downloads', () => {
    expect(assetUrl('abc123.png', { admin: true })).toBe(
      '/api/v1/admin/assets/download/abc123.png'
    );
  });

  it('never lets an object key escape the download path', () => {
    expect(assetUrl('../users?x=1#y')).toBe('/api/v1/assets/download/..%2Fusers%3Fx%3D1%23y');
  });
});
