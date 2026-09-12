import React, { useState } from 'react';
import { Project } from '../../../../types';
import { IconFolder, IconTrash, IconPlus, IconFolderOpen } from '../../../../components/Icons';

interface ProjectsTabProps {
  projects: Project[];
  activeProjectId: string;
  onSelectProject: (id: string) => void;
  onAddProject?: (name: string) => void;
  onDeleteProject?: (id: string) => void;
  onOpenFolder?: () => void;
  onCloseModal: () => void;
}

export const ProjectsTab: React.FC<ProjectsTabProps> = ({
  projects,
  activeProjectId,
  onSelectProject,
  onAddProject,
  onDeleteProject,
  onOpenFolder,
  onCloseModal,
}) => {
  const [newProjectName, setNewProjectName] = useState('');

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newProjectName.trim() || !onAddProject) return;
    onAddProject(newProjectName.trim());
    setNewProjectName('');
  };

  return (
    <div className="flex flex-col gap-9 max-w-3xl">
      <div>
        {/* Encabezado con botón de acción destacado */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-4">
          <div>
            <h3 className="text-xs font-mono font-semibold tracking-wider text-accent-primary uppercase">
              Espacios de Trabajo
            </h3>
            <p className="text-xs text-content-dim mt-1 leading-relaxed">
              Administra tus carpetas de desarrollo y el proyecto activo en Merlin Code.
            </p>
          </div>

          {onOpenFolder && (
            <button
              type="button"
              onClick={() => {
                onCloseModal();
                onOpenFolder();
              }}
              className="inline-flex items-center gap-2 px-4 py-2 bg-accent-primary hover:bg-accent-primary-hover text-[#0b0e10] font-semibold text-sm rounded-lg transition-colors cursor-pointer shadow-sm shrink-0 self-start sm:self-auto"
            >
              <IconFolderOpen size={16} />
              <span>Abrir Carpeta...</span>
            </button>
          )}
        </div>

        {/* Lista limpia de proyectos con divisores horizontales */}
        <div className="divide-y divide-border-subtle/40 border-t border-b border-border-subtle/40">
          {projects.length === 0 ? (
            <div className="py-10 text-center text-sm font-mono text-content-dim">
              No hay carpetas vinculadas en el historial.
            </div>
          ) : (
            projects.map((proj) => {
              const isCurrent = proj.id === activeProjectId;
              return (
                <div
                  key={proj.id}
                  className={`py-3.5 px-3 flex items-center justify-between gap-4 transition-colors rounded-lg ${
                    isCurrent ? 'bg-[#152329]/60' : 'hover:bg-[#12181b]/60'
                  }`}
                >
                  <div className="flex items-center gap-3.5 min-w-0 flex-1">
                    <IconFolder
                      size={18}
                      className={isCurrent ? 'text-accent-primary shrink-0' : 'text-content-dim shrink-0'}
                    />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2.5">
                        <span className="text-sm font-mono font-medium text-content-headline">
                          {proj.name}
                        </span>
                        {isCurrent && (
                          <span className="text-xs font-mono px-2 py-0.5 rounded bg-[#162931] text-accent-primary border border-border-petrol/60">
                            Activo
                          </span>
                        )}
                      </div>
                      {proj.path && (
                        <span
                          className="text-xs font-mono text-content-dim block truncate mt-0.5"
                          title={proj.path}
                        >
                          {proj.path}
                        </span>
                      )}
                    </div>
                  </div>

                  <div className="flex items-center gap-2 shrink-0">
                    {!isCurrent && (
                      <button
                        type="button"
                        className="text-sm text-accent-primary hover:text-accent-primary-hover font-medium px-2.5 py-1 rounded hover:bg-[#152329] transition-colors cursor-pointer"
                        onClick={() => onSelectProject(proj.id)}
                      >
                        Activar
                      </button>
                    )}
                    {onDeleteProject && (
                      <button
                        type="button"
                        className="p-1.5 text-content-dim hover:text-status-error hover:bg-[#25181a] rounded transition-colors cursor-pointer"
                        onClick={() => onDeleteProject(proj.id)}
                        title="Quitar proyecto del historial"
                      >
                        <IconTrash size={15} />
                      </button>
                    )}
                  </div>
                </div>
              );
            })
          )}
        </div>

        {/* Input para añadir manual si la función está habilitada */}
        {onAddProject && (
          <form onSubmit={handleCreate} className="flex gap-3 pt-5">
            <input
              type="text"
              value={newProjectName}
              onChange={(e) => setNewProjectName(e.target.value)}
              placeholder="Nombre para un nuevo espacio..."
              className="flex-1 bg-[#0d1214] border border-border-subtle hover:border-border-petrol focus:border-accent-primary rounded-lg px-3.5 py-2 text-sm text-content-headline font-mono outline-none transition-colors"
            />
            <button
              type="submit"
              disabled={!newProjectName.trim()}
              className="flex items-center gap-2 px-4 py-2 bg-[#162329] hover:bg-[#1e323b] disabled:opacity-40 text-accent-primary border border-border-petrol text-sm font-mono font-medium rounded-lg transition-colors cursor-pointer"
            >
              <IconPlus size={14} />
              <span>Añadir</span>
            </button>
          </form>
        )}
      </div>
    </div>
  );
};
