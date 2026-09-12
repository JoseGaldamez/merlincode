import React from 'react';
import { IconCopy, IconCheck } from '../../../components/Icons';
import { useClipboard } from '../../../hooks/useClipboard';

interface CodeSnippetViewProps {
  code: string;
  language?: string;
  filename?: string;
  messageId: string;
}

export const CodeSnippetView: React.FC<CodeSnippetViewProps> = ({
  code,
  language,
  filename,
  messageId,
}) => {
  const { copy, isCopied } = useClipboard();
  const copyKey = `code-${messageId}`;

  return (
    <div className="mt-3 rounded-lg border border-border-subtle bg-[#0c1012] overflow-hidden">
      <div className="flex items-center justify-between px-3.5 py-1.5 bg-[#141b1e] border-b border-border-subtle">
        <span className="text-xs font-mono text-content-dim uppercase tracking-wider">
          {filename || language || 'CODE'}
        </span>
        <button
          type="button"
          className="flex items-center gap-1.5 text-xs font-mono text-content-dim hover:text-accent-primary transition-colors cursor-pointer"
          onClick={() => copy(copyKey, code)}
          title="Copiar código"
        >
          {isCopied(copyKey) ? (
            <>
              <IconCheck size={13} className="text-status-success" />
              <span>Copiado</span>
            </>
          ) : (
            <>
              <IconCopy size={13} />
              <span>Copiar</span>
            </>
          )}
        </button>
      </div>
      <pre className="p-3.5 overflow-x-auto text-[13.5px] font-mono text-[#e2e8f0] leading-relaxed">
        <code>{code}</code>
      </pre>
    </div>
  );
};
