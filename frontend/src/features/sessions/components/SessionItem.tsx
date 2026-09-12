import React from 'react';
import { Session } from '../../../types';
import { IconMessage, IconTrash } from '../../../components/Icons';

interface SessionItemProps {
  session: Session;
  isActive: boolean;
  onSelect: (id: string) => void;
  onDelete: (id: string, e: React.MouseEvent) => void;
}

export const SessionItem: React.FC<SessionItemProps> = ({
  session,
  isActive,
  onSelect,
  onDelete,
}) => {
  return (
    <li
      className={`group flex items-center justify-between px-2.5 py-2 rounded text-xs transition-colors cursor-pointer border ${
        isActive
          ? 'bg-[#18262c] text-accent-primary border-border-petrol font-medium'
          : 'text-content-muted hover:bg-[#151c20] hover:text-content-body border-transparent'
      }`}
      onClick={() => onSelect(session.id)}
    >
      <div className="flex items-center gap-2.5 min-w-0 flex-1 pr-2">
        <IconMessage
          size={14}
          className={`shrink-0 ${isActive ? 'text-accent-primary' : 'text-content-dim'}`}
        />
        <div className="min-w-0 flex-1">
          <span className="block truncate text-xs" title={session.title}>
            {session.title}
          </span>
          <span className="block text-[10px] text-content-dim font-mono mt-0.5">
            {session.date}
          </span>
        </div>
      </div>

      <button
        type="button"
        className="opacity-0 group-hover:opacity-100 p-1 text-content-dim hover:text-status-error hover:bg-[#25181a] rounded transition-all cursor-pointer"
        onClick={(e) => onDelete(session.id, e)}
        title="Eliminar sesión"
      >
        <IconTrash size={13} />
      </button>
    </li>
  );
};
