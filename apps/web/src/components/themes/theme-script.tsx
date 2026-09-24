// Blocking pre-paint theme init. ThemeProvider only applies the theme after
// hydration, so without this a dark page first paints light and flashes white
// on every reload. Rendered by the root layout (server component) in <head>
// with the per-request CSP nonce; must mirror ThemeProvider's class strategy.

export const THEME_STORAGE_KEY = 'theme';

export function themeInitScript(storageKey: string, defaultTheme: string): string {
  return `(function(){try{var t=localStorage.getItem(${JSON.stringify(storageKey)})||${JSON.stringify(defaultTheme)};if(t==='system')t=matchMedia('(prefers-color-scheme: dark)').matches?'dark':'light';var r=document.documentElement;r.classList.remove('light','dark');r.classList.add(t);if(t==='light'||t==='dark')r.style.colorScheme=t}catch(e){}})()`;
}

export default function ThemeScript({
  nonce,
  storageKey = THEME_STORAGE_KEY,
  defaultTheme = 'system'
}: {
  nonce?: string;
  storageKey?: string;
  defaultTheme?: string;
}) {
  return (
    <script
      nonce={nonce}
      dangerouslySetInnerHTML={{ __html: themeInitScript(storageKey, defaultTheme) }}
    />
  );
}
