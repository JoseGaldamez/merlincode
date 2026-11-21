import { useState, useCallback, useEffect, useRef } from 'react';
import { Session, Project } from '../../../types';
import {
  ListSessions,
  CreateSession,
  DeleteSession,
  UpdateSessionTitle,
} from '../../../../wailsjs/go/app/App';

interface UseSessionsOptions {
  activeProject?: Project | null;
}

function formatSessionDate(raw: any): string {
  if (!raw) return new Date().toLocaleDateString([], { month: 'short', day: 'numeric' });
  const d = new Date(raw);
  if (isNaN(d.getTime())) return new Date().toLocaleDateString([], { month: 'short', day: 'numeric' });
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' });
}

export function useSessions({ activeProject }: UseSessionsOptions = {}) {
  const [storedSessions, setSessions] = useState<Session[]>([]);
  const [selectedSessionId, setActiveSessionId] = useState<string>('');
  const projectId = activeProject?.id || '';
  const projectPath = activeProject?.path || '';
  const projectKey = JSON.stringify([projectId, projectPath]);
  const currentProjectRef = useRef(projectKey);
  const refreshRequestRef = useRef<symbol>();
  currentProjectRef.current = projectKey;

  const belongsToProject = useCallback((session: Pick<Session, 'projectId' | 'projectPath'>) => {
    if (!projectId) return false;
    if (session.projectId) {
      return session.projectId === projectId && (!session.projectPath || session.projectPath === projectPath);
    }
    return Boolean(projectPath && session.projectPath === projectPath);
  }, [projectId, projectPath]);
  const sessions = storedSessions.filter(belongsToProject);
  const activeSessionId = sessions.some((session) => session.id === selectedSessionId) ? selectedSessionId : '';
  const [searchQuery, setSearchQuery] = useState<string>('');

  const activeSession = sessions.find((s) => s.id === activeSessionId) || null;

  // Cargar lista de sesiones desde la base de datos local SQLite
  const refreshSessions = useCallback(async () => {
    if (currentProjectRef.current !== projectKey || !projectId) return;
    const request = Symbol();
    refreshRequestRef.current = request;
    try {
      const records = await ListSessions();
      if (currentProjectRef.current !== projectKey || refreshRequestRef.current !== request) return;
      if (Array.isArray(records)) {
        const mapped: Session[] = records.filter(belongsToProject).map((s) => ({
          id: s.id,
          title: s.title || 'Nueva sesión',
          date: formatSessionDate(s.updatedAt || s.createdAt),
          messagesCount: s.messagesCount || 0,
          projectId: s.projectId,
          projectPath: s.projectPath,
        }));
        setSessions(mapped);
        setActiveSessionId((current) => mapped.some((session) => session.id === current) ? current : '');
      }
    } catch (err) {
      console.error('[Merlin Sessions] Error listando sesiones de SQLite:', err);
    } finally {
      if (refreshRequestRef.current === request) refreshRequestRef.current = undefined;
    }
  }, [projectKey, projectId, belongsToProject]);

  useEffect(() => {
    setSessions([]);
    setActiveSessionId('');
    setSearchQuery('');
    void refreshSessions();
    return () => { refreshRequestRef.current = undefined; };
  }, [refreshSessions]);

  const createSession = useCallback(
    async (initialTitle?: string, shouldSelect?: () => boolean) => {
      if (!projectId || currentProjectRef.current !== projectKey) return '';
      const newId = `s-${Date.now()}`;
      const title = initialTitle || `Sesión ${sessions.length + 1}`;

      await CreateSession(newId, title, projectId, projectPath);

      if (currentProjectRef.current !== projectKey) return newId;

      const newSession: Session = {
        id: newId,
        title,
        date: new Date().toLocaleDateString([], { month: 'short', day: 'numeric' }),
        messagesCount: 0,
        projectId,
        projectPath,
      };

      setSessions((prev) => [newSession, ...prev.filter((session) => session.id !== newId)]);
      if (shouldSelect?.() !== false) setActiveSessionId(newId);
      // Una consulta iniciada antes de persistir la sesión puede devolver un historial anterior.
      if (refreshRequestRef.current) void refreshSessions();
      return newId;
    },
    [sessions.length, projectId, projectPath, projectKey, refreshSessions]
  );

  const selectSession = useCallback((id: string) => {
    if (currentProjectRef.current === projectKey && sessions.some((session) => session.id === id)) {
      setActiveSessionId(id);
    }
  }, [sessions, projectKey]);

  const deleteSession = useCallback((id: string, e?: React.MouseEvent) => {
    if (e) e.stopPropagation();

    DeleteSession(id).catch((err) => {
      console.error('[Merlin Sessions] Error eliminando sesión en SQLite:', err);
    });

    setSessions((prev) => prev.filter((s) => s.id !== id));

    setActiveSessionId((currentActive) => {
      if (currentActive === id) {
        return '';
      }
      return currentActive;
    });
  }, []);

  const ensureSession = useCallback(
    async (userText: string, shouldSelect: () => boolean): Promise<string> => {
      if (!projectId || currentProjectRef.current !== projectKey) return '';
      const title = userText.length > 25 ? `${userText.substring(0, 25)}...` : userText;
      if (!activeSessionId) return createSession(title, shouldSelect);

      setSessions((prev) =>
        prev.map((s) => {
          if (s.id !== activeSessionId) return s;
          const shouldUpdateTitle =
            s.messagesCount === 0 || /^Sesión \d+$/i.test(s.title) || s.title === 'Nueva sesión';
          const newTitle = shouldUpdateTitle ? title : s.title;
          if (shouldUpdateTitle) {
            UpdateSessionTitle(activeSessionId, newTitle).catch(() => {});
          }
          return { ...s, title: newTitle, messagesCount: s.messagesCount + 1 };
        })
      );
      return activeSessionId;
    },
    [activeSessionId, createSession, projectId, projectKey]
  );

  const incrementSessionCount = useCallback((sessionId: string) => {
    setSessions((prev) =>
      prev.map((s) =>
        s.id === sessionId ? { ...s, messagesCount: s.messagesCount + 1 } : s
      )
    );
  }, []);

  const updateSessionTitle = useCallback(async (id: string, newTitle: string) => {
    try {
      await UpdateSessionTitle(id, newTitle);
      setSessions((prev) =>
        prev.map((s) => (s.id === id ? { ...s, title: newTitle } : s))
      );
    } catch (err) {
      console.error('[Merlin Sessions] Error actualizando título en SQLite:', err);
    }
  }, []);

  return {
    sessions,
    setSessions,
    activeSessionId,
    setActiveSessionId,
    activeSession,
    searchQuery,
    setSearchQuery,
    createSession,
    selectSession,
    deleteSession,
    ensureSession,
    incrementSessionCount,
    updateSessionTitle,
    refreshSessions,
  };
}
