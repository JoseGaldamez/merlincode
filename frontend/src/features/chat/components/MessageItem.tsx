import React, { useState, useEffect } from 'react';
import { Message } from '../../../types';
import logoImg from '../../../assets/images/logo.png';
import {
  IconFolder,
  IconCopy,
  IconCheck,
  IconThumbUp,
  IconThumbDown,
} from '../../../components/Icons';
import { ThoughtChain } from './ThoughtChain';
import { FormattedMessageContent } from './FormattedMessageContent';

interface MessageItemProps {
  message: Message;
  onOpenFolder?: () => void;
  onFeedback?: (messageId: string, feedback: 'like' | 'dislike' | null) => void;
}

const LiveTimer: React.FC<{
  startedAt?: number;
  durationSeconds?: number;
  isLive: boolean;
}> = ({ startedAt, durationSeconds, isLive }) => {
  const [elapsed, setElapsed] = useState(() => {
    if (!isLive && durationSeconds != null) return durationSeconds;
    if (startedAt) return Math.floor((Date.now() - startedAt) / 1000);
    return 0;
  });

  useEffect(() => {
    if (!isLive) return;
    const start = startedAt || Date.now();
    const updateElapsed = () => {
      setElapsed(Math.floor((Date.now() - start) / 1000));
    };
    updateElapsed();
    const timer = setInterval(updateElapsed, 250);
    return () => clearInterval(timer);
  }, [isLive, startedAt]);

  if (isLive) {
    return (
      <span className="inline-flex items-center gap-1.5 text-xs font-mono text-accent-primary font-medium">
        <span className="w-1.5 h-1.5 rounded-full bg-accent-primary animate-pulse" />
        <span>{elapsed} seg</span>
      </span>
    );
  }

  const finalSec = durationSeconds ?? Math.max(1, elapsed);
  return (
    <span className="text-xs font-mono font-medium text-content-dim">
      Resuelto en {finalSec} seg
    </span>
  );
};

