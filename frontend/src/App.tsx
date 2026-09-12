import React, { useState, useEffect } from 'react';
import './App.css';
import { Header } from './components/Header';
import { LeftSidebar } from './components/LeftSidebar';
import { RightSidebar } from './components/RightSidebar';
import { ChatArea } from './components/ChatArea';
import { SettingsModal } from './components/SettingsModal';
import { Message, Session, AgentTelemetry, Artifact, Project } from './types';
import { Greet, SaveWindowSize, SavePanelsState, GetPanelsState } from '../wailsjs/go/main/App';

const STORAGE_KEY_LEFT = 'merlin_left_panel_open';
const STORAGE_KEY_RIGHT = 'merlin_right_panel_open';
const STORAGE_KEY_PROJECTS = 'merlin_projects_list';
const STORAGE_KEY_ACTIVE_PROJECT = 'merlin_active_project';

const DEFAULT_PROJECTS: Project[] = [
  { id: 'merlincode', name: 'merlincode' },
];

function getInitialPanelState(key: string, defaultVal: boolean): boolean {
  try {
    const saved = localStorage.getItem(key);
    if (saved !== null) {
      return saved === 'true';
    }
  } catch {
    // localStorage no disponible o bloqueado
  }
  return defaultVal;
}

function getInitialProjects(): Project[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY_PROJECTS);
    if (raw) {
      const parsed = JSON.parse(raw);
      if (Array.isArray(parsed) && parsed.length > 0) return parsed;
    }
  } catch {}
  return DEFAULT_PROJECTS;
}

function getInitialActiveProject(projs: Project[]): string {
  try {
    const saved = localStorage.getItem(STORAGE_KEY_ACTIVE_PROJECT);
    if (saved && projs.some((p) => p.id === saved)) {
      return saved;
    }
  } catch {}
  return projs[0]?.id || 'merlincode';
}

