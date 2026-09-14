'use client';

import * as React from 'react';

type DataAttribute = `data-${string}`;
type ThemeAttribute = 'class' | DataAttribute;

type ValueObject = Record<string, string>;

export type ThemeProviderProps = React.PropsWithChildren<{
  themes?: string[];
  forcedTheme?: string;
  enableSystem?: boolean;
  disableTransitionOnChange?: boolean;
  enableColorScheme?: boolean;
  storageKey?: string;
  defaultTheme?: string;
  attribute?: ThemeAttribute | ThemeAttribute[];
  value?: ValueObject;
}>;

type UseThemeProps = {
  themes: string[];
  forcedTheme?: string;
  setTheme: React.Dispatch<React.SetStateAction<string>>;
  theme?: string;
  resolvedTheme?: string;
  systemTheme?: 'dark' | 'light';
};

const DEFAULT_THEMES = ['light', 'dark'];
const MEDIA = '(prefers-color-scheme: dark)';

const ThemeContext = React.createContext<UseThemeProps>({
  themes: [],
  setTheme: () => {}
});

function getSystemTheme(): 'dark' | 'light' {
  if (typeof window === 'undefined') return 'light';
  return window.matchMedia(MEDIA).matches ? 'dark' : 'light';
}

function getStoredTheme(storageKey: string, fallback: string): string {
  if (typeof window === 'undefined') return fallback;

  try {
    return localStorage.getItem(storageKey) || fallback;
  } catch {
    return fallback;
  }
}

function disableTransitions() {
  const css = document.createElement('style');
  css.appendChild(
    document.createTextNode(
      '*,*::before,*::after{transition:none!important;animation:none!important}'
    )
  );
  document.head.appendChild(css);

  return () => {
    window.getComputedStyle(document.body);
    setTimeout(() => {
      css.remove();
    }, 1);
  };
}

export function useTheme() {
  return React.useContext(ThemeContext);
}

export default function ThemeProvider({ children, ...props }: ThemeProviderProps) {
  const {
    themes = DEFAULT_THEMES,
    forcedTheme,
    enableSystem = true,
    enableColorScheme = true,
    disableTransitionOnChange = false,
    storageKey = 'theme',
    defaultTheme = enableSystem ? 'system' : 'light',
    attribute = 'data-theme',
    value
  } = props;

  const [theme, setThemeState] = React.useState(() => getStoredTheme(storageKey, defaultTheme));
  const [systemTheme, setSystemTheme] = React.useState<'dark' | 'light'>(() => getSystemTheme());

  const attrs = React.useMemo(
    () => (Array.isArray(attribute) ? attribute : [attribute]),
    [attribute]
  );

  const applyTheme = React.useCallback(
    (themeName: string) => {
      if (typeof document === 'undefined') return;

      const resolved = themeName === 'system' && enableSystem ? getSystemTheme() : themeName;
      const attrValue = value?.[resolved] ?? resolved;
      const cleanupTransitions = disableTransitionOnChange ? disableTransitions() : undefined;
      const root = document.documentElement;
      const removableValues = new Set([...themes, ...Object.values(value ?? {}), 'light', 'dark']);

      attrs.forEach((attr) => {
        if (attr === 'class') {
          root.classList.remove(...removableValues);
          if (attrValue) root.classList.add(attrValue);
          return;
        }

        if (attrValue) {
          root.setAttribute(attr, attrValue);
        } else {
          root.removeAttribute(attr);
        }
      });

      if (enableColorScheme) {
        root.style.colorScheme = resolved === 'light' || resolved === 'dark' ? resolved : '';
      }

      cleanupTransitions?.();
    },
    [attrs, disableTransitionOnChange, enableColorScheme, enableSystem, themes, value]
  );

  const setTheme = React.useCallback<React.Dispatch<React.SetStateAction<string>>>(
    (nextTheme) => {
      setThemeState((currentTheme) => {
        const resolvedNextTheme =
          typeof nextTheme === 'function' ? nextTheme(currentTheme) : nextTheme;

        try {
          localStorage.setItem(storageKey, resolvedNextTheme);
        } catch {
          // Ignore storage failures; the in-memory theme still updates.
        }

        return resolvedNextTheme;
      });
    },
    [storageKey]
  );

  React.useEffect(() => {
    applyTheme(forcedTheme ?? theme);
  }, [applyTheme, forcedTheme, theme]);

  React.useEffect(() => {
    if (!enableSystem) return;

    const media = window.matchMedia(MEDIA);
    const updateSystemTheme = () => setSystemTheme(media.matches ? 'dark' : 'light');

    updateSystemTheme();
    media.addEventListener('change', updateSystemTheme);

    return () => {
      media.removeEventListener('change', updateSystemTheme);
    };
  }, [enableSystem]);

  React.useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (event.key !== storageKey) return;
      setThemeState(event.newValue || defaultTheme);
    };

    window.addEventListener('storage', onStorage);
    return () => window.removeEventListener('storage', onStorage);
  }, [defaultTheme, storageKey]);

  const resolvedTheme = theme === 'system' && enableSystem ? systemTheme : theme;
  const contextValue = React.useMemo<UseThemeProps>(
    () => ({
      theme,
      setTheme,
      forcedTheme,
      resolvedTheme,
      themes: enableSystem ? [...themes, 'system'] : themes,
      systemTheme: enableSystem ? systemTheme : undefined
    }),
    [enableSystem, forcedTheme, resolvedTheme, setTheme, systemTheme, theme, themes]
  );

  return <ThemeContext.Provider value={contextValue}>{children}</ThemeContext.Provider>;
}