export const MessageItem: React.FC<MessageItemProps> = ({ message, onOpenFolder, onFeedback }) => {
  const isAssistant = message.role === 'assistant';
  const isLive = isAssistant && (message.status === 'thinking' || message.status === 'synthesizing');

  const [copied, setCopied] = useState(false);
  const [feedback, setFeedback] = useState<'like' | 'dislike' | null>(message.feedback || null);

  useEffect(() => {
    setFeedback(message.feedback || null);
  }, [message.feedback]);

  const handleCopy = async () => {
    if (!message.content) return;
    try {
      await navigator.clipboard.writeText(message.content);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Error al copiar:', err);
    }
  };

  const handleThumb = (type: 'like' | 'dislike') => {
    const next = feedback === type ? null : type;
    setFeedback(next);
    onFeedback?.(message.id, next);
  };

  return (
    <div
      className={`w-full flex flex-col ${
        isAssistant ? 'items-start' : 'items-end'
      }`}
    >
      <div
        className={`w-full flex flex-col ${
          isAssistant
            ? 'max-w-[960px] mr-auto'
            : 'max-w-2xl md:max-w-3xl ml-auto items-end'
        }`}
      >
        {/* Message Header */}
        <div
          className={`flex items-center gap-2 mb-1.5 px-1 ${
            isAssistant ? 'justify-start' : 'justify-end flex-row-reverse'
          }`}
        >
          <div className="flex items-center gap-2">
            {isAssistant ? (
              <>
                <img src={logoImg} alt="Merlin" className="w-5 h-5 object-contain select-none" />
                <LiveTimer
                  startedAt={message.startedAt}
                  durationSeconds={message.durationSeconds}
                  isLive={isLive}
                />
              </>
            ) : (
              <>
                <div className="w-5 h-5 rounded flex items-center justify-center bg-[#1f353d] text-accent-primary text-[10px] font-mono font-bold border border-border-petrol">
                  <span>TU</span>
                </div>
                <span className="text-xs font-mono font-bold text-content-dim tracking-wider">
                  USUARIO
                </span>
              </>
            )}
          </div>
          <span className="text-[11px] font-mono text-content-dim">{message.timestamp}</span>
        </div>

        {/* Optional Thought Stream Accordion */}
        {isAssistant && message.thoughtChain && (
          <ThoughtChain thoughts={message.thoughtChain} />
        )}

        {/* Message Body */}
        <div
          className={`text-md leading-relaxed transition-all ${
            isAssistant
              ? 'w-full py-1 px-1 text-content-body'
              : 'p-3.5 md:p-4 rounded-2xl rounded-tr-sm bg-[#16252c] border border-[#233d47] text-[#f1f5f9] text-left max-w-full'
          }`}
        >
          {message.status === 'error' ? (
            <div className="p-3.5 rounded-xl bg-red-950/30 border border-red-500/30 text-red-200 text-sm flex flex-col gap-1.5 my-1">
              <div className="flex items-center gap-2 text-red-400 font-mono text-xs font-semibold">
                <span>⚠️ Error al procesar solicitud</span>
              </div>
              <p className="whitespace-pre-wrap font-mono text-xs text-red-300/90 leading-relaxed">
                {message.content}
              </p>
            </div>
          ) : isAssistant && !message.content && !message.thoughtChain ? (
            <div className="flex items-center gap-2.5 py-1 text-xs text-content-dim font-mono">
              <span className="w-2 h-2 rounded-full bg-accent-primary animate-pulse" />
              <span>Conectando y procesando respuesta...</span>
            </div>
          ) : (
            <FormattedMessageContent content={message.content} messageId={message.id} />
          )}

          {/* Optional Action Button inside message */}
          {message.action && message.action.type === 'open_folder' && onOpenFolder && (
            <div className="mt-3">
              <button
                type="button"
                onClick={onOpenFolder}
                className="inline-flex items-center gap-2 px-4 py-2 bg-accent-primary hover:bg-accent-primary-hover text-[#0b0e10] font-semibold text-xs rounded-lg transition-colors cursor-pointer shadow-md"
              >
                <IconFolder size={15} />
                <span>{message.action.label}</span>
              </button>
            </div>
          )}

          {/* Footer toolbar: SOLO visible cuando la respuesta ha terminado por completo */}
          {isAssistant && message.status === 'done' && message.content && (
            <div className="flex items-center justify-between gap-3 mt-3.5 pt-2.5 border-t border-border-subtle/25">
              {/* Botones de acción: Copiar y Calificar */}
              <div className="flex items-center gap-1.5">
                <button
                  type="button"
                  onClick={handleCopy}
                  className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-mono transition-all cursor-pointer border ${
                    copied
                      ? 'bg-accent-primary/15 border-accent-primary/50 text-accent-primary'
                      : 'bg-[#12191d] hover:bg-[#1a252b] border-border-subtle/50 text-content-dim hover:text-content-body'
                  }`}
                  title={copied ? 'Copiado al portapapeles' : 'Copiar respuesta completa'}
                  aria-label="Copiar respuesta completa"
                >
                  {copied ? <IconCheck size={13} /> : <IconCopy size={13} />}
                  <span>{copied ? 'Copiado' : 'Copiar'}</span>
                </button>

                <button
                  type="button"
                  onClick={() => handleThumb('like')}
                  className={`inline-flex items-center justify-center w-7 h-7 rounded-md transition-all cursor-pointer border ${
                    feedback === 'like'
                      ? 'bg-emerald-950/60 border-emerald-500/50 text-emerald-400'
                      : 'bg-[#12191d] hover:bg-[#1a252b] border-border-subtle/50 text-content-dim hover:text-content-body'
                  }`}
                  title="Me gusta"
                  aria-label="Me gusta"
                >
                  <IconThumbUp size={13} />
                </button>

                <button
                  type="button"
                  onClick={() => handleThumb('dislike')}
                  className={`inline-flex items-center justify-center w-7 h-7 rounded-md transition-all cursor-pointer border ${
                    feedback === 'dislike'
                      ? 'bg-red-950/60 border-red-500/50 text-red-400'
                      : 'bg-[#12191d] hover:bg-[#1a252b] border-border-subtle/50 text-content-dim hover:text-content-body'
                  }`}
                  title="No me gusta"
                  aria-label="No me gusta"
                >
                  <IconThumbDown size={13} />
                </button>
              </div>

              {/* Conteo de tokens de entrada y salida */}
              {(message.tokensPrompt != null || message.tokensCompletion != null) && (
                <div className="inline-flex items-center gap-1.5 text-[11px] font-mono text-content-dim/80 bg-[#10171a] px-2.5 py-1 rounded-md border border-border-subtle/40 select-none">
                  <span className="text-content-dim/50">Tokens:</span>
                  <span className="text-content-headline font-semibold">{message.tokensPrompt ?? 0}</span>
                  <span className="text-content-dim/60">entrada</span>
                  <span className="text-content-dim/30">·</span>
                  <span className="text-content-headline font-semibold">{message.tokensCompletion ?? 0}</span>
                  <span className="text-content-dim/60">salida</span>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
