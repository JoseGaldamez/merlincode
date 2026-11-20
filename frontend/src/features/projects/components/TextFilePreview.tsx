import { Fragment, useMemo } from 'react';
import type { CSSProperties } from 'react';
import hljs from 'highlight.js/lib/common';
import dockerfile from 'highlight.js/lib/languages/dockerfile';
import powershell from 'highlight.js/lib/languages/powershell';
import 'highlight.js/styles/github-dark.css';

hljs.registerLanguage('dockerfile', dockerfile);
hljs.registerLanguage('powershell', powershell);

const languageAliases = new Map([
  ['astro', 'xml'], ['svelte', 'xml'], ['vue', 'xml'], ['mdx', 'markdown'],
  ['jsx', 'javascript'], ['mjs', 'javascript'], ['cjs', 'javascript'],
  ['tsx', 'typescript'], ['mts', 'typescript'], ['cts', 'typescript'],
  ['jsonc', 'json'], ['ps1', 'powershell'], ['env', 'ini'],
  ['h', 'c'], ['hpp', 'cpp'], ['pyw', 'python'],
]);

// Limit the number of syntax spans produced for large or generated sources.
const MAX_HIGHLIGHT_LENGTH = 512 * 1024;
// A source file can contain millions of short lines within the workspace size limit.
const MAX_NUMBERED_LINES = 50_000;

function splitHighlightedLines(html: string) {
  const lines: string[] = [];
  const openSpans: string[] = [];
  let lineStart = 0;
  let prefix = '';
  // Highlight.js escapes source text and emits span tags for syntax tokens.
  for (const match of html.matchAll(/<span\b[^>]*>|<\/span>|\n/g)) {
    const token = match[0];
    if (token === '\n') {
      lines.push(prefix + html.slice(lineStart, match.index) + '</span>'.repeat(openSpans.length));
      lineStart = match.index + 1;
      prefix = openSpans.join('');
    } else if (token === '</span>') {
      openSpans.pop();
    } else {
      openSpans.push(token);
    }
  }
  lines.push(prefix + html.slice(lineStart));
  return lines;
}

export default function TextFilePreview({ name, text }: { name: string; text: string }) {
  const source = useMemo(() => {
    const extension = name.toLowerCase().split('.').at(-1) ?? '';
    const language = languageAliases.get(extension) ?? extension;
    const grammar = hljs.getLanguage(language);
    if (!grammar || grammar === hljs.getLanguage('plaintext')) return;

    const code = text.replace(/\r\n?/g, '\n');
    const lines = code.split('\n', MAX_NUMBERED_LINES + 1);
    if (lines.length > MAX_NUMBERED_LINES) return;
    if (text.length > MAX_HIGHLIGHT_LENGTH) return { lines, highlighted: false };
    return { lines: splitHighlightedLines(hljs.highlight(code, { language }).value), highlighted: true };
  }, [name, text]);
  if (!text) return <p className="file-preview-status" role="status">El archivo está vacío.</p>;
  if (!source) return <pre className="file-preview-text"><code>{text}</code></pre>;

  const { lines, highlighted } = source;

  return (
    <pre className="file-preview-text">
      <code style={{ '--file-preview-line-digits': String(lines.length).length + 'ch' } as CSSProperties}>
        {lines.map((line, index) => (
          <Fragment key={index}>
            <span
              className="file-preview-code-line"
              data-line={index + 1}
              dangerouslySetInnerHTML={highlighted ? { __html: line } : undefined}
            >
              {highlighted ? undefined : line}
            </span>
            {index + 1 < lines.length ? '\n' : null}
          </Fragment>
        ))}
      </code>
    </pre>
  );
}
