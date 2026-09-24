import { describe, expect, it } from 'vitest';
import { jobSchema } from './job';

const schema = jobSchema({ cronInvalid: 'invalid' });

describe('jobSchema', () => {
  it('accepts standard five-field expressions', () => {
    for (const cron of ['0 * * * *', ' 30  3 * * 1-5 ', '*/15 * * * *']) {
      expect(schema.safeParse({ cron, isEnabled: true }).success).toBe(true);
    }
  });

  it('rejects descriptors, time zone prefixes and wrong field counts', () => {
    for (const cron of ['', '* * * *', '0 0 * * * *', '@hourly', 'TZ=UTC 0 3 * * *']) {
      expect(schema.safeParse({ cron, isEnabled: true }).success).toBe(false);
    }
  });
});
