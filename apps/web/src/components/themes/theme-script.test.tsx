import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import ThemeScript, { themeInitScript } from './theme-script';

function run(stored: string | null, systemDark: boolean) {
  const classes = new Set<string>(['light']);
  const root = {
    classList: {
      add: (...c: string[]) => c.forEach((x) => classes.add(x)),
      remove: (...c: string[]) => c.forEach((x) => classes.delete(x))
    },
    style: { colorScheme: '' }
  };
  new Function('localStorage', 'matchMedia', 'document', themeInitScript('theme', 'system'))(
    { getItem: () => stored },
    () => ({ matches: systemDark }),
    { documentElement: root }
  );
  return { classes: [...classes], colorScheme: root.style.colorScheme };
}

describe('themeInitScript', () => {
  it('applies a stored dark theme before paint', () => {
    expect(run('dark', false)).toEqual({ classes: ['dark'], colorScheme: 'dark' });
  });

  it('resolves the system preference when nothing is stored', () => {
    expect(run(null, true)).toEqual({ classes: ['dark'], colorScheme: 'dark' });
    expect(run('system', false)).toEqual({ classes: ['light'], colorScheme: 'light' });
  });

  it('stamps the CSP nonce on the inline script', () => {
    expect(renderToStaticMarkup(<ThemeScript nonce='abc' />)).toContain('nonce="abc"');
  });
});
