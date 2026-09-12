import React, { useState } from 'react';
import './App.css';
import { Header, usePanelsState } from './features/layout';
import { LeftSidebar, useSessions } from './features/sessions';
import { RightSidebar, useArtifacts } from './features/artifacts';
import { ChatArea, useChat } from './features/chat';
import { SettingsModal } from './features/settings';
import { useProjects } from './features/projects';

export function App() {
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);

  // Proyectos y espacios de trabajo
  const {
    projects,
    activeProjectId,
    activeProject,
    openFolderDialog,
    selectProject,
    deleteProject,
    addProject,
  } = useProjects();

  // Historial de sesiones y búsqueda
  const {
    sessions,
    activeSessionId,
    searchQuery,
    setSearchQuery,
    createSession,
    selectSession,
    deleteSession,
    ensureSession,
    incrementSessionCount,
  } = useSessions();

  // Gestión de mensajes de chat y telemetría de IA
  const {
    messages,
    clearMessages,
    sendMessage,
    isLoading,
    telemetry,
    setTelemetry,
  } = useChat({
    activeProject,
    activeSessionId,
    onEnsureSession: ensureSession,
    onIncrementSessionCount: incrementSessionCount,
  });

  // Artefactos generados
  const {
    artifacts,
    selectedArtifactId,
    setSelectedArtifactId,
  } = useArtifacts();

  // Estado y dimensiones de paneles laterales y ventana
  const {
    leftOpen,
    rightOpen,
    leftWidth,
    rightWidth,
    toggleLeft,
    toggleRight,
    setLeftWidth,
    setRightWidth,
  } = usePanelsState();

  const handleNewSession = () => {
    createSession();
    clearMessages();
  };

  const handleDeleteSession = (id: string, e: React.MouseEvent) => {
    deleteSession(id, e);
    if (activeSessionId === id) {
      clearMessages();
    }
  };

  return (
    <div className="h-screen w-screen flex flex-col bg-canvas text-content-body overflow-hidden select-none">
      <Header
        projects={projects}
        activeProjectId={activeProjectId}
        onSelectProject={selectProject}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        onOpenSettings={() => setIsSettingsOpen(true)}
        telemetry={telemetry}
      />

      <div className="flex-1 flex overflow-hidden min-h-0 w-full">
        <LeftSidebar
          isOpen={leftOpen}
          onToggle={toggleLeft}
          width={leftWidth}
          onWidthChange={setLeftWidth}
          sessions={sessions}
          activeSessionId={activeSessionId}
          onSelectSession={selectSession}
          onNewSession={handleNewSession}
          onDeleteSession={handleDeleteSession}
          searchQuery={searchQuery}
          projectName={activeProject?.name || ''}
          projectPath={activeProject?.path}
          onOpenFolder={openFolderDialog}
        />

        <ChatArea
          messages={messages}
          onSendMessage={sendMessage}
          isLoading={isLoading}
          activeProjectName={activeProject?.name}
          onOpenFolder={openFolderDialog}
        />

        <RightSidebar
          isOpen={rightOpen}
          onToggle={toggleRight}
          width={rightWidth}
          onWidthChange={setRightWidth}
          artifacts={artifacts}
          selectedArtifactId={selectedArtifactId}
          onSelectArtifact={setSelectedArtifactId}
        />
      </div>

      <SettingsModal
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
        projects={projects}
        activeProjectId={activeProjectId}
        onSelectProject={selectProject}
        onDeleteProject={deleteProject}
        onOpenFolder={openFolderDialog}
        onAddProject={addProject}
        telemetry={telemetry}
        onUpdateTelemetry={(updated) => {
          setTelemetry((prev) => ({ ...prev, ...updated }));
        }}
      />
    </div>
  );
}

export default App;
