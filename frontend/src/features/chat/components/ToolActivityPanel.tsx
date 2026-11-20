import React, { useId, useState } from 'react';
import { ToolActivityEntry } from '../../../types';
import { IconChevronDown, IconChevronRight, IconFile, IconCheck, IconClose } from '../../../components/Icons';

interface ToolActivityPanelProps {
  activity: ToolActivityEntry[];
}

const STATUS_LABEL: Record<ToolActivityEntry['status'], string> = {
  running: 'Ejecutando...',
  awaiting_approval: 'Esperando aprobación',
  done: 'Completado',
  error: 'Error',
  rejected: 'Rechazado',
};

const StatusIcon: React.FC<{ status: ToolActivityEntry['status'] }> = ({ status }) => {
  if (status === 'done') return <IconCheck size={13} className="text-accent-primary" />;
  if (status === 'error' || status === 'rejected') return <IconClose size={13} className="text-red-400" />;
  return <span className="w-2 h-2 rounded-full bg-accent-cyan animate-pulse" />;
};

export const ToolActivityPanel: React.FC<ToolActivityPanelProps> = ({ activity }) => {
  const [isOpen, setIsOpen] = useState(false);
  const panelId = useId();

  if (!activity || activity.length === 0) return null;

  return (
    <div className="w-full mb-1">
      <button
        type="button"
        className="inline-flex items-center gap-1.5 rounded-md px-1.5 py-1 text-xs text-content-dim hover:text-content-body hover:bg-surface-hover focus-visible:outline-2 focus-visible:outline-accent-primary transition-colors"
        aria-expanded={isOpen}
        aria-controls={panelId}
        onClick={() => setIsOpen((prev) => !prev)}
      >
        <span aria-hidden="true">
          {isOpen ? <IconChevronDown size={12} /> : <IconChevronRight size={12} />}
        </span>
        <span>Actividad de herramientas</span>
        <span className="font-mono text-[11px]">({activity.length})</span>
      </button>
      <div id={panelId} hidden={!isOpen} className="mt-1 overflow-hidden rounded-md border border-border-subtle bg-surface-lowest divide-y divide-border-subtle">
        {isOpen && activity.map((entry) => (
            <div key={entry.toolCallId}>
              <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 px-2.5 py-1.5 text-[11px] font-mono">
                <div className="flex min-w-0 items-center gap-2">
                  <IconFile size={12} className="text-content-dim shrink-0" />
                  <span className="text-content-muted shrink-0">{entry.toolName}</span>
                  {entry.argsSummary && (
                    <span className="text-content-dim truncate">{entry.argsSummary}</span>
                  )}
                </div>
                <span className="flex items-center gap-1.5 text-content-dim">
                  <StatusIcon status={entry.status} />
                  <span>{STATUS_LABEL[entry.status]}</span>
                </span>
              </div>
              {entry.error && <p role="alert" className="px-2.5 pb-2 text-xs leading-relaxed text-red-300 break-words">{entry.error}</p>}
            </div>
        ))}
      </div>
    </div>
  );
};
