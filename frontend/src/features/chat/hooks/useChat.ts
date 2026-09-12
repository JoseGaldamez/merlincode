import { useState, useCallback } from 'react';
import { Message, AgentTelemetry, Project } from '../../../types';
import { Greet } from '../../../../wailsjs/go/main/App';

interface UseChatOptions {
  activeProject?: Project | null;
  activeSessionId?: string;
  onEnsureSession?: (userText: string) => void;
  onIncrementSessionCount?: (sessionId: string) => void;
}

export function useChat({
  activeProject,
  activeSessionId,
  onEnsureSession,
  onIncrementSessionCount,
}: UseChatOptions = {}) {
  const [messages, setMessages] = useState<Message[]>([]);
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

  const clearMessages = useCallback(() => {
    setMessages([]);
  }, []);

  const sendMessage = useCallback(
    async (text: string) => {
      const startTime = performance.now();
      const userMsg: Message = {
        id: `usr-${Date.now()}`,
        role: 'user',
        content: text,
        timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      };

      // Garantizar que exista una sesión activa
      if (onEnsureSession) {
        onEnsureSession(text);
      } else if (activeSessionId && onIncrementSessionCount) {
        onIncrementSessionCount(activeSessionId);
      }

      setMessages((prev) => [...prev, userMsg]);

      // Si no hay carpeta/proyecto activo, mostrar mensaje instructivo con acción
      if (!activeProject || !activeProject.path) {
        const promptFolderMsg: Message = {
          id: `asst-${Date.now()}`,
          role: 'assistant',
          content: `Para poder trabajar y procesar tus instrucciones con seguridad, primero debes seleccionar una carpeta de proyecto. Merlin solo modificará los archivos que se encuentren dentro del directorio que elijas.`,
          timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          action: {
            type: 'open_folder',
            label: '📁 Seleccionar carpeta de trabajo',
          },
        };
        setMessages((prev) => [...prev, promptFolderMsg]);
        return;
      }

      setIsLoading(true);
      setTelemetry((prev) => ({
        ...prev,
        status: 'synthesizing',
        tokensPrompt: prev.tokensPrompt + Math.max(1, Math.floor(text.length / 4)),
      }));

      try {
        const res = await Greet(text);
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
      } catch (err: any) {
        const elapsed = Math.round(performance.now() - startTime);
        const errMsg: Message = {
          id: `err-${Date.now()}`,
          role: 'assistant',
          content: `Error: ${String(err?.message || err || 'No se pudo comunicar con el backend.')}`,
          timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          status: 'error',
        };
        setMessages((prev) => [...prev, errMsg]);
        setTelemetry((prev) => ({
          ...prev,
          status: 'error',
          latencyMs: elapsed,
        }));
      } finally {
        setIsLoading(false);
      }
    },
    [activeProject, activeSessionId, onEnsureSession, onIncrementSessionCount]
  );

  return {
    messages,
    setMessages,
    clearMessages,
    isLoading,
    telemetry,
    setTelemetry,
    sendMessage,
  };
}
