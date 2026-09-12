import React, { useState, useRef, useEffect } from 'react';
import { Message } from '../types';
import logoImg from '../assets/images/logo.png';
import {
  IconSend,
  IconSparkles,
  IconChevronDown,
  IconChevronRight,
  IconCopy,
  IconCheck,
} from './Icons';

interface ChatAreaProps {
  messages: Message[];
  onSendMessage: (text: string) => void;
  isLoading: boolean;
}

export const ChatArea: React.FC<ChatAreaProps> = ({
  messages,
  onSendMessage,
  isLoading,
}) => {
  const [inputText, setInputText] = useState('');
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [openThoughts, setOpenThoughts] = useState<Record<string, boolean>>({});
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages, isLoading]);

  // Ajustar altura dinámicamente hasta 4 líneas sin scrollbar prematuro
  useEffect(() => {
    const el = textareaRef.current;
    if (!el) return;

    el.style.height = 'auto';
    const minHeight = 38; // 1 línea
    const maxFourLinesHeight = 102; // Hasta 4 líneas
    const scrollH = el.scrollHeight;

    if (!inputText) {
      el.style.height = `${minHeight}px`;
      el.style.overflowY = 'hidden';
    } else if (scrollH <= maxFourLinesHeight) {
      el.style.height = `${Math.max(minHeight, scrollH)}px`;
      el.style.overflowY = 'hidden';
    } else {
      el.style.height = `${maxFourLinesHeight}px`;
      el.style.overflowY = 'auto';
    }
  }, [inputText]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputText.trim() || isLoading) return;
    const text = inputText;
    setInputText('');
    onSendMessage(text);
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setInputText(e.target.value);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  const toggleThought = (id: string) => {
    setOpenThoughts((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  return (
    <main className="flex-1 h-full flex flex-col bg-canvas overflow-hidden min-w-0">
      {/* Messages Scroll Area */}
      <div className="flex-1 overflow-y-auto px-4 md:px-8 py-6 flex flex-col gap-6">
        {messages.length === 0 ? (
          <div className="flex-1 flex items-center justify-center">
            <div className="flex flex-col items-center justify-center p-8 text-center max-w-sm">
              <div className="w-16 h-16 rounded-full bg-surface-card border border-border-subtle p-3 flex items-center justify-center mb-4 shadow-sm">
                <img src={logoImg} alt="Merlin Code Logo" className="w-full h-full object-contain" />
              </div>
              <h2 className="text-base font-semibold text-content-headline mb-1.5 font-sans">
                Merlin Code
              </h2>
              <p className="text-xs text-content-dim leading-relaxed">
                Escribe un mensaje o instrucción en el panel inferior para comenzar.
              </p>
            </div>
          </div>
        ) : (
          messages.map((msg) => {
            const isAssistant = msg.role === 'assistant';
            const isThoughtOpen = openThoughts[msg.id] ?? true;

            return (
              <div
                key={msg.id}
                className="flex flex-col w-full"
              >
                <div className="max-w-3xl w-full mx-auto">
                  {/* Message Header */}
                  <div className="flex items-center justify-between gap-3 mb-1.5">
                    <div className="flex items-center gap-2">
                      {isAssistant ? (
                        <>
                          <div className="w-5 h-5 rounded flex items-center justify-center bg-[#15272e] text-accent-primary border border-border-petrol">
                            <IconSparkles size={12} />
                          </div>
                          <span className="text-[11px] font-mono font-bold text-content-dim tracking-wider">
                            MERLIN
                          </span>
                        </>
                      ) : (
                        <>
                          <div className="w-5 h-5 rounded flex items-center justify-center bg-[#1c2428] text-content-muted text-[9px] font-mono font-bold border border-border-subtle">
                            <span>TU</span>
                          </div>
                          <span className="text-[11px] font-mono font-bold text-content-dim tracking-wider">
                            USUARIO
                          </span>
                        </>
                      )}
                    </div>
                    <span className="text-[10px] font-mono text-content-dim">{msg.timestamp}</span>
                  </div>

                  {/* Optional Thought Stream Accordion */}
                  {isAssistant && msg.thoughtChain && (
                    <div className="ml-7 mb-2 rounded border border-border-petrol bg-[#10171b] overflow-hidden">
                      <button
                        className="w-full flex items-center gap-2 px-2.5 py-1.5 text-xs text-content-muted hover:text-content-body hover:bg-[#152227] transition-colors cursor-pointer text-left"
                        onClick={() => toggleThought(msg.id)}
                      >
                        {isThoughtOpen ? (
                          <IconChevronDown size={13} />
                        ) : (
                          <IconChevronRight size={13} />
                        )}
                        <span className="text-[10px] font-mono text-accent-cyan tracking-wider font-semibold">
                          CADENA DE RAZONAMIENTO
                        </span>
                      </button>
                      {isThoughtOpen && (
                        <div className="p-2.5 text-xs font-mono text-content-dim bg-[#0a0f12] border-t border-border-petrol whitespace-pre-wrap leading-relaxed">
                          {msg.thoughtChain}
                        </div>
                      )}
                    </div>
                  )}

                  {/* Message Body */}
                  <div className="text-xs text-content-body leading-relaxed pl-7">
                    <p className="whitespace-pre-wrap">{msg.content}</p>

                    {/* Optional Code Block */}
                    {msg.codeSnippet && (
                      <div className="mt-2.5 rounded border border-border-subtle bg-[#0c1012] overflow-hidden">
                        <div className="flex items-center justify-between px-3 py-1.5 bg-[#12181b] border-b border-border-subtle">
                          <span className="text-[10px] font-mono text-content-dim uppercase tracking-wider">
                            {msg.codeSnippet.filename || msg.codeSnippet.language}
                          </span>
                          <button
                            className="flex items-center gap-1.5 text-[10px] font-mono text-content-dim hover:text-accent-primary transition-colors cursor-pointer"
                            onClick={() =>
                              copyToClipboard(
                                msg.codeSnippet!.code,
                                `code-${msg.id}`
                              )
                            }
                            title="Copiar código"
                          >
                            {copiedId === `code-${msg.id}` ? (
                              <>
                                <IconCheck size={12} className="text-status-success" />
                                <span>Copiado</span>
                              </>
                            ) : (
                              <>
                                <IconCopy size={12} />
                                <span>Copiar</span>
                              </>
                            )}
                          </button>
                        </div>
                        <pre className="p-3 overflow-x-auto text-xs font-mono text-[#d1d5db] leading-5">
                          <code>
                            {msg.codeSnippet.code}
                          </code>
                        </pre>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            );
          })
        )}

        {isLoading && (
          <div className="flex flex-col w-full">
            <div className="max-w-3xl w-full mx-auto">
              <div className="flex items-center gap-2 mb-1.5">
                <div className="w-5 h-5 rounded flex items-center justify-center bg-[#15272e] text-accent-primary border border-border-petrol">
                  <IconSparkles size={12} />
                </div>
                <span className="text-[11px] font-mono font-bold text-content-dim tracking-wider">
                  MERLIN
                </span>
              </div>
              <div className="flex items-center gap-2 pl-7 py-2">
                <span className="w-2 h-2 rounded-full bg-accent-primary animate-pulse" />
                <span className="text-xs text-content-dim font-mono">Procesando...</span>
              </div>
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Input Prompt Section */}
      <div className="p-3 md:p-4 border-t border-border-subtle bg-[#0d1214] shrink-0">
        <form className="max-w-3xl w-full mx-auto" onSubmit={handleSubmit}>
          <div className="flex items-stretch gap-2 bg-[#12181b] border border-border-subtle focus-within:border-accent-primary rounded-md p-1.5 transition-colors shadow-sm">
            <textarea
              ref={textareaRef}
              className="flex-1 bg-transparent text-xs text-content-headline placeholder-content-dim font-mono outline-none resize-none px-2 py-1.5 leading-5 min-h-[38px] max-h-[102px]"
              value={inputText}
              onChange={handleInputChange}
              onKeyDown={handleKeyDown}
              placeholder="Escribe un mensaje o instrucción... (Enter para enviar, Shift+Enter para salto de línea)"
              rows={1}
            />

            <button
              type="submit"
              className="flex items-center justify-center gap-1.5 px-4 bg-accent-primary hover:bg-accent-primary-hover disabled:opacity-40 disabled:hover:bg-accent-primary text-[#0b0e10] font-semibold text-xs rounded transition-colors cursor-pointer shrink-0 min-h-[38px]"
              disabled={!inputText.trim() || isLoading}
              title="Enviar instrucción"
            >
              <IconSend size={14} />
              <span>Enviar</span>
            </button>
          </div>
        </form>
      </div>
    </main>
  );
};
