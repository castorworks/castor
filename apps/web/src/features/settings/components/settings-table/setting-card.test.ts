import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';

const source = readFileSync(join(import.meta.dirname, 'setting-card.tsx'), 'utf8');

describe('SettingCard login method editor', () => {
  it('uses a dedicated editor for the allowed login methods setting', () => {
    expect(source).toContain('LOGIN_METHODS_SETTING_KEY');
    expect(source).toContain('AllowedLoginMethodsEditor');
    expect(source).toContain('usesInlineEditor');
    expect(source).toContain('!usesInlineEditor && canUpdateSetting');
    expect(source).toContain('security.login.allowedMethods');
  });

  it('prevents disabling the final login method', () => {
    expect(source).toContain('selectedMethods.length === 1');
    expect(source).toContain('settings.loginMethods.atLeastOne');
  });
});
