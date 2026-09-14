import React, { useState, useEffect } from 'react';
import './App.css';
import { Header, usePanelsState } from './features/layout';
import { LeftSidebar, useSessions } from './features/sessions';
import { RightSidebar, useArtifacts } from './features/artifacts';
import { ChatArea, useChat } from './features/chat';
import { SettingsModal, useSettingsForm, useAIProviderStatus } from './features/settings';
import { useProjects } from './features/projects';

export function App() {
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);

  // Estado global de configuración de IA y estado de credenciales en el llavero
  const { statuses, refreshStatuses } = useAIProviderStatus();
  const { settings, updateSetting } = useSettingsForm();

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
    setActiveSessionId,
    searchQuery,
    setSearchQuery,
    createSession,
    selectSession,
    deleteSession,
    ensureSession,
    incrementSessionCount,
    refreshSessions,
  } = useSessions({ activeProject });

  // Gestión de mensajes de chat y telemetría de IA con streaming en tiempo real
  const {
    messages,
    clearMessages,
    sendMessage,
    isLoading,
    streamingStatusText,
    cancelStream,
    telemetry,
    setTelemetry,
    handleFeedback,
  } = useChat({
    activeProject,
    activeSessionId,
    selectedProviderId: settings.modelProvider,
    selectedModelId: settings.model,
    onEnsureSession: ensureSession,
    onIncrementSessionCount: incrementSessionCount,
    onRefreshSessions: refreshSessions,
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

  // Refrescar estado de proveedores al cerrar el modal de ajustes
  useEffect(() => {
    if (!isSettingsOpen) {
      refreshStatuses();
    }
  }, [isSettingsOpen, refreshStatuses]);

  const handleNewSession = () => {
    if (isLoading) {
      cancelStream();
    }
    setActiveSessionId('');
    clearMessages();
  };

  const handleSelectSession = (id: string) => {
    if (isLoading && activeSessionId !== id) {
      cancelStream();
    }
    selectSession(id);
  };

  const handleDeleteSession = (id: string, e: React.MouseEvent) => {
    if (isLoading && activeSessionId === id) {
      cancelStream();
    }
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
        statuses={statuses}
        selectedProviderId={settings.modelProvider}
        onSelectProvider={(provId) => updateSetting('modelProvider', provId)}
      />

      <div className="flex-1 flex overflow-hidden min-h-0 w-full">
        <LeftSidebar
          isOpen={leftOpen}
          onToggle={toggleLeft}
          width={leftWidth}
          onWidthChange={setLeftWidth}
          sessions={sessions}
          activeSessionId={activeSessionId}
          onSelectSession={handleSelectSession}
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
          onCancelStream={cancelStream}
          streamingStatusText={streamingStatusText}
          activeProjectName={activeProject?.name}
          onOpenFolder={openFolderDialog}
          onFeedback={handleFeedback}
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
