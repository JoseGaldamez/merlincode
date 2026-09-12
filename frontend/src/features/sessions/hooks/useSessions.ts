import { useState, useCallback } from 'react';
import { Session } from '../../../types';

export function useSessions() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [activeSessionId, setActiveSessionId] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState<string>('');

  const activeSession = sessions.find((s) => s.id === activeSessionId) || null;

  const createSession = useCallback((initialTitle?: string) => {
    const newId = `s-${Date.now()}`;
    const newSession: Session = {
      id: newId,
      title: initialTitle || `Sesión ${sessions.length + 1}`,
      date: new Date().toLocaleDateString([], { month: 'short', day: 'numeric' }),
      messagesCount: 0,
    };
    setSessions((prev) => [newSession, ...prev]);
    setActiveSessionId(newId);
    return newId;
  }, [sessions.length]);

  const selectSession = useCallback((id: string) => {
    setActiveSessionId(id);
  }, []);

  const deleteSession = useCallback((id: string, e?: React.MouseEvent) => {
    if (e) e.stopPropagation();
    setSessions((prev) => {
      const updated = prev.filter((s) => s.id !== id);
      return updated;
    });

    setActiveSessionId((currentActive) => {
      if (currentActive === id) {
        return '';
      }
      return currentActive;
    });
  }, []);

  const ensureSession = useCallback((userText: string) => {
    setActiveSessionId((currentActive) => {
      if (!currentActive) {
        const newId = `s-${Date.now()}`;
        const newSession: Session = {
          id: newId,
          title: userText.length > 25 ? `${userText.substring(0, 25)}...` : userText,
          date: new Date().toLocaleDateString([], { month: 'short', day: 'numeric' }),
          messagesCount: 1,
        };
        setSessions((prev) => [newSession, ...prev]);
        return newId;
      } else {
        setSessions((prev) =>
          prev.map((s) =>
            s.id === currentActive ? { ...s, messagesCount: s.messagesCount + 1 } : s
          )
        );
        return currentActive;
      }
    });
  }, []);

  const incrementSessionCount = useCallback((sessionId: string) => {
    setSessions((prev) =>
      prev.map((s) =>
        s.id === sessionId ? { ...s, messagesCount: s.messagesCount + 1 } : s
      )
    );
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
  };
}
