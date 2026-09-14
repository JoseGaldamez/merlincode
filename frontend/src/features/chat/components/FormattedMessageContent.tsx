import React, { useMemo } from 'react';
import { marked } from 'marked';
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
}> = ({ content, messageId }) => {
  const blocks = useMemo(() => parseMarkdownBlocks(content), [content]);

  return (
    <div className="space-y-3 text-content-body">
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

        const html = marked.parse(block.content, {
          async: false,
          breaks: true,
          gfm: true,
        }) as string;

        return (
          <div
            key={`text-${messageId}-${idx}`}
            className="markdown-body"
            dangerouslySetInnerHTML={{ __html: html }}
          />
        );
      })}
    </div>
  );
};
