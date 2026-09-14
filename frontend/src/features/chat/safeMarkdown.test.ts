// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';
import { renderSafeMarkdown } from './safeMarkdown';

describe('chat HTML security', () => {
  it('removes active content, UI styles, forms and unsafe links', () => {
    const html = renderSafeMarkdown(`<style>body{display:none}</style><script>alert(1)</script>
<img src=x onerror=alert(1)><svg onload=alert(1)></svg><iframe src="https://evil.test"></iframe>
<form><input name=apiKey></form><p style="position:fixed" onclick="alert(1)" id="root">text</p>
<a href="javascript:alert(1)">js</a><a href="file:///secret">file</a><a href="wails://test">ipc</a>`);
    const doc = new DOMParser().parseFromString(html, 'text/html');
    expect(doc.querySelector('style,script,img,svg,iframe,form,input')).toBeNull();
    expect(doc.querySelector('[style],[onclick],[onerror],[id]')).toBeNull();
    expect(doc.querySelector('a[href]')).toBeNull();
    expect(doc.body.textContent).toContain('text');
  });
  it('preserves ordinary Markdown, tables, code and HTTPS links', () => {
    const html = renderSafeMarkdown('**bold** and `code` [docs](https://example.com)\n\n| a | b |\n|---|---|\n| 1 | 2 |');
    expect(html).toContain('<strong>bold</strong>');
    expect(html).toContain('<code>code</code>');
    expect(html).toContain('href="https://example.com"');
    expect(html).toContain('<table>');
  });
});