export function App() {
  // Proyectos y búsqueda en sesiones
  const [projects, setProjects] = useState<Project[]>(() => getInitialProjects());
  const [activeProjectId, setActiveProjectId] = useState<string>(() =>
    getInitialActiveProject(getInitialProjects())
  );
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);

  // El panel derecho debe estar cerrado por defecto (false) y el izquierdo abierto (true)
  const [leftOpen, setLeftOpen] = useState<boolean>(() => getInitialPanelState(STORAGE_KEY_LEFT, true));
  const [rightOpen, setRightOpen] = useState<boolean>(() => getInitialPanelState(STORAGE_KEY_RIGHT, false));
  const [sessions, setSessions] = useState<Session[]>([]);
  const [activeSessionId, setActiveSessionId] = useState<string>('');
  const [messages, setMessages] = useState<Message[]>([]);
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [selectedArtifactId, setSelectedArtifactId] = useState<string | undefined>();
  const [isLoading, setIsLoading] = useState(false);

  const [telemetry, setTelemetry] = useState<AgentTelemetry>({
    status: 'idle',
    activeModel: 'Merlin',
    tokensPrompt: 0,
    tokensCompletion: 0,
    latencyMs: 0,
    activeFile: '',
    temperature: 0.7,
  });

  // Sincronizar estado inicial guardado en el backend de Go si no existe en localStorage
  useEffect(() => {
    GetPanelsState()
      .then((state) => {
        if (state) {
          if (localStorage.getItem(STORAGE_KEY_LEFT) === null && typeof state.leftOpen === 'boolean') {
            setLeftOpen(state.leftOpen);
          }
          if (localStorage.getItem(STORAGE_KEY_RIGHT) === null && typeof state.rightOpen === 'boolean') {
            setRightOpen(state.rightOpen);
          }
        }
      })
      .catch(() => {
        // Modo navegador fuera de Wails
      });
  }, []);

  // Escuchar y guardar automáticamente las dimensiones de la ventana al redimensionar
  useEffect(() => {
    let timeoutId: number;
    const handleResize = () => {
      window.clearTimeout(timeoutId);
      timeoutId = window.setTimeout(() => {
        if (typeof window !== 'undefined' && window.outerWidth && window.outerHeight) {
          const isMax = window.screen && (window.outerWidth >= window.screen.availWidth && window.outerHeight >= window.screen.availHeight);
          SaveWindowSize(window.outerWidth, window.outerHeight, !!isMax).catch(() => {
            // Silencioso en modo navegador fuera de Wails
          });
        }
      }, 400);
    };

    window.addEventListener('resize', handleResize);
    return () => {
      window.removeEventListener('resize', handleResize);
      window.clearTimeout(timeoutId);
    };
  }, []);

  const handleToggleLeft = () => {
    const next = !leftOpen;
    setLeftOpen(next);
    try {
      localStorage.setItem(STORAGE_KEY_LEFT, String(next));
    } catch {}
    SavePanelsState(next, rightOpen).catch(() => {});
  };

  const handleToggleRight = () => {
    const next = !rightOpen;
    setRightOpen(next);
    try {
      localStorage.setItem(STORAGE_KEY_RIGHT, String(next));
    } catch {}
    SavePanelsState(leftOpen, next).catch(() => {});
  };

  const handleSelectProject = (id: string) => {
    if (id === '__new__') {
      const name = window.prompt('Nombre del nuevo proyecto:');
      if (name && name.trim()) {
        const cleanName = name.trim();
        const newProject: Project = {
          id: `p-${Date.now()}`,
          name: cleanName,
        };
        const updated = [...projects, newProject];
        setProjects(updated);
        setActiveProjectId(newProject.id);
        try {
          localStorage.setItem(STORAGE_KEY_PROJECTS, JSON.stringify(updated));
          localStorage.setItem(STORAGE_KEY_ACTIVE_PROJECT, newProject.id);
        } catch {}
      }
      return;
    }

    setActiveProjectId(id);
    try {
      localStorage.setItem(STORAGE_KEY_ACTIVE_PROJECT, id);
    } catch {}
  };

  const handleDeleteProject = (id: string) => {
    if (projects.length <= 1) return;
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
    }
  };

  const handleNewSession = () => {
    const newId = `s-${Date.now()}`;
    const newSession: Session = {
      id: newId,
      title: `Sesión ${sessions.length + 1}`,
      date: new Date().toLocaleDateString([], { month: 'short', day: 'numeric' }),
      messagesCount: 0,
    };
    setSessions((prev) => [newSession, ...prev]);
    setActiveSessionId(newId);
    setMessages([]);
  };

  const handleSelectSession = (id: string) => {
    setActiveSessionId(id);
  };

  const handleDeleteSession = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    const updated = sessions.filter((s) => s.id !== id);
    setSessions(updated);
    if (activeSessionId === id) {
      setActiveSessionId(updated.length > 0 ? updated[0].id : '');
      setMessages([]);
    }
  };

  const handleSendMessage = (text: string) => {
    const startTime = performance.now();
    const userMsg: Message = {
      id: `usr-${Date.now()}`,
      role: 'user',
      content: text,
      timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    };

    // Si no hay sesión activa, creamos una automáticamente
    if (!activeSessionId) {
      const newId = `s-${Date.now()}`;
      const newSession: Session = {
        id: newId,
        title: text.length > 25 ? text.substring(0, 25) + '...' : text,
        date: new Date().toLocaleDateString([], { month: 'short', day: 'numeric' }),
        messagesCount: 1,
      };
      setSessions((prev) => [newSession, ...prev]);
      setActiveSessionId(newId);
    } else {
      setSessions((prev) =>
        prev.map((s) =>
          s.id === activeSessionId
            ? { ...s, messagesCount: s.messagesCount + 1 }
            : s
        )
      );
    }

    setMessages((prev) => [...prev, userMsg]);
    setIsLoading(true);
    setTelemetry((prev) => ({
      ...prev,
      status: 'synthesizing',
      tokensPrompt: prev.tokensPrompt + Math.max(1, Math.floor(text.length / 4)),
    }));

    // Llamada real al backend de Go
    Greet(text)
      .then((res: string) => {
        const elapsed = Math.round(performance.now() - startTime);
        const assistantMsg: Message = {
          id: `asst-${Date.now()}`,
          role: 'assistant',
          content: res,
          timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
        };

        setMessages((prev) => [...prev, assistantMsg]);
        setTelemetry((prev) => ({
          ...prev,
          status: 'idle',
          latencyMs: elapsed,
          tokensCompletion: prev.tokensCompletion + Math.max(1, Math.floor(res.length / 4)),
        }));
      })
      .catch((err: any) => {
        const elapsed = Math.round(performance.now() - startTime);
        const errMsg: Message = {
          id: `err-${Date.now()}`,
          role: 'assistant',
          content: `Error: ${String(err || 'No se pudo comunicar con el backend.')}`,
          timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          status: 'error',
        };
        setMessages((prev) => [...prev, errMsg]);
        setTelemetry((prev) => ({
          ...prev,
          status: 'error',
          latencyMs: elapsed,
        }));
      })
      .finally(() => {
        setIsLoading(false);
      });
  };

  return (
    <div className="h-screen w-screen flex flex-col bg-canvas text-content-body overflow-hidden select-none">
      <Header
        projects={projects}
        activeProjectId={activeProjectId}
        onSelectProject={handleSelectProject}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        onOpenSettings={() => setIsSettingsOpen(true)}
        telemetry={telemetry}
      />

      <div className="flex-1 flex overflow-hidden min-h-0 w-full">
        <LeftSidebar
          isOpen={leftOpen}
          onToggle={handleToggleLeft}
          sessions={sessions}
          activeSessionId={activeSessionId}
          onSelectSession={handleSelectSession}
          onNewSession={handleNewSession}
          onDeleteSession={handleDeleteSession}
          searchQuery={searchQuery}
          projectName={projects.find((p) => p.id === activeProjectId)?.name || 'merlincode'}
        />

        <ChatArea
          messages={messages}
          onSendMessage={handleSendMessage}
          isLoading={isLoading}
        />

        <RightSidebar
          isOpen={rightOpen}
          onToggle={handleToggleRight}
          artifacts={artifacts}
          selectedArtifactId={selectedArtifactId}
          onSelectArtifact={(id) => setSelectedArtifactId(id)}
        />
      </div>

      <SettingsModal
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
        projects={projects}
        activeProjectId={activeProjectId}
        onSelectProject={handleSelectProject}
        onDeleteProject={handleDeleteProject}
        onAddProject={(name) => {
          const newProj = { id: `p-${Date.now()}`, name };
          const updated = [...projects, newProj];
          setProjects(updated);
          setActiveProjectId(newProj.id);
          try {
            localStorage.setItem(STORAGE_KEY_PROJECTS, JSON.stringify(updated));
            localStorage.setItem(STORAGE_KEY_ACTIVE_PROJECT, newProj.id);
          } catch {}
        }}
        telemetry={telemetry}
        onUpdateTelemetry={(updated) => {
          setTelemetry((prev) => ({ ...prev, ...updated }));
        }}
      />
    </div>
  );
}

export default App;
