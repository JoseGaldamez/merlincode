// @vitest-environment jsdom
import { renderToStaticMarkup } from 'react-dom/server';
import { expect, it } from 'vitest';
import TextFilePreview from './TextFilePreview';

function renderPreview(name: string, text: string) {
  const html = renderToStaticMarkup(<TextFilePreview name={name} text={text} />);
  return new DOMParser().parseFromString(html, 'text/html');
}

it('numbers source lines, including blank lines, and preserves multiline syntax', () => {
  const expected = ['/* Inicio', '   fin */', '', 'const sample = "<script>";', ''];
  const doc = renderPreview('example.ts', expected.join('\r\n'));
  const lines = [...doc.querySelectorAll('.file-preview-code-line')];

  expect(lines.map((line) => line.getAttribute('data-line'))).toEqual(['1', '2', '3', '4', '5']);
  expect(lines.map((line) => line.textContent)).toEqual(expected);
  expect(lines[0].querySelector('.hljs-comment')?.textContent).toBe(expected[0]);
  expect(lines[1].querySelector('.hljs-comment')?.textContent).toBe(expected[1]);
  expect(doc.querySelector('code')?.textContent).toBe(expected.join('\n'));
  expect(doc.querySelector('script')).toBeNull();
});

it('keeps regular text readable without a code gutter', () => {
  const doc = renderPreview('notes.txt', 'Primera línea\nSegunda línea');

  expect(doc.querySelector('[data-line]')).toBeNull();
  expect(doc.querySelector('code')?.textContent).toBe('Primera línea\nSegunda línea');
});
