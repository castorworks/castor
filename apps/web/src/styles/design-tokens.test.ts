import fs from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';
import { DEFAULT_THEME, THEMES } from '@/components/themes/theme.config';
import { cn } from '@/lib/utils';

const srcDir = path.resolve(__dirname, '..');
const themesDir = path.join(__dirname, 'themes');
const themeCss = fs.readFileSync(path.join(__dirname, 'theme.css'), 'utf8');
const themeNames = THEMES.map((theme) => theme.value);

const FALLBACK_SELECTOR = ':root:not([data-theme])';

function readTheme(name: string): string {
  return fs.readFileSync(path.join(themesDir, `${name}.css`), 'utf8');
}

/** Custom properties declared in the rule whose selector list contains `selector`. */
function readTokens(css: string, selector: string): Set<string> {
  for (const [, selectors, body] of css.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    const list = selectors.split(',').map((item) => item.trim());
    if (list.includes(selector)) {
      return new Set([...body.matchAll(/^\s*(--[a-z0-9-]+):/gm)].map((match) => match[1]));
    }
  }
  return new Set();
}

function diff(base: Set<string>, other: Set<string>) {
  return {
    missing: [...base].filter((token) => !other.has(token)),
    extra: [...other].filter((token) => !base.has(token))
  };
}

describe('theme registry', () => {
  it('has one stylesheet per registered theme, imported by theme.css', () => {
    const files = fs
      .readdirSync(themesDir)
      .filter((file) => file.endsWith('.css'))
      .map((file) => file.replace(/\.css$/, ''));
    const imported = [...themeCss.matchAll(/@import '\.\/themes\/([a-z0-9-]+)\.css';/g)].map(
      (match) => match[1]
    );

    expect(files.sort()).toEqual([...themeNames].sort());
    expect(imported.sort()).toEqual([...themeNames].sort());
  });

  it('registers the default theme', () => {
    expect(themeNames).toContain(DEFAULT_THEME);
  });

  it.each(themeNames)('%s owns the no-data-theme fallback only when it is the default', (name) => {
    expect(readTheme(name).includes(FALLBACK_SELECTOR)).toBe(name === DEFAULT_THEME);
  });
});

describe('theme token parity', () => {
  const [baseTheme, ...otherThemes] = themeNames;
  const baseLight = readTokens(readTheme(baseTheme), `[data-theme='${baseTheme}']`);
  const baseDark = readTokens(readTheme(baseTheme), `[data-theme='${baseTheme}'].dark`);

  it('defines tokens for light and dark, with dark overriding a subset of light', () => {
    expect(baseLight.size).toBeGreaterThan(0);
    expect(baseDark.size).toBeGreaterThan(0);
    expect(diff(baseLight, baseDark).extra).toEqual([]);
  });

  it.each(otherThemes)('%s defines exactly the same tokens as the base theme', (name) => {
    const css = readTheme(name);
    expect(diff(baseLight, readTokens(css, `[data-theme='${name}']`))).toEqual({
      missing: [],
      extra: []
    });
    expect(diff(baseDark, readTokens(css, `[data-theme='${name}'].dark`))).toEqual({
      missing: [],
      extra: []
    });
  });

  it('defines every token that theme.css maps into Tailwind', () => {
    const themeBlock = themeCss.match(/@theme inline \{([^}]*)\}/)?.[1] ?? '';
    const referenced = [...themeBlock.matchAll(/var\((--[a-z0-9-]+)\)/g)].map((m) => m[1]);
    const isColor = (token: string) => themeBlock.includes(`--color-${token.slice(2)}:`);
    // Theme-independent tokens (tag-*) live in theme.css itself
    const globalLight = readTokens(themeCss, ':root');
    const globalDark = readTokens(themeCss, '.dark');

    expect(referenced.length).toBeGreaterThan(0);
    expect(referenced.filter((token) => !baseLight.has(token) && !globalLight.has(token))).toEqual(
      []
    );
    // Every color needs a dark value
    expect(
      referenced.filter((token) => isColor(token) && !baseDark.has(token) && !globalDark.has(token))
    ).toEqual([]);
  });
});

describe('design token usage', () => {
  // Tailwind palette utilities (bg-green-600, text-zinc-500, ...) bypass the theme.
  // Use semantic tokens (success / warning / info / destructive / muted ...) or tag-* instead.
  const PALETTE_CLASS =
    /\b(?:bg|text|border|ring|outline|fill|stroke|from|to|via|divide|decoration|shadow|accent|caret)-(?:slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-\d{2,3}\b/g;

  function sourceFiles(dir: string): string[] {
    return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) return sourceFiles(full);
      return /\.tsx?$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name) ? [full] : [];
    });
  }

  // Font sizes come from the type scale (text-2xs, text-xs, text-sm, ...), not text-[11px].
  const ARBITRARY_FONT_SIZE = /\btext-\[[\d.]+(?:px|rem|em)\]/g;
  // Tokens are oklch values: wrapping them in hsl()/rgb() yields invalid CSS.
  const WRAPPED_TOKEN = /\b(?:hsl|rgb)a?\(var\(--[a-z0-9-]+\)/g;

  function findOffenders(pattern: RegExp): string[] {
    return sourceFiles(srcDir).flatMap((file) => {
      const matches = fs.readFileSync(file, 'utf8').match(pattern) ?? [];
      return matches.map((match) => `${path.relative(srcDir, file)}: ${match}`);
    });
  }

  it('does not use raw Tailwind palette classes', () => {
    expect(findOffenders(PALETTE_CLASS)).toEqual([]);
  });

  it('does not use arbitrary font sizes', () => {
    expect(findOffenders(ARBITRARY_FONT_SIZE)).toEqual([]);
  });

  it('does not wrap color tokens in hsl() or rgb()', () => {
    expect(findOffenders(WRAPPED_TOKEN)).toEqual([]);
  });

  // Select triggers, menus and tabs render translated text on one line, so a fixed width
  // clips the longer locales ("Englis…"). Size them to content; use min-w-* to keep a floor.
  const SINGLE_LINE_CONTROL =
    /<(SelectTrigger|SelectContent|DropdownMenuContent|DropdownMenuSubContent|ContextMenuContent|ContextMenuSubContent|MenubarContent|MenubarSubContent|TabsTrigger)\b((?:=>|[^<>])*?)\/?>/g;
  const FIXED_WIDTH = /(?:^|[\s'"`:])(w-(?:\d+(?:\.\d+)?|px|\[(?!var\()[^\]]+\]))(?=[\s'"`]|$)/g;

  it('does not give single-line text controls a fixed width', () => {
    const offenders = sourceFiles(srcDir).flatMap((file) =>
      [...fs.readFileSync(file, 'utf8').matchAll(SINGLE_LINE_CONTROL)].flatMap(([, tag, attrs]) =>
        [...attrs.matchAll(FIXED_WIDTH)].map(
          ([, width]) => `${path.relative(srcDir, file)}: <${tag}> ${width}`
        )
      )
    );
    expect(offenders).toEqual([]);
  });

  it('registers custom font-size tokens with tailwind-merge', () => {
    const sizes = [...themeCss.matchAll(/^\s*--text-([a-z0-9]+):/gm)].map((match) => match[1]);
    expect(sizes.length).toBeGreaterThan(0);
    for (const size of sizes) {
      expect(cn('text-xs text-primary-foreground', `text-${size}`)).toBe(
        `text-primary-foreground text-${size}`
      );
    }
  });
});
