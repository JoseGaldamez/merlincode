import React, { useState } from 'react';
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
  const [isOpen, setIsOpen] = useState(true);

  if (!activity || activity.length === 0) return null;

  return (
    <div className="w-full mb-2.5 rounded-lg border border-border-petrol bg-[#10171b] overflow-hidden">
      <button
        type="button"
        className="w-full flex items-center gap-2 px-3 py-2 text-xs text-content-muted hover:text-content-body hover:bg-[#152227] transition-colors cursor-pointer text-left"
        onClick={() => setIsOpen((prev) => !prev)}
      >
        {isOpen ? <IconChevronDown size={14} /> : <IconChevronRight size={14} />}
        <span className="text-[11px] font-mono text-accent-cyan tracking-wider font-semibold">
          ACTIVIDAD DE HERRAMIENTAS
        </span>
      </button>
      {isOpen && (
        <div className="border-t border-border-petrol divide-y divide-border-petrol/60">
          {activity.map((entry) => (
            <div
              key={entry.toolCallId}
              className="flex items-center gap-2.5 px-3 py-2 text-xs font-mono bg-[#0a0f12]"
            >
              <IconFile size={13} className="text-content-dim shrink-0" />
              <span className="text-content-body font-semibold shrink-0">{entry.toolName}</span>
              {entry.argsSummary && (
                <span className="text-content-dim truncate">{entry.argsSummary}</span>
              )}
              <span className="ml-auto flex items-center gap-1.5 text-content-dim shrink-0">
                <StatusIcon status={entry.status} />
                <span>{STATUS_LABEL[entry.status]}</span>
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
