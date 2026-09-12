import React from 'react';
import logotipo from '../../../assets/images/logotipo.png';
import { IconSearch, IconClose, IconSettings } from '../../../components/Icons';
import { Project, AgentTelemetry } from '../../../types';
import { ProjectSelector } from '../../projects/components/ProjectSelector';
import { WindowControls } from './WindowControls';

export interface HeaderProps {
  projects: Project[];
  activeProjectId: string;
  onSelectProject: (id: string) => void;
  searchQuery: string;
  onSearchChange: (query: string) => void;
  onOpenSettings?: () => void;
  telemetry?: AgentTelemetry;
}

export const Header: React.FC<HeaderProps> = ({
  projects,
  activeProjectId,
  onSelectProject,
  searchQuery,
  onSearchChange,
  onOpenSettings,
}) => {
  return (
    <header className="wails-drag h-10 bg-[#0f1416] border-b border-border-subtle flex items-center justify-between select-none shrink-0 z-30">
      {/* Logotipo y Selector de proyectos */}
      <div className="flex items-center gap-3 pl-3.5 h-full">
        <div className="wails-no-drag flex items-center">
          <img src={logotipo} alt="Merlin Code" className="h-5 w-auto object-contain" />
        </div>

        <ProjectSelector
          projects={projects}
          activeProjectId={activeProjectId}
          onSelectProject={onSelectProject}
        />
      </div>

      {/* Cuadro de búsqueda para encontrar palabras en las sesiones */}
      <div className="wails-no-drag flex-1 flex justify-center max-w-sm mx-4">
        <div className="relative flex items-center w-full bg-[#13191d] border border-border-subtle rounded px-2.5 py-1 focus-within:border-accent-primary transition-colors">
          <IconSearch size={13} className="text-content-dim mr-2 shrink-0" />
          <input
            type="text"
            className="w-full bg-transparent text-xs text-content-headline placeholder-content-dim outline-none font-mono"
            placeholder="Buscar en sesiones..."
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
          />
          {searchQuery && (
            <button
              type="button"
              className="text-content-dim hover:text-content-headline p-0.5 rounded cursor-pointer transition-colors"
              onClick={() => onSearchChange('')}
              title="Limpiar búsqueda"
              aria-label="Limpiar búsqueda"
            >
              <IconClose size={12} />
            </button>
          )}
        </div>
      </div>

      {/* Botón de configuración y controles de ventana */}
      <div className="wails-no-drag flex items-center h-full">
        <button
          type="button"
          className="p-1.5 mr-1 text-content-muted hover:text-accent-primary hover:bg-[#182328] rounded transition-colors cursor-pointer"
          onClick={onOpenSettings}
          title="Configuración"
          aria-label="Abrir configuración"
        >
          <IconSettings size={15} />
        </button>

        <WindowControls />
      </div>
    </header>
  );
};
