import React, { useEffect, useMemo, useState } from 'react';
import { diffLines } from 'diff';
import { ToolApprovalRequest } from '../../../types';
import { IconFile, IconCheck, IconClose } from '../../../components/Icons';

interface ToolApprovalCardProps {
  request: ToolApprovalRequest;
  onApprove: (editedContent: string) => void;
  onReject: () => void;
}

function formatCountdown(msRemaining: number): string {
  const totalSeconds = Math.max(0, Math.floor(msRemaining / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
}

export const ToolApprovalCard: React.FC<ToolApprovalCardProps> = ({ request, onApprove, onReject }) => {
  const [editedContent, setEditedContent] = useState(request.newContent || '');
  const [isEditing, setIsEditing] = useState(false);
  const [msRemaining, setMsRemaining] = useState(() => request.expiresAt - Date.now());

  useEffect(() => {
    setEditedContent(request.newContent || '');
  }, [request.requestId, request.newContent]);

  useEffect(() => {
    const interval = setInterval(() => {
      setMsRemaining(request.expiresAt - Date.now());
    }, 1000);
    return () => clearInterval(interval);
  }, [request.expiresAt]);

  const diffParts = useMemo(
    () => diffLines(request.oldContent || '', editedContent || ''),
    [request.oldContent, editedContent]
  );

  const isExpired = msRemaining <= 0;
  const hasEdits = editedContent !== (request.newContent || '');

  return (
    <div className="mt-3 rounded-lg border border-amber-500/40 bg-[#12191d] overflow-hidden">
      <div className="flex items-center gap-2 px-3 py-2 bg-amber-950/20 border-b border-amber-500/30">
        <IconFile size={14} className="text-amber-400" />
        <span className="text-xs font-mono font-semibold text-amber-300">{request.path}</span>
        <button
          type="button"
          onClick={() => setIsEditing((v) => !v)}
          className="text-[10px] font-mono text-content-dim hover:text-content-body uppercase tracking-wider cursor-pointer"
        >
          {isEditing ? 'Ver diff' : 'Editar antes de aprobar'}
        </button>
        <span
          className={`ml-auto text-[10px] font-mono uppercase tracking-wider ${
            isExpired ? 'text-red-400' : 'text-amber-400/70'
          }`}
        >
          {isExpired ? 'Expirada' : `Expira en ${formatCountdown(msRemaining)}`}
        </span>
      </div>

      {isEditing ? (
        <textarea
          value={editedContent}
          onChange={(e) => setEditedContent(e.target.value)}
          spellCheck={false}
          className="w-full h-64 resize-none bg-[#0b0e10] text-content-body font-mono text-[11px] leading-relaxed p-3 outline-none"
        />
      ) : (
        <div className="max-h-64 overflow-y-auto font-mono text-[11px] leading-relaxed">
          {diffParts.map((part, idx) => {
            const lines = part.value.replace(/\n$/, '').split('\n');
            const type = part.added ? 'add' : part.removed ? 'remove' : 'context';
            return lines.map((line, lineIdx) => (
              <div
                key={`${idx}-${lineIdx}`}
                className={`px-3 py-0.5 whitespace-pre-wrap ${
                  type === 'add'
                    ? 'bg-emerald-950/30 text-emerald-300'
                    : type === 'remove'
                    ? 'bg-red-950/30 text-red-300'
                    : 'text-content-dim'
                }`}
              >
                <span className="select-none mr-2 opacity-60">
                  {type === 'add' ? '+' : type === 'remove' ? '-' : ' '}
                </span>
                {line || ' '}
              </div>
            ));
          })}
        </div>
      )}

      <div className="flex items-center gap-2 px-3 py-2.5 border-t border-border-subtle/40">
        <button
          type="button"
          disabled={isExpired}
          onClick={() => onApprove(editedContent)}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-accent-primary hover:bg-accent-primary-hover disabled:opacity-40 disabled:cursor-not-allowed text-[#0b0e10] font-semibold text-xs rounded-md transition-colors cursor-pointer"
        >
          <IconCheck size={13} />
          <span>{hasEdits ? 'Aprobar con cambios' : 'Aprobar'}</span>
        </button>
        <button
          type="button"
          onClick={onReject}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-[#1a252b] hover:bg-[#233038] border border-border-subtle/50 text-content-body font-semibold text-xs rounded-md transition-colors cursor-pointer"
        >
          <IconClose size={13} />
          <span>Rechazar</span>
        </button>
      </div>
    </div>
  );
};
