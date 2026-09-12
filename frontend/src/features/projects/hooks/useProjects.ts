import { useState, useEffect, useCallback } from 'react';
import { Project } from '../../../types';
import {
  SelectProjectFolder,
  SetActiveProject,
} from '../../../../wailsjs/go/main/App';

const STORAGE_KEY_PROJECTS = 'merlin_projects_list';
const STORAGE_KEY_ACTIVE_PROJECT = 'merlin_active_project';

function getInitialProjects(): Project[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY_PROJECTS);
    if (raw) {
      const parsed = JSON.parse(raw);
      if (Array.isArray(parsed) && parsed.length > 0) return parsed;
    }
  } catch {}
  return [];
}

function getInitialActiveProject(projs: Project[]): string {
  try {
    const saved = localStorage.getItem(STORAGE_KEY_ACTIVE_PROJECT);
    if (saved && projs.some((p) => p.id === saved)) {
      return saved;
    }
  } catch {}
  return '';
}

export function useProjects() {
  const [projects, setProjects] = useState<Project[]>(() => getInitialProjects());
  const [activeProjectId, setActiveProjectId] = useState<string>(() =>
    getInitialActiveProject(getInitialProjects())
  );

  const activeProject = projects.find((p) => p.id === activeProjectId) || null;

  // Sincronizar y validar la carpeta del proyecto activo con el backend de Go al iniciar
  useEffect(() => {
    const savedActive = projects.find((p) => p.id === activeProjectId);
    if (savedActive && savedActive.path) {
      SetActiveProject(savedActive.path)
        .then((proj) => {
          if (!proj) {
            setActiveProjectId('');
            try {
              localStorage.removeItem(STORAGE_KEY_ACTIVE_PROJECT);
            } catch {}
          }
        })
        .catch(() => {
          setActiveProjectId('');
          try {
            localStorage.removeItem(STORAGE_KEY_ACTIVE_PROJECT);
          } catch {}
        });
    }
  }, []);

  const openFolderDialog = useCallback(async () => {
    try {
      const proj = await SelectProjectFolder();
      if (!proj || !proj.path) {
        // Cancelado por el usuario
        return null;
      }

      const newProj: Project = {
        id: proj.id || proj.path,
        name: proj.name,
        path: proj.path,
      };

      setProjects((prev) => {
        const filtered = prev.filter((p) => p.path !== newProj.path);
        const next = [newProj, ...filtered];
        try {
          localStorage.setItem(STORAGE_KEY_PROJECTS, JSON.stringify(next));
        } catch {}
        return next;
      });

      setActiveProjectId(newProj.id);
      try {
        localStorage.setItem(STORAGE_KEY_ACTIVE_PROJECT, newProj.id);
      } catch {}
      return newProj;
    } catch (err: any) {
      console.error('Error al abrir la carpeta:', err);
      return null;
    }
  }, []);

  const selectProject = useCallback(
    async (id: string) => {
      if (id === '__open_folder__' || id === '__new__') {
        await openFolderDialog();
        return;
      }

      const selected = projects.find((p) => p.id === id);
      if (selected && selected.path) {
        try {
          await SetActiveProject(selected.path);
          setActiveProjectId(id);
          try {
            localStorage.setItem(STORAGE_KEY_ACTIVE_PROJECT, id);
          } catch {}
        } catch (err: any) {
          alert(`No se pudo acceder a la carpeta (${selected.path}): ${err?.message || err}`);
        }
      } else {
        setActiveProjectId(id);
        try {
          localStorage.setItem(STORAGE_KEY_ACTIVE_PROJECT, id);
        } catch {}
      }
    },
    [projects, openFolderDialog]
  );

  const deleteProject = useCallback(
    (id: string) => {
      const updated = projects.filter((p) => p.id !== id);
      setProjects(updated);
      try {
        localStorage.setItem(STORAGE_KEY_PROJECTS, JSON.stringify(updated));
      } catch {}

      if (activeProjectId === id) {
        const nextActive = updated[0]?.id || '';
        setActiveProjectId(nextActive);
        try {
          localStorage.setItem(STORAGE_KEY_ACTIVE_PROJECT, nextActive);
        } catch {}
        if (nextActive) {
          const nextProj = updated[0];
          if (nextProj?.path) {
            SetActiveProject(nextProj.path).catch(() => {});
          }
        } else {
          SetActiveProject('').catch(() => {});
        }
      }
    },
    [projects, activeProjectId]
  );

  const addProject = useCallback((name: string) => {
    const newProj: Project = { id: `p-${Date.now()}`, name, path: '' };
    setProjects((prev) => {
      const updated = [...prev, newProj];
      try {
        localStorage.setItem(STORAGE_KEY_PROJECTS, JSON.stringify(updated));
        localStorage.setItem(STORAGE_KEY_ACTIVE_PROJECT, newProj.id);
      } catch {}
      return updated;
    });
    setActiveProjectId(newProj.id);
  }, []);

  return {
    projects,
    activeProjectId,
    activeProject,
    openFolderDialog,
    selectProject,
    deleteProject,
    addProject,
  };
}
