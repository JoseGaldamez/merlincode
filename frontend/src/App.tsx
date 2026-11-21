import React, { useState, useEffect, useCallback } from 'react';
import './App.css';
import { Header, usePanelsState } from './features/layout';
import { LeftSidebar, useSessions } from './features/sessions';
import { RightSidebar, useArtifacts } from './features/artifacts';
import { ChatArea, useChat } from './features/chat';
import { SettingsModal, useSettingsForm, useAIProviderStatus } from './features/settings';
import { useProjects } from './features/projects';
import { FilePreviewModal } from './features/projects/components/FilePreviewModal';
import { ExecutionProvider } from './features/executions/ExecutionProvider';
import { ExecutionFlight } from './features/executions/ExecutionFlight';
import type { ExecutionFocus } from './features/executions/types';
import './features/executions/executions.css';

export function App() {
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [executionFocus, setExecutionFocus] = useState<ExecutionFocus>();
  const [filePreview, setFilePreview] = useState<{ project: string; path: string }>();

  // Estado global de configuración de IA y estado de credenciales en el llavero
  const { statuses, refreshStatuses } = useAIProviderStatus();
  const { settings, selectProvider, saveSettings } = useSettingsForm(statuses);
  const closeSettings = useCallback(() => {
    setIsSettingsOpen(false);
    void refreshStatuses();
  }, [refreshStatuses]);

  // Proyectos y espacios de trabajo
  const {
    projects,
    activeProjectId,
    activeProject,
    openFolderDialog,
    selectProject,
    selectRunningProject,
    deleteProject,
    addProject,
  } = useProjects();

  const projectPath = activeProject?.path;
  const openFilePreview = useCallback((path: string) => {
    if (projectPath) setFilePreview({ project: projectPath, path });
  }, [projectPath]);

  const closeFilePreview = useCallback(() => setFilePreview(undefined), []);
  useEffect(() => {
    closeFilePreview();
  }, [projectPath, closeFilePreview]);

  // Historial de sesiones y búsqueda
  const {
    sessions,
    activeSessionId,
    setActiveSessionId,
    searchQuery,
    setSearchQuery,
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
    handleFeedback,
    resolveToolApproval,
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
    openRight,
    setLeftWidth,
    setRightWidth,
  } = usePanelsState();

  const revealExecution = useCallback((focus: ExecutionFocus) => {
    openRight();
    setExecutionFocus(focus);
  }, [openRight]);

  const showRunningProject = useCallback(async (path: string, focus: ExecutionFocus) => {
    await selectRunningProject(path);
    revealExecution(focus);
  }, [selectRunningProject, revealExecution]);

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
    <ExecutionProvider project={activeProject?.path || ''} onReveal={revealExecution} onRequestFix={sendMessage} chatBusy={isLoading}>
    <div className="h-screen w-screen flex flex-col bg-canvas text-content-body overflow-hidden select-none">
      <Header
        projects={projects}
        activeProjectId={activeProjectId}
        onSelectProject={selectProject}
        onSelectRunningProject={showRunningProject}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        onOpenSettings={() => setIsSettingsOpen(true)}
        telemetry={telemetry}
        statuses={statuses}
        selectedProviderId={settings.modelProvider}
        onSelectProvider={selectProvider}
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
          onPreviewFile={projectPath ? openFilePreview : undefined}
        />

        <ChatArea
          messages={messages}
          onSendMessage={sendMessage}
          isLoading={isLoading}
          onCancelStream={cancelStream}
          streamingStatusText={streamingStatusText}
          activeProjectName={activeProject?.name}
          onOpenFolder={openFolderDialog}
          onPreviewFile={projectPath ? openFilePreview : undefined}
          onFeedback={handleFeedback}
          onResolveApproval={resolveToolApproval}
          activeModel={settings.model}
        />

        <RightSidebar
          isOpen={rightOpen}
          onToggle={toggleRight}
          width={rightWidth}
          onWidthChange={setRightWidth}
          artifacts={artifacts}
          selectedArtifactId={selectedArtifactId}
          onSelectArtifact={setSelectedArtifactId}
          executionFocus={executionFocus}
        />
      </div>

      {isSettingsOpen && (
        <SettingsModal
          onClose={closeSettings}
          projects={projects}
          activeProjectId={activeProjectId}
          onSelectProject={selectProject}
          onDeleteProject={deleteProject}
          onOpenFolder={openFolderDialog}
          onAddProject={addProject}
          initialSettings={settings}
          onSave={saveSettings}
        />
      )}
      {filePreview && filePreview.project === projectPath && (
        <FilePreviewModal
          key={filePreview.path}
          project={filePreview.project}
          path={filePreview.path}
          onClose={closeFilePreview}
        />
      )}
      <ExecutionFlight focus={executionFocus} />
    </div>
    </ExecutionProvider>
  );
}

export default App;
