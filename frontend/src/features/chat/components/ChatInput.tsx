import React, { useState } from 'react';
import { IconArrowUp, IconStop } from '../../../components/Icons';
import { useAutoResizeTextarea } from '../../../hooks/useAutoResizeTextarea';

interface ChatInputProps {
  onSendMessage: (text: string) => void;
  isLoading: boolean;
  onCancelStream?: () => void;
  streamingStatusText?: string;
  disabled?: boolean;
  placeholder?: string;
}

export const ChatInput: React.FC<ChatInputProps> = ({
  onSendMessage,
  isLoading,
  onCancelStream,
  disabled = false,
  placeholder = 'Escribe un mensaje...',
}) => {
  const [inputText, setInputText] = useState('');
  const textareaRef = useAutoResizeTextarea(inputText, { minHeight: 24, maxHeight: 140 });

  const handleSubmit = (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    if (!inputText.trim() || isLoading || disabled) return;
    const text = inputText;
    setInputText('');
    onSendMessage(text);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const canSubmit = Boolean(inputText.trim()) && !isLoading && !disabled;
  const isMultiLine = inputText.includes('\n');

  return (
    <div className="px-4 md:px-8 pb-4 pt-1 bg-transparent shrink-0 w-full">
      <form className="max-w-[960px] w-full mx-auto" onSubmit={handleSubmit}>
        <div
          className={`w-full flex ${
            isMultiLine ? 'items-end' : 'items-center'
          } gap-2.5 bg-[#101619] border border-border-subtle focus-within:border-accent-primary/70 focus-within:ring-1 focus-within:ring-accent-primary/20 rounded-xl px-3.5 py-2 transition-all shadow-lg shadow-black/25`}
        >
          {/* Textarea alineado perfectamente al centro */}
          <textarea
            ref={textareaRef}
            className="flex-1 bg-transparent text-sm text-content-headline placeholder-content-dim font-mono outline-none resize-none p-0 leading-6 min-h-[24px] max-h-[140px] selection:bg-accent-primary/20"
            value={inputText}
            onChange={(e) => setInputText(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            rows={1}
            disabled={disabled}
          />

          {/* Botón de acción moderno */}
          {isLoading ? (
            <button
              type="button"
              onClick={(e) => {
                e.preventDefault();
                onCancelStream?.();
              }}
              className="w-8 h-8 rounded-lg flex items-center justify-center bg-red-950/60 hover:bg-red-900 border border-red-500/40 text-red-400 hover:text-red-300 transition-all cursor-pointer shrink-0 active:scale-95 shadow-sm"
              title="Detener respuesta"
              aria-label="Detener respuesta"
            >
              <IconStop size={12} />
            </button>
          ) : (
            <button
              type="submit"
              disabled={!canSubmit}
              className={`w-8 h-8 rounded-lg flex items-center justify-center transition-all shrink-0 ${
                canSubmit
                  ? 'bg-accent-primary hover:bg-accent-primary-hover text-[#081013] shadow-md shadow-accent-primary/15 hover:shadow-accent-primary/30 cursor-pointer active:scale-95'
                  : 'bg-[#162024] text-content-dim/35 cursor-not-allowed border border-border-subtle/40'
              }`}
              title="Enviar mensaje"
              aria-label="Enviar mensaje"
            >
              <IconArrowUp size={16} />
            </button>
          )}
        </div>
      </form>
    </div>
  );
};
