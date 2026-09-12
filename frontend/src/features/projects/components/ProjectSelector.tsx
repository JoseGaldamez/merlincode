import React from 'react';
import { Project } from '../../../types';
import { IconFolder, IconChevronDown } from '../../../components/Icons';

interface ProjectSelectorProps {
  projects: Project[];
  activeProjectId: string;
  onSelectProject: (id: string) => void;
}

export const ProjectSelector: React.FC<ProjectSelectorProps> = ({
  projects,
  activeProjectId,
  onSelectProject,
}) => {
  return (
    <div className="wails-no-drag flex items-center">
      <div className="relative flex items-center bg-[#141b1f] border border-border-subtle hover:border-border-petrol focus-within:border-accent-primary rounded transition-colors">
        <IconFolder size={13} className="absolute left-2 text-content-dim pointer-events-none" />
        <select
          className="appearance-none bg-transparent pl-7 pr-6 py-1 text-xs text-content-body font-mono outline-none cursor-pointer"
          value={activeProjectId || ''}
          onChange={(e) => onSelectProject(e.target.value)}
          aria-label="Seleccionar proyecto"
        >
          {!activeProjectId && (
            <option value="" disabled className="bg-[#141b1f] text-content-dim">
              Seleccionar carpeta...
            </option>
          )}
          {projects.map((p) => (
            <option key={p.id} value={p.id} className="bg-[#141b1f] text-content-body" title={p.path}>
              {p.name}
            </option>
          ))}
          <option value="__open_folder__" className="bg-[#141b1f] text-accent-primary font-semibold">
            + Abrir carpeta...
          </option>
        </select>
        <IconChevronDown size={11} className="absolute right-2 text-content-dim pointer-events-none" />
      </div>
    </div>
  );
};
