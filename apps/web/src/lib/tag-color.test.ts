import { describe, expect, it } from 'vitest';
import { tagColorBg } from './tag-color';

describe('tagColorBg', () => {
  it('maps named colors to tag tokens', () => {
    expect(tagColorBg('red')).toBe('bg-tag-red');
    expect(tagColorBg('orange')).toBe('bg-tag-orange');
  });

  it('returns undefined for empty or non-token colors', () => {
    expect(tagColorBg('')).toBeUndefined();
    expect(tagColorBg(null)).toBeUndefined();
    expect(tagColorBg('#ff0000')).toBeUndefined();
    expect(tagColorBg('toString')).toBeUndefined();
  });
});
