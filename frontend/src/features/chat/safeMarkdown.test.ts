// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';
import { renderSafeMarkdown } from './safeMarkdown';

describe('chat HTML security', () => {
  it('removes active content, UI styles, forms and unsafe links', () => {
    const html = renderSafeMarkdown(`<style>body{display:none}</style><script>alert(1)</script>
<img src=x onerror=alert(1)><svg onload=alert(1)></svg><iframe src="https://evil.test"></iframe>
<form><input name=apiKey></form><p style="position:fixed" onclick="alert(1)" id="root" class="markdown-table" tabindex="0">text</p>
<a href="javascript:alert(1)">js</a><a href="file:///secret">file</a><a href="wails://test">ipc</a>`);
    const doc = new DOMParser().parseFromString(html, 'text/html');
    expect(doc.querySelector('style,script,img,svg,iframe,form,input')).toBeNull();
    expect(doc.querySelector('[style],[onclick],[onerror],[id],[class],[tabindex]')).toBeNull();
    expect(doc.querySelector('a[href]')).toBeNull();
    expect(doc.body.textContent).toContain('text');
  });
  it('preserves ordinary Markdown, tables, code and HTTPS links', () => {
    const html = renderSafeMarkdown('**bold** and `code` [docs](https://example.com)\n\n| a | b |\n|---|---:|\n| 1 | 2 |');
    expect(html).toContain('<strong>bold</strong>');
    expect(html).toContain('<code>code</code>');
    expect(html).toContain('href="https://example.com"');
    const doc = new DOMParser().parseFromString(html, 'text/html');
    const wrapper = doc.querySelector('.markdown-table[role="region"][tabindex="0"]');
    expect(wrapper?.querySelector('table thead th')?.textContent).toBe('a');
    expect(wrapper?.querySelector('table tbody td[align="right"]')?.textContent).toBe('2');
  });

  it('turns links and file mentions into local preview controls', () => {
    const html = renderSafeMarkdown(
      'Creé [el informe](docs/Informe%20final.pdf), `src/main.ts:8`, README.md y notas.markdown. [Web](https://example.com/readme.md).',
      { enableFileLinks: true },
    );
    const doc = new DOMParser().parseFromString(html, 'text/html');
    const files = [...doc.querySelectorAll<HTMLButtonElement>('button[data-preview-path]')];
    expect(files.map((file) => file.dataset.previewPath)).toEqual(['docs/Informe final.pdf', 'src/main.ts', 'README.md', 'notas.markdown']);
    expect(doc.querySelector('a')?.getAttribute('href')).toBe('https://example.com/readme.md');
  });

  it('keeps untrusted attributes, external schemes and traversal out of preview controls', () => {
    const html = renderSafeMarkdown(
      '<button data-preview-path="secret.txt">fake</button> ' +
      '[escape](%2e%2e/secret.txt) [remote](//example.test/secret.txt) [file](file:///secret.txt) ' +
      '[script](javascript:evil) <pre><code>inside.txt</code></pre>',
      { enableFileLinks: true },
    );
    const doc = new DOMParser().parseFromString(html, 'text/html');
    expect(doc.querySelector('button, a[href], [data-preview-path]')).toBeNull();
  });
});
