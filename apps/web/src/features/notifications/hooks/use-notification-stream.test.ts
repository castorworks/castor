import { describe, expect, it } from 'vitest';
import { parseEvent, reconnectDelay } from './use-notification-stream';

describe('notification stream helpers', () => {
  it('backs off exponentially up to a minute', () => {
    expect([1, 2, 3, 4, 5, 6, 7, 20].map(reconnectDelay)).toEqual([
      2000, 4000, 8000, 16000, 32000, 60000, 60000, 60000
    ]);
    expect(reconnectDelay(0)).toBe(2000);
  });

  it('parses event payloads and ignores malformed ones', () => {
    expect(parseEvent<{ count: number }>('{"count":3}')).toEqual({ count: 3 });
    expect(parseEvent('not json')).toBeNull();
  });
});
