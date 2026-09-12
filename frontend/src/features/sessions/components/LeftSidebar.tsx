import React from 'react';
import { Session } from '../../../types';
import { IconPlus, IconPanelLeft } from '../../../components/Icons';
import { ProjectFileTree } from '../../projects/components/ProjectFileTree';
import { useResizablePanel } from '../../../hooks/useResizablePanel';
import { SessionItem } from './SessionItem';

export interface LeftSidebarProps {
  isOpen: boolean;
  onToggle: () => void;
  width: number;
  onWidthChange: (width: number) => void;
  sessions: Session[];
  activeSessionId: string;
  onSelectSession: (id: string) => void;
  onNewSession: () => void;
  onDeleteSession: (id: string, e: React.MouseEvent) => void;
  searchQuery?: string;
  projectName?: string;
  projectPath?: string;
  onOpenFolder?: () => void;
}

export const LeftSidebar: React.FC<LeftSidebarProps> = ({
  isOpen,
  onToggle,
  width,
  onWidthChange,
  sessions,
  activeSessionId,
  onSelectSession,
  onNewSession,
  onDeleteSession,
  searchQuery = '',
  projectName = '',
  projectPath,
  onOpenFolder,
}) => {
  const { handleMouseDown } = useResizablePanel({
    minWidth: 180,
    maxWidth: 360,
    side: 'left',
    onWidthChange,
  });

  const trimmedSearch = (searchQuery || '').trim().toLowerCase();
  const filteredSessions = sessions.filter((s) => {
    if (!trimmedSearch) return true;
    return s.title.toLowerCase().includes(trimmedSearch);
  });

  return (
    <aside
      className={`relative h-full bg-surface border-r border-border-subtle flex flex-col select-none shrink-0 overflow-hidden ${
        !isOpen ? 'w-11 transition-all duration-200' : ''
      }`}
      style={{
        width: isOpen ? `${width}px` : undefined,
        minWidth: isOpen ? `${width}px` : undefined,
        maxWidth: isOpen ? '360px' : undefined,
      }}
    >
      {/* Botón para expandir cuando está colapsado */}
      {!isOpen ? (
        <button
          type="button"
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
        <div className="h-full flex flex-col w-full relative">
          {/* Resize Handle en el borde derecho */}
          <div
            className="absolute top-0 right-0 w-1.5 h-full cursor-col-resize hover:bg-accent-primary/60 active:bg-accent-primary transition-colors z-20 select-none"
            onMouseDown={handleMouseDown}
            title="Arrastrar para redimensionar panel (máximo 360px)"
          />

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
              type="button"
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
              type="button"
              className="w-full flex items-center justify-center gap-2 py-1.5 px-3 bg-accent-primary hover:bg-accent-primary-hover text-[#0b0e10] font-semibold text-xs rounded transition-colors cursor-pointer shadow-sm"
              onClick={onNewSession}
            >
              <IconPlus size={14} />
              <span>Nueva Sesión</span>
            </button>
          </div>

          {/* Lista de sesiones */}
          <div className="flex-1 overflow-y-auto px-2 py-1 flex flex-col">
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
                {filteredSessions.map((s) => (
                  <SessionItem
                    key={s.id}
                    session={s}
                    isActive={s.id === activeSessionId}
                    onSelect={onSelectSession}
                    onDelete={onDeleteSession}
                  />
                ))}
              </ul>
            )}
          </div>

          {/* Subpanel de Árbol de Archivos en la parte inferior */}
          <ProjectFileTree
            projectPath={projectPath}
            projectName={projectName}
            onOpenFolder={onOpenFolder}
          />
        </div>
      )}
    </aside>
  );
};
