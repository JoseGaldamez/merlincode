import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('defensas del WebView', () => {
  it('usa una CSP restrictiva sin scripts inline ni orígenes de fuentes remotos', () => {
    const html = readFileSync(resolve(process.cwd(), 'index.html'), 'utf8');

    expect(html).toContain("default-src 'self'");
    expect(html).toContain("script-src 'self'");
    expect(html).not.toMatch(/script-src[^;]*'unsafe-inline'/);
    expect(html).toContain("object-src 'none'");
    expect(html).toContain("frame-ancestors 'none'");
    expect(html).not.toContain('fonts.googleapis.com');
    expect(html).not.toContain('fonts.gstatic.com');
  });

  it('incluye localmente las fuentes declaradas por la hoja de estilos', () => {
    const style = readFileSync(resolve(process.cwd(), 'src/style.css'), 'utf8');

    expect(style).toContain("url('./assets/fonts/geist-latin.woff2')");
    expect(style).toContain("url('./assets/fonts/jetbrains-mono-latin.woff2')");
    expect(existsSync(resolve(process.cwd(), 'src/assets/fonts/geist-latin.woff2'))).toBe(true);
    expect(existsSync(resolve(process.cwd(), 'src/assets/fonts/jetbrains-mono-latin.woff2'))).toBe(true);
  });
});
