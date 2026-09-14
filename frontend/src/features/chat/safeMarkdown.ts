import DOMPurify from 'dompurify';
import { marked } from 'marked';

export function renderSafeMarkdown(content: string): string {
  return DOMPurify.sanitize(marked.parse(content, { async: false, breaks: true, gfm: true }), {
    ALLOWED_TAGS: ['p', 'br', 'strong', 'em', 'del', 's', 'blockquote', 'pre', 'code',
      'ul', 'ol', 'li', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'hr', 'table', 'thead',
      'tbody', 'tr', 'th', 'td', 'a'],
    ALLOWED_ATTR: ['href', 'title', 'start', 'colspan', 'rowspan'],
    ALLOW_DATA_ATTR: false,
    ALLOW_ARIA_ATTR: false,
    ALLOWED_URI_REGEXP: /^https?:\/\//i,
  });
}
