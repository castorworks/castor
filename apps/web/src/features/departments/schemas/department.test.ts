import { describe, expect, it } from 'vitest';
import { departmentSchema } from './department';

const schema = departmentSchema({ codeInvalid: 'code', nameRequired: 'name' });
const valid = { parentId: 'root', code: 'rd-1', name: 'R&D', sortOrder: 0, isEnabled: true };

describe('departmentSchema', () => {
  it('accepts a valid department', () => {
    expect(schema.safeParse(valid).success).toBe(true);
  });

  it('rejects codes the backend would reject', () => {
    for (const code of ['RD', '-rd', 'r d', 'a'.repeat(65)]) {
      const result = schema.safeParse({ ...valid, code });
      expect(result.error?.issues[0]?.message).toBe('code');
    }
  });

  it('requires a name', () => {
    expect(schema.safeParse({ ...valid, name: '  ' }).error?.issues[0]?.message).toBe('name');
  });
});
