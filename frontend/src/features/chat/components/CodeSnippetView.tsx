import React from 'react';
import { IconCopy, IconCheck } from '../../../components/Icons';
import { useClipboard } from '../../../hooks/useClipboard';

interface CodeSnippetViewProps {
  code: string;
  language?: string;
  filename?: string;
  id?: string;
  messageId?: string;
}

export const CodeSnippetView: React.FC<CodeSnippetViewProps> = ({
  code,
  language,
  filename,
  id,
  messageId,
}) => {
  const { copy, isCopied } = useClipboard();
  const copyKey = id || (messageId ? `code-${messageId}` : `code-${code.slice(0, 24)}`);

  const cleanLang = (language || '').trim();
  const displayTitle = filename || (cleanLang ? cleanLang.toUpperCase() : 'CÓDIGO');

  return (
    <div className="my-3 rounded-xl border border-border-subtle/70 bg-[#0b1013] overflow-hidden shadow-lg shadow-black/25">
      {/* Barra superior con lenguaje y botón copiar en la esquina superior derecha */}
      <div className="flex items-center justify-between px-3.5 py-2 bg-[#12191d] border-b border-border-subtle/50 select-none">
        <span className="text-[11px] font-mono font-semibold text-accent-primary/90 tracking-wider">
          {displayTitle}
        </span>
        <button
          type="button"
          className={`inline-flex items-center gap-1.5 text-xs font-mono px-2 py-1 rounded-md transition-all cursor-pointer border ${
            isCopied(copyKey)
              ? 'bg-accent-primary/15 border-accent-primary/40 text-accent-primary'
              : 'bg-[#182226] hover:bg-[#223036] border-border-subtle/60 text-content-dim hover:text-content-body active:scale-95'
          }`}
          onClick={(e) => {
            e.stopPropagation();
            copy(copyKey, code.trimEnd());
          }}
          title="Copiar bloque de código"
          aria-label="Copiar bloque de código"
        >
          {isCopied(copyKey) ? (
            <>
              <IconCheck size={13} className="text-accent-primary" />
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

      {/* Bloque de código con scroll horizontal */}
      <pre className="p-4 overflow-x-auto text-[13px] font-mono text-[#f1f5f9] leading-relaxed selection:bg-accent-primary/20">
        <code>{code}</code>
      </pre>
    </div>
  );
};
