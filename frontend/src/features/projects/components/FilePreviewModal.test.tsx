// @vitest-environment jsdom
import { act } from 'react';
import { createRoot } from 'react-dom/client';
import { expect, it, vi } from 'vitest';
import { FilePreviewModal } from './FilePreviewModal';

const readPreview = vi.hoisted(() => vi.fn());
vi.mock('../../../../wailsjs/go/app/App', () => ({ ReadProjectFilePreview: readPreview }));

it.each(['README.md', 'docs/Guia.MARKDOWN'])('renders %s as a sanitized Markdown document', async (path) => {
  readPreview.mockResolvedValue(btoa([
    '# Proyecto',
    '',
    '**Primer** renglon',
    'segundo renglon.',
    '',
    '- Ejemplo',
    '',
    '| Nombre |',
    '| --- |',
    '| Merlin |',
    '',
    '```js',
    'const example = true;',
    '```',
    '',
    '<script>unsafe()</script><a href="javascript:unsafe()">Enlace</a>',
  ].join('\n')));
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  const dialog = HTMLDialogElement.prototype;
  const previousMethods = { showModal: dialog.showModal, close: dialog.close };
  Object.assign(dialog, { showModal() {}, close() {} });
  const host = document.createElement('div');
  document.body.append(host);
  const root = createRoot(host);

  try {
    await act(async () => root.render(<FilePreviewModal project="/project" path={path} onClose={() => {}} />));
    expect(readPreview).toHaveBeenCalledWith('/project', path);
    const article = host.querySelector('article');
    expect(article?.querySelector('h1')?.textContent).toBe('Proyecto');
    expect(article?.querySelector('strong')?.textContent).toBe('Primer');
    expect(article?.querySelector('li')?.textContent).toBe('Ejemplo');
    expect(article?.querySelector('[aria-label="Tabla del documento"] td')?.textContent).toBe('Merlin');
    expect(article?.querySelector('pre code')?.textContent).toContain('const example = true;');
    expect(article?.querySelector('a[href], br, script')).toBeNull();
  } finally {
    await act(async () => root.unmount());
    host.remove();
    Object.assign(dialog, previousMethods);
    vi.unstubAllGlobals();
  }
});
