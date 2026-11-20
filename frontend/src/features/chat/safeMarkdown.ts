import DOMPurify from 'dompurify';
import { marked } from 'marked';

const webLink = /^https?:\/\//i;
const localOrWebLink = /^(?:https?:\/\/|(?:[a-z]:[\\/])?[^:]+(?::\d+){0,2}$)/i;
const fileMention = /(?<![\p{L}\p{N}_./\\:@%-])(?:[a-z]:)?[\p{L}\p{N}_./\\-]*(?:\.(?:astro|avif|bash|bmp|c|cc|conf|cpp|cs|css|csv|diff|env|gif|go|h|hpp|htm|html|ico|ini|java|jpe?g|js|jsonc?|jsx|kt|log|lua|markdown|md|mdx|mjs|cjs|pdf|php|png|ps1|py|rb|rs|sass|scss|sh|sql|svelte|svg|swift|tiff?|toml|ts|tsx|txt|vue|webp|xml|ya?ml)|\.gitignore|Dockerfile|Makefile)(?::\d+(?::\d+)?|#L\d+(?:-L?\d+)?)?(?![\p{L}\p{N}_/\\@-]|\.[\p{L}\p{N}_])/giu;

function previewPath(value: string, encoded = false): string | undefined {
  const target = value.trim().replace(/#L\d+(?:-L?\d+)?$|:\d+(?::\d+)?$/i, '');
  try {
    const path = (encoded ? decodeURIComponent(target) : target).replace(/\\/g, '/');
    const segments = path.replace(/^[a-z]:\//i, '').split('/');
    if (!path || path.startsWith('//') || !segments.at(-1) || segments.includes('..')) return;
    if (segments.some((segment) => /[:?#*<>|"\u0000-\u001f\u007f]/.test(segment))) return;
    if (segments.some((segment) => segment !== '' && segment !== '.')) return path;
  } catch {
    // Malformed percent encoding does not identify a file.
  }
}

function fileButton(doc: Document, path: string) {
  const button = doc.createElement('button');
  button.type = 'button';
  button.className = 'markdown-file-link';
  button.dataset.previewPath = path;
  return button;
}

function addFileLinks(fragment: DocumentFragment) {
  const doc = fragment.ownerDocument;
  for (const anchor of fragment.querySelectorAll('a[href]')) {
    const href = anchor.getAttribute('href') ?? '';
    if (webLink.test(href)) continue;
    const path = previewPath(href, true);
    if (!path) {
      anchor.removeAttribute('href');
      continue;
    }
    const button = fileButton(doc, path);
    button.append(...anchor.childNodes);
    anchor.replaceWith(button);
  }

  for (const code of fragment.querySelectorAll('code')) {
    if (code.closest('a, button, pre')) continue;
    const path = previewPath(code.textContent ?? '');
    if (!path || !/(?:\.[\p{L}][\p{L}\p{N}_-]*|(?:^|\/)(?:Dockerfile|Makefile))$/iu.test(path)) continue;
    const button = fileButton(doc, path);
    code.replaceWith(button);
    button.append(code);
  }

  const walker = doc.createTreeWalker(fragment, NodeFilter.SHOW_TEXT);
  const mentions: Text[] = [];
  for (let node = walker.nextNode(); node; node = walker.nextNode()) {
    if (!node.parentElement?.closest('a, button, code, pre')) mentions.push(node as Text);
  }
  for (const node of mentions) {
    const replacement = doc.createDocumentFragment();
    let end = 0;
    for (const match of node.data.matchAll(fileMention)) {
      const path = previewPath(match[0]);
      if (!path) continue;
      replacement.append(node.data.slice(end, match.index));
      const button = fileButton(doc, path);
      button.textContent = match[0];
      replacement.append(button);
      end = match.index + match[0].length;
    }
    if (end) {
      replacement.append(node.data.slice(end));
      node.replaceWith(replacement);
    }
  }
}

interface MarkdownOptions {
  mode?: 'chat' | 'document';
  enableFileLinks?: boolean;
}

export function renderSafeMarkdown(content: string, { mode = 'chat', enableFileLinks = false }: MarkdownOptions = {}): string {
  const isChat = mode === 'chat';
  const fragment = DOMPurify.sanitize(marked.parse(content, { async: false, breaks: isChat, gfm: true }), {
    ALLOWED_TAGS: ['p', 'br', 'strong', 'em', 'del', 's', 'blockquote', 'pre', 'code',
      'ul', 'ol', 'li', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'hr', 'table', 'thead',
      'tbody', 'tr', 'th', 'td', 'a'],
    ALLOWED_ATTR: ['href', 'title', 'start', 'colspan', 'rowspan'],
    ADD_ATTR: (name, tag) => name === 'align' && (tag === 'th' || tag === 'td'),
    ADD_URI_SAFE_ATTR: ['align'],
    ALLOW_DATA_ATTR: false,
    ALLOW_ARIA_ATTR: false,
    ALLOWED_URI_REGEXP: enableFileLinks ? localOrWebLink : webLink,
    RETURN_DOM_FRAGMENT: true,
  });

  if (enableFileLinks) addFileLinks(fragment);

  const container = fragment.ownerDocument.createElement('div');
  for (const table of fragment.querySelectorAll('table')) {
    const wrapper = fragment.ownerDocument.createElement('div');
    wrapper.className = 'markdown-table';
    wrapper.tabIndex = 0;
    wrapper.setAttribute('role', 'region');
    wrapper.setAttribute('aria-label', isChat ? 'Tabla de la respuesta' : 'Tabla del documento');
    table.replaceWith(wrapper);
    wrapper.append(table);
  }
  container.append(fragment);
  return container.innerHTML;
}
