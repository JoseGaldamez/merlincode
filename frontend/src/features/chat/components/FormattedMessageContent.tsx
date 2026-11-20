import React, { useMemo } from 'react';
import { renderSafeMarkdown } from '../safeMarkdown';
import { CodeSnippetView } from './CodeSnippetView';

interface ContentBlock {
  type: 'text' | 'code';
  content: string;
  language?: string;
}

export function parseMarkdownBlocks(text: string): ContentBlock[] {
  if (!text) return [];

  const blocks: ContentBlock[] = [];
  const codeBlockRegex = /```([a-zA-Z0-9_-]*)\n([\s\S]*?)(?:```|$)/g;
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = codeBlockRegex.exec(text)) !== null) {
    const textBefore = text.slice(lastIndex, match.index);
    if (textBefore.trim()) {
      blocks.push({ type: 'text', content: textBefore });
    }
    const language = match[1]?.trim() || '';
    const code = match[2] || '';
    blocks.push({
      type: 'code',
      language,
      content: code,
    });
    lastIndex = match.index + match[0].length;
  }

  const remaining = text.slice(lastIndex);
  if (remaining.trim() || blocks.length === 0) {
    blocks.push({ type: 'text', content: remaining });
  }

  return blocks;
}

export const FormattedMessageContent: React.FC<{
  content: string;
  messageId: string;
  onPreviewFile?: (path: string) => void;
}> = ({ content, messageId, onPreviewFile }) => {
  const enableFileLinks = Boolean(onPreviewFile);
  const blocks = useMemo(() => parseMarkdownBlocks(content).map((block) => (
    block.type === 'text'
      ? { ...block, html: renderSafeMarkdown(block.content, { enableFileLinks }) }
      : { ...block, html: '' }
  )), [content, enableFileLinks]);

  return (
    <div className="space-y-3 text-content-body" onClick={onPreviewFile && ((event) => {
      if (!(event.target instanceof Element)) return;
      const path = event.target.closest<HTMLButtonElement>('button[data-preview-path]')?.dataset.previewPath;
      if (path) onPreviewFile(path);
    })}>
      {blocks.map((block, idx) => {
        if (block.type === 'code') {
          return (
            <CodeSnippetView
              key={`code-${messageId}-${idx}`}
              code={block.content}
              language={block.language}
              id={`code-${messageId}-${idx}`}
            />
          );
        }

        return (
          <div
            key={`text-${messageId}-${idx}`}
            className="markdown-body"
            dangerouslySetInnerHTML={{ __html: block.html }}
          />
        );
      })}
    </div>
  );
};
