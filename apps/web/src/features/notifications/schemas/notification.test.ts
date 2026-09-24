import { describe, expect, it } from 'vitest';
import { createNotificationSchema } from './notification';

const t = (key: string) => key;

function validValues(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    title: 'System notice',
    content: '',
    type: 'SYSTEM',
    level: 'INFO',
    link: '',
    extra: '',
    isGlobal: false,
    sendEmail: false,
    userIds: '1,2,3',
    expireAt: '',
    attachments: [],
    ...overrides
  };
}

const file = (n: number) => ({ objectKey: `k${n}.pdf`, name: `f${n}.pdf` });

describe('createNotificationSchema', () => {
  it('requires recipients even when the notification is marked global', () => {
    const schema = createNotificationSchema(t);

    const result = schema.safeParse(validValues({ isGlobal: true, userIds: '' }));

    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.message).toBe('validation.recipientsRequired');
  });

  it('rejects recipient text that does not contain positive integer IDs', () => {
    const schema = createNotificationSchema(t);

    const result = schema.safeParse(validValues({ userIds: 'abc, 0, -1' }));

    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.message).toBe('validation.recipientsInvalid');
  });

  it('accepts attachments up to the limit', () => {
    const schema = createNotificationSchema(t);

    expect(schema.safeParse(validValues({ attachments: [file(1), file(2)] })).success).toBe(true);

    const tooMany = schema.safeParse(
      validValues({ attachments: Array.from({ length: 11 }, (_, n) => file(n)) })
    );
    expect(tooMany.error?.issues[0]?.message).toBe('validation.attachmentsMax');

    const malformed = schema.safeParse(validValues({ attachments: [{ name: 'no-key.pdf' }] }));
    expect(malformed.success).toBe(false);
  });
});
