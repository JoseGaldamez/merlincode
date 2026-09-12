import React from 'react';
import { Session } from '../types';
import { IconPlus, IconMessage, IconTrash, IconPanelLeft } from './Icons';

interface LeftSidebarProps {
  isOpen: boolean;
  onToggle: () => void;
  sessions: Session[];
  activeSessionId: string;
  onSelectSession: (id: string) => void;
  onNewSession: () => void;
  onDeleteSession: (id: string, e: React.MouseEvent) => void;
  searchQuery?: string;
  projectName?: string;
}

export const LeftSidebar: React.FC<LeftSidebarProps> = ({
  isOpen,
  onToggle,
  sessions,
  activeSessionId,
  onSelectSession,
  onNewSession,
  onDeleteSession,
  searchQuery = '',
  projectName = 'merlincode',
}) => {
  return (
    <aside
      className={`h-full bg-surface border-r border-border-subtle flex flex-col select-none shrink-0 transition-all duration-200 overflow-hidden ${
        isOpen ? 'w-64' : 'w-11'
      }`}
    >
      {/* Botón en el mismo panel para expandir cuando está colapsado */}
      {!isOpen ? (
        <button
          className="w-full h-full flex flex-col items-center justify-start pt-3 gap-3 text-content-dim hover:text-accent-primary transition-colors cursor-pointer"
          onClick={onToggle}
          title="Expandir panel de sesiones"
          aria-label="Expandir panel de sesiones"
        >
          <IconPanelLeft size={16} />
          <span className="[writing-mode:vertical-lr] text-[10px] tracking-widest font-mono text-content-dim font-semibold mt-1">
            SESIONES
          </span>
        </button>
      ) : (
        <div className="h-full flex flex-col">
          {/* Encabezado del panel con botón de colapso */}
          <div className="flex items-center justify-between px-3.5 py-2.5 border-b border-border-subtle shrink-0">
            <div className="flex items-center gap-2">
              <span className="text-[11px] font-semibold tracking-wider text-content-dim font-mono">
                SESIONES
              </span>
              <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-[#1c282e] text-accent-primary font-medium">
                {sessions.length}
              </span>
            </div>
            <button
              className="p-1 text-content-dim hover:text-content-headline hover:bg-[#192226] rounded transition-colors cursor-pointer"
              onClick={onToggle}
              title="Colapsar panel"
              aria-label="Colapsar panel izquierdo"
            >
              <IconPanelLeft size={15} />
            </button>
          </div>

          {/* Botón de acción: Nueva Sesión */}
          <div className="p-3 pb-2 shrink-0">
            <button
              className="w-full flex items-center justify-center gap-2 py-1.5 px-3 bg-accent-primary hover:bg-accent-primary-hover text-[#0b0e10] font-semibold text-xs rounded transition-colors cursor-pointer shadow-sm"
              onClick={onNewSession}
            >
              <IconPlus size={14} />
              <span>Nueva Sesión</span>
            </button>
          </div>

          {/* Lista de sesiones */}
          <div className="flex-1 overflow-y-auto px-2 py-1 flex flex-col">
            {(() => {
              const trimmedSearch = (searchQuery || '').trim().toLowerCase();
              const filteredSessions = sessions.filter((s) => {
                if (!trimmedSearch) return true;
                return s.title.toLowerCase().includes(trimmedSearch);
              });

              return (
                <>
                  <div className="px-2 py-1.5 text-[10px] font-mono font-semibold tracking-wider text-content-dim">
                    {trimmedSearch
                      ? `RESULTADOS (${filteredSessions.length})`
                      : 'HISTORIAL RECIENTE'}
                  </div>
                  {filteredSessions.length === 0 ? (
                    <div className="px-3 py-6 text-xs font-mono text-content-dim text-center">
                      {trimmedSearch
                        ? `Sin resultados para "${searchQuery}"`
                        : 'Sin sesiones recientes'}
                    </div>
                  ) : (
                    <ul className="flex flex-col gap-1 list-none p-0 m-0">
                      {filteredSessions.map((s) => {
                        const isActive = s.id === activeSessionId;
                        return (
                          <li
                            key={s.id}
                            className={`group flex items-center justify-between px-2.5 py-2 rounded text-xs transition-colors cursor-pointer border ${
                              isActive
                                ? 'bg-[#18262c] text-accent-primary border-border-petrol font-medium'
                                : 'text-content-muted hover:bg-[#151c20] hover:text-content-body border-transparent'
                            }`}
                            onClick={() => onSelectSession(s.id)}
                          >
                            <div className="flex items-center gap-2.5 min-w-0 flex-1 pr-2">
                              <IconMessage
                                size={14}
                                className={`shrink-0 ${
                                  isActive ? 'text-accent-primary' : 'text-content-dim'
                                }`}
                              />
                              <div className="min-w-0 flex-1">
                                <span className="block truncate text-xs" title={s.title}>
                                  {s.title}
                                </span>
                                <span className="block text-[10px] text-content-dim font-mono mt-0.5">
                                  {s.date}
                                </span>
                              </div>
                            </div>

                            <button
                              className="opacity-0 group-hover:opacity-100 p-1 text-content-dim hover:text-status-error hover:bg-[#25181a] rounded transition-all cursor-pointer"
                              onClick={(e) => onDeleteSession(s.id, e)}
                              title="Eliminar sesión"
                            >
                              <IconTrash size={13} />
                            </button>
                          </li>
                        );
                      })}
                    </ul>
                  )}
                </>
              );
            })()}
          </div>

          {/* Pie del panel */}
          <div className="p-2.5 border-t border-border-subtle bg-[#0e1214] shrink-0">
            <div className="flex items-center gap-2 px-2.5 py-1.5 rounded bg-[#13191d] border border-border-subtle text-xs text-content-dim">
              <span className="w-2 h-2 rounded-full bg-status-success shrink-0" />
              <span className="truncate text-content-body font-mono text-[11px]">
                {projectName}
              </span>
            </div>
          </div>
        </div>
      )}
    </aside>
  );
};
