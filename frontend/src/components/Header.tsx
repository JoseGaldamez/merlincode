import React, { useState, useEffect } from 'react';
import logotipo from '../assets/images/logotipo.png';
import {
  IconFolder,
  IconChevronDown,
  IconSearch,
  IconClose,
  IconSettings,
  IconMinus,
  IconSquare,
  IconRestore,
} from './Icons';
import { Project, AgentTelemetry } from '../types';
import {
  WindowMinimise,
  WindowToggleMaximise,
  Quit,
  WindowIsMaximised,
} from '../../wailsjs/runtime/runtime';

interface HeaderProps {
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
  const [isMaximized, setIsMaximized] = useState(false);

  useEffect(() => {
    WindowIsMaximised().then(setIsMaximized).catch(() => {});
    const handleResize = () => {
      WindowIsMaximised().then(setIsMaximized).catch(() => {});
    };
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  const handleMinimize = () => {
    WindowMinimise();
  };

  const handleToggleMaximize = () => {
    WindowToggleMaximise();
    setTimeout(() => {
      WindowIsMaximised().then(setIsMaximized).catch(() => {});
    }, 150);
  };

  const handleClose = () => {
    Quit();
  };

  return (
    <header className="wails-drag h-10 bg-[#0f1416] border-b border-border-subtle flex items-center justify-between select-none shrink-0 z-30">
      {/* 1) Logotipo reducido y 2) Selector de proyectos al lado */}
      <div className="flex items-center gap-3 pl-3.5 h-full">
        <div className="wails-no-drag flex items-center">
          <img src={logotipo} alt="Merlin Code" className="h-5 w-auto object-contain" />
        </div>

        <div className="wails-no-drag flex items-center">
          <div className="relative flex items-center bg-[#141b1f] border border-border-subtle hover:border-border-petrol focus-within:border-accent-primary rounded transition-colors">
            <IconFolder size={13} className="absolute left-2 text-content-dim pointer-events-none" />
            <select
              className="appearance-none bg-transparent pl-7 pr-6 py-1 text-xs text-content-body font-mono outline-none cursor-pointer"
              value={activeProjectId}
              onChange={(e) => onSelectProject(e.target.value)}
              aria-label="Seleccionar proyecto"
            >
              {projects.map((p) => (
                <option key={p.id} value={p.id} className="bg-[#141b1f] text-content-body">
                  {p.name}
                </option>
              ))}
              <option value="__new__" className="bg-[#141b1f] text-accent-primary">
                + Nuevo proyecto...
              </option>
            </select>
            <IconChevronDown size={11} className="absolute right-2 text-content-dim pointer-events-none" />
          </div>
        </div>
      </div>

      {/* 3) Cuadro de búsqueda para encontrar palabras en las sesiones */}
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

      {/* 4) Botón de configuración y botones de ventana (minimizar, maximizar, cerrar) */}
      <div className="wails-no-drag flex items-center h-full">
        <button
          className="p-1.5 mr-1 text-content-muted hover:text-accent-primary hover:bg-[#182328] rounded transition-colors cursor-pointer"
          onClick={onOpenSettings}
          title="Configuración"
          aria-label="Abrir configuración"
        >
          <IconSettings size={15} />
        </button>

        <div className="flex items-center h-full">
          <button
            className="h-full px-3.5 flex items-center justify-center text-content-muted hover:text-content-headline hover:bg-[#1a2429] transition-colors cursor-pointer"
            onClick={handleMinimize}
            title="Minimizar"
            aria-label="Minimizar ventana"
          >
            <IconMinus size={11} />
          </button>
          <button
            className="h-full px-3.5 flex items-center justify-center text-content-muted hover:text-content-headline hover:bg-[#1a2429] transition-colors cursor-pointer"
            onClick={handleToggleMaximize}
            title={isMaximized ? 'Restaurar' : 'Maximizar'}
            aria-label={isMaximized ? 'Restaurar ventana' : 'Maximizar ventana'}
          >
            {isMaximized ? <IconRestore size={11} /> : <IconSquare size={11} />}
          </button>
          <button
            className="h-full px-3.5 flex items-center justify-center text-content-muted hover:text-white hover:bg-[#c42b1c] transition-colors cursor-pointer"
            onClick={handleClose}
            title="Cerrar"
            aria-label="Cerrar ventana"
          >
            <IconClose size={13} />
          </button>
        </div>
      </div>
    </header>
  );
};
