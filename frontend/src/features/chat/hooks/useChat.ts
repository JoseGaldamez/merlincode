import { useState, useCallback, useEffect, useRef } from 'react';
import { Message, AgentTelemetry, Project, ChatStreamEvent } from '../../../types';
import {
  StartChatStream,
  CancelChatStream,
  GetSessionMessages,
  UpdateMessageFeedback,
} from '../../../../wailsjs/go/app/App';
import { domain } from '../../../../wailsjs/go/models';
import { EventsOn } from '../../../../wailsjs/runtime/runtime';

interface UseChatOptions {
  activeProject?: Project | null;
  activeSessionId?: string;
  selectedProviderId?: string;
  selectedModelId?: string;
  onEnsureSession?: (userText: string) => string;
  onIncrementSessionCount?: (sessionId: string) => void;
  onRefreshSessions?: () => void;
}

export function useChat({
  activeProject,
  activeSessionId,
  selectedProviderId = 'google',
  selectedModelId = '',
  onEnsureSession,
  onIncrementSessionCount,
  onRefreshSessions,
}: UseChatOptions = {}) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [streamingStatusText, setStreamingStatusText] = useState<string>('');
  const [telemetry, setTelemetry] = useState<AgentTelemetry>({
    status: 'idle',
    activeModel: selectedModelId || 'Merlin',
    tokensPrompt: 0,
    tokensCompletion: 0,
    latencyMs: 0,
    activeFile: '',
    temperature: 0.7,
  });

  const currentStreamingIdRef = useRef<string | null>(null);
  const currentStreamingSessionIdRef = useRef<string | null>(null);
  const skipNextLoadSessionIdRef = useRef<string | null>(null);
  const streamStartTimeRef = useRef<number>(0);

  const clearMessages = useCallback(() => {
    setMessages([]);
    currentStreamingIdRef.current = null;
    currentStreamingSessionIdRef.current = null;
    skipNextLoadSessionIdRef.current = null;
    setIsLoading(false);
    setStreamingStatusText('');
  }, []);

  // Cargar mensajes persistidos desde SQLite al cambiar de sesión activa
  useEffect(() => {
    if (!activeSessionId) {
      setMessages([]);
      return;
    }

    // Si la sesión activa coincide con la que acabamos de iniciar o estamos transmitiendo,
    // NO debemos recargar de la base de datos para no pisar el streaming en curso.
    if (
      currentStreamingIdRef.current ||
      currentStreamingSessionIdRef.current === activeSessionId ||
      skipNextLoadSessionIdRef.current === activeSessionId
    ) {
      skipNextLoadSessionIdRef.current = null;
      return;
    }

    let isSubscribed = true;
    GetSessionMessages(activeSessionId)
      .then((records) => {
        if (!isSubscribed || currentStreamingIdRef.current) return;
        if (Array.isArray(records) && records.length > 0) {
          const loaded: Message[] = records.map((rec) => ({
            id: rec.id,
            role: (rec.role as any) || 'user',
            content: rec.content || '',
            timestamp: rec.createdAt
              ? new Date(rec.createdAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
              : new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
            thoughtChain: rec.thoughtChain || '',
            status: (rec.status as any) || 'done',
            durationSeconds: rec.durationSeconds,
            tokensPrompt: rec.tokensPrompt,
            tokensCompletion: rec.tokensCompletion,
            feedback: (rec.feedback as 'like' | 'dislike') || null,
          }));
          setMessages(loaded);
        } else {
          if (!currentStreamingIdRef.current) {
            setMessages([]);
          }
        }
      })
      .catch((err) => {
        console.error('[Merlin Chat] Error cargando mensajes de sesión de SQLite:', err);
        if (isSubscribed && !currentStreamingIdRef.current) setMessages([]);
      });

    return () => {
      isSubscribed = false;
    };
  }, [activeSessionId]);

  // Escuchar eventos en streaming emitidos por el backend de Go
  useEffect(() => {
    const unsubscribe = EventsOn('chat:stream', (event: ChatStreamEvent) => {
      if (!event || !event.messageId) return;

      if (event.type === 'status') {
        if (event.statusText) {
          console.log(`[Merlin Chat] Estado: ${event.statusText}`);
          setStreamingStatusText(event.statusText);
        }
        return;
      }

      if (event.type === 'thinking') {
        setStreamingStatusText('Pensando...');
        setMessages((prev) =>
          prev.map((m) =>
            m.id === event.messageId
              ? {
                  ...m,
                  thoughtChain: (m.thoughtChain || '') + (event.thinking || ''),
                  status: 'thinking',
                }
              : m
          )
        );
        return;
      }

      if (event.type === 'content') {
        setStreamingStatusText('Generando respuesta...');
        setMessages((prev) =>
          prev.map((m) =>
            m.id === event.messageId
              ? {
                  ...m,
                  content: m.content + (event.content || ''),
                  status: 'synthesizing',
                }
              : m
          )
        );
        return;
      }

      if (event.type === 'done') {
        const elapsed = Math.round(performance.now() - streamStartTimeRef.current);
        const elapsedSec = Math.max(1, Math.round(elapsed / 1000));
        console.log(`[Merlin Chat] Stream finalizado en ${elapsed}ms:`, {
          model: event.modelId,
          tokensPrompt: event.tokensPrompt,
          tokensCompletion: event.tokensCompletion,
        });
        setMessages((prev) =>
          prev.map((m) =>
            m.id === event.messageId
              ? {
                  ...m,
                  status: 'done',
                  durationSeconds: elapsedSec,
                  tokensPrompt: event.tokensPrompt || m.tokensPrompt || 0,
                  tokensCompletion: event.tokensCompletion || m.tokensCompletion || 0,
                }
              : m
          )
        );
        setIsLoading(false);
        setStreamingStatusText('');
        currentStreamingIdRef.current = null;
        currentStreamingSessionIdRef.current = null;
        skipNextLoadSessionIdRef.current = null;

        if (event.tokensPrompt || event.tokensCompletion) {
          setTelemetry((prev) => ({
            ...prev,
            status: 'idle',
            tokensPrompt: prev.tokensPrompt + (event.tokensPrompt || 0),
            tokensCompletion: prev.tokensCompletion + (event.tokensCompletion || 0),
            activeModel: event.modelId || prev.activeModel,
            latencyMs: elapsed,
          }));
        } else {
          setTelemetry((prev) => ({
            ...prev,
            status: 'idle',
            latencyMs: elapsed,
          }));
        }
        if (onRefreshSessions) {
          onRefreshSessions();
        }
        return;
      }

      if (event.type === 'error') {
        const elapsed = Math.round(performance.now() - streamStartTimeRef.current);
        const errMsg = event.error || 'Ocurrió un error inesperado al procesar la respuesta.';
        console.error(`[Merlin Chat] Error en stream (${elapsed}ms):`, errMsg);
        setMessages((prev) =>
          prev.map((m) =>
            m.id === event.messageId
              ? {
                  ...m,
                  content: m.content ? `${m.content}\n\n[Error: ${errMsg}]` : `Error: ${errMsg}`,
                  status: 'error',
                }
              : m
          )
        );
        setIsLoading(false);
        setStreamingStatusText('');
        currentStreamingIdRef.current = null;
        currentStreamingSessionIdRef.current = null;
        skipNextLoadSessionIdRef.current = null;
        setTelemetry((prev) => ({
          ...prev,
          status: 'error',
          latencyMs: elapsed,
        }));
        if (onRefreshSessions) {
          onRefreshSessions();
        }
      }
    });

    return () => {
      if (typeof unsubscribe === 'function') {
        unsubscribe();
      }
    };
  }, [onRefreshSessions]);

  const cancelStream = useCallback(async () => {
    if (currentStreamingIdRef.current) {
      const streamId = currentStreamingIdRef.current;
      const elapsedSec = Math.max(1, Math.round((performance.now() - streamStartTimeRef.current) / 1000));
      console.log(`[Merlin Chat] Cancelando stream: ${streamId}`);
      try {
        await CancelChatStream(streamId);
      } catch (err) {
        console.error(`[Merlin Chat] Error invocando CancelChatStream:`, err);
      }
      setMessages((prev) =>
        prev.map((m) =>
          m.id === streamId
            ? {
                ...m,
                content: m.content ? `${m.content}\n\n*(Generación detenida)*` : '*(Generación detenida)*',
                status: 'done',
                durationSeconds: elapsedSec,
              }
            : m
        )
      );
      setIsLoading(false);
      setStreamingStatusText('');
      currentStreamingIdRef.current = null;
      currentStreamingSessionIdRef.current = null;
      skipNextLoadSessionIdRef.current = null;
      if (onRefreshSessions) {
        onRefreshSessions();
      }
    }
  }, [onRefreshSessions]);

  const sendMessage = useCallback(
    async (text: string) => {
      const trimmed = text.trim();
      if (!trimmed || isLoading) return;

      const userMsgId = `usr-${Date.now()}`;
      const userMsg: Message = {
        id: userMsgId,
        role: 'user',
        content: trimmed,
        timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      };

      // Garantizar que exista una sesión activa
      let currentSessionId = activeSessionId;
      if (onEnsureSession) {
        const ensuredId = onEnsureSession(trimmed);
        if (ensuredId) {
          currentSessionId = ensuredId;
        }
      } else if (activeSessionId && onIncrementSessionCount) {
        onIncrementSessionCount(activeSessionId);
      }

      if (currentSessionId) {
        skipNextLoadSessionIdRef.current = currentSessionId;
      }

      // Si no hay carpeta/proyecto activo, requerir consentimiento y selección
      if (!activeProject || !activeProject.path) {
        setMessages((prev) => [
          ...prev,
          userMsg,
          {
            id: `asst-${Date.now()}`,
            role: 'assistant',
            content: `Para poder trabajar y procesar tus instrucciones con seguridad, primero debes seleccionar una carpeta de proyecto. Merlin solo modificará los archivos que se encuentren dentro del directorio que elijas.`,
            timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
            action: {
              type: 'open_folder',
              label: '📁 Seleccionar carpeta de trabajo',
            },
          },
        ]);
        return;
      }

      const asstMessageId = `asst-${Date.now()}`;
      currentStreamingSessionIdRef.current = currentSessionId || null;
      currentStreamingIdRef.current = asstMessageId;
      streamStartTimeRef.current = performance.now();

      const placeholderAssistantMsg: Message = {
        id: asstMessageId,
        role: 'assistant',
        content: '',
        thoughtChain: '',
        timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
        status: 'thinking',
        startedAt: Date.now(),
        tokensPrompt: 0,
        tokensCompletion: 0,
      };

      const historyToSend = messages
        .filter((m) => m.role === 'user' || m.role === 'assistant')
        .map((m) => ({
          role: m.role,
          content: m.content,
        }));

      setMessages((prev) => [...prev, userMsg, placeholderAssistantMsg]);
      setIsLoading(true);
      setStreamingStatusText('Iniciando conexión...');

      setTelemetry((prev) => ({
        ...prev,
        status: 'synthesizing',
        activeModel: selectedModelId || prev.activeModel,
        tokensPrompt: prev.tokensPrompt + Math.max(1, Math.floor(trimmed.length / 4)),
      }));

      console.log(`[Merlin Chat] Iniciando envío de mensaje:`, {
        prompt: trimmed,
        provider: selectedProviderId,
        model: selectedModelId || 'automático',
        historyCount: historyToSend.length,
      });

      StartChatStream(
        new domain.ChatStreamRequest({
          sessionId: currentSessionId || 'default-session',
          messageId: asstMessageId,
          userMessageId: userMsgId,
          providerId: selectedProviderId,
          modelId: selectedModelId,
          prompt: trimmed,
          history: historyToSend,
        })
      ).catch((err: any) => {
        const elapsed = Math.round(performance.now() - streamStartTimeRef.current);
        const errMsg = err?.message || err || 'No se pudo conectar con el backend de IA.';
        console.error(`[Merlin Chat] Error al iniciar stream (${elapsed}ms):`, errMsg);
        setMessages((prev) =>
          prev.map((m) =>
            m.id === asstMessageId
              ? {
                  ...m,
                  content: `Error: ${String(errMsg)}`,
                  status: 'error',
                }
              : m
          )
        );
        setIsLoading(false);
        setStreamingStatusText('');
        currentStreamingIdRef.current = null;
        currentStreamingSessionIdRef.current = null;
        skipNextLoadSessionIdRef.current = null;
        setTelemetry((prev) => ({
          ...prev,
          status: 'error',
          latencyMs: elapsed,
        }));
      });
    },
    [
      isLoading,
      messages,
      activeProject,
      activeSessionId,
      selectedProviderId,
      selectedModelId,
      onEnsureSession,
      onIncrementSessionCount,
    ]
  );

  const handleFeedback = useCallback((messageId: string, feedback: 'like' | 'dislike' | null) => {
    setMessages((prev) =>
      prev.map((m) =>
        m.id === messageId
          ? {
              ...m,
              feedback,
            }
          : m
      )
    );

    UpdateMessageFeedback(messageId, feedback || '').catch((err) => {
      console.error('[Merlin Chat] Error persistiendo calificación en SQLite:', err);
    });
  }, []);

  return {
    messages,
    setMessages,
    clearMessages,
    isLoading,
    streamingStatusText,
    cancelStream,
    telemetry,
    setTelemetry,
    sendMessage,
    handleFeedback,
  };
}
