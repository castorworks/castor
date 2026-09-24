import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import ThemeProvider from './theme-provider';

describe('ThemeProvider', () => {
  it('does not render inline script tags during SSR', () => {
    const html = renderToStaticMarkup(
      <ThemeProvider attribute='class' defaultTheme='system' enableSystem>
        <div>content</div>
      </ThemeProvider>
    );

    expect(html).not.toContain('<script');
    expect(html).toContain('content');
  });
});
