import { useState, useCallback, useEffect } from 'react';
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
  const [sessions, setSessions] = useState<Session[]>([]);
  const [activeSessionId, setActiveSessionId] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState<string>('');

  const activeSession = sessions.find((s) => s.id === activeSessionId) || null;

  // Cargar lista de sesiones desde la base de datos local SQLite
  const refreshSessions = useCallback(async () => {
    try {
      const records = await ListSessions();
      if (Array.isArray(records)) {
        const mapped: Session[] = records.map((s) => ({
          id: s.id,
          title: s.title || 'Nueva sesión',
          date: formatSessionDate(s.updatedAt || s.createdAt),
          messagesCount: s.messagesCount || 0,
          projectId: s.projectId,
          projectPath: s.projectPath,
        }));
        setSessions(mapped);
        setActiveSessionId((current) => {
          if (current && mapped.some((s) => s.id === current)) {
            return current;
          }
          return mapped.length > 0 ? mapped[0].id : '';
        });
      }
    } catch (err) {
      console.error('[Merlin Sessions] Error listando sesiones de SQLite:', err);
    }
  }, []);

  useEffect(() => {
    refreshSessions();
  }, [refreshSessions]);

  const createSession = useCallback(
    async (initialTitle?: string) => {
      const newId = `s-${Date.now()}`;
      const title = initialTitle || `Sesión ${sessions.length + 1}`;

      try {
        await CreateSession(
          newId,
          title,
          activeProject?.id || '',
          activeProject?.path || ''
        );
      } catch (err) {
        console.error('[Merlin Sessions] Error creando sesión en SQLite:', err);
      }

      const newSession: Session = {
        id: newId,
        title,
        date: new Date().toLocaleDateString([], { month: 'short', day: 'numeric' }),
        messagesCount: 0,
        projectId: activeProject?.id,
        projectPath: activeProject?.path,
      };

      setSessions((prev) => [newSession, ...prev]);
      setActiveSessionId(newId);
      return newId;
    },
    [sessions.length, activeProject]
  );

  const selectSession = useCallback((id: string) => {
    setActiveSessionId(id);
  }, []);

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
    (userText: string): string => {
      let targetId = activeSessionId;
      const title = userText.length > 25 ? `${userText.substring(0, 25)}...` : userText;

      if (!targetId) {
        targetId = `s-${Date.now()}`;

        CreateSession(
          targetId,
          title,
          activeProject?.id || '',
          activeProject?.path || ''
        ).catch((err) => {
          console.error('[Merlin Sessions] Error creando sesión en SQLite (ensureSession):', err);
        });

        const newSession: Session = {
          id: targetId,
          title,
          date: new Date().toLocaleDateString([], { month: 'short', day: 'numeric' }),
          messagesCount: 1,
          projectId: activeProject?.id,
          projectPath: activeProject?.path,
        };
        setSessions((prev) => [newSession, ...prev]);
        setActiveSessionId(targetId);
      } else {
        setSessions((prev) =>
          prev.map((s) => {
            if (s.id === targetId) {
              const shouldUpdateTitle =
                s.messagesCount === 0 || /^Sesión \d+$/i.test(s.title) || s.title === 'Nueva sesión';
              const newTitle = shouldUpdateTitle ? title : s.title;
              if (shouldUpdateTitle) {
                UpdateSessionTitle(targetId, newTitle).catch(() => {});
              }
              return { ...s, title: newTitle, messagesCount: s.messagesCount + 1 };
            }
            return s;
          })
        );
      }
      return targetId;
    },
    [activeSessionId, activeProject]
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
