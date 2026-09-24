import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';

const pageSource = readFileSync(join(import.meta.dirname, 'page.tsx'), 'utf8');

describe('CaptchaImage', () => {
  it('keeps a light captcha surface in dark mode', () => {
    expect(pageSource).toContain('bg-white');
    expect(pageSource).not.toContain('dark:bg-');
  });

  it('loads public settings to derive allowed login methods', () => {
    expect(pageSource).toContain('publicSettingsQueryOptions');
    expect(pageSource).toContain('security.login.allowedMethods');
  });

  it('keeps all login methods as the public settings fallback', () => {
    expect(pageSource).toContain('DEFAULT_LOGIN_METHODS');
    expect(pageSource).toContain("'password'");
    expect(pageSource).toContain("'email'");
    expect(pageSource).toContain("'mobile'");
  });
});
