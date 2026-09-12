import React, { useState } from 'react';
import { IconSend } from '../../../components/Icons';
import { useAutoResizeTextarea } from '../../../hooks/useAutoResizeTextarea';

interface ChatInputProps {
  onSendMessage: (text: string) => void;
  isLoading: boolean;
  disabled?: boolean;
  placeholder?: string;
}

export const ChatInput: React.FC<ChatInputProps> = ({
  onSendMessage,
  isLoading,
  disabled = false,
  placeholder = 'Escribe un mensaje...',
}) => {
  const [inputText, setInputText] = useState('');
  const textareaRef = useAutoResizeTextarea(inputText, { minHeight: 40, maxHeight: 112 });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputText.trim() || isLoading || disabled) return;
    const text = inputText;
    setInputText('');
    onSendMessage(text);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  return (
    <div className="px-4 md:px-8 pb-4 pt-1 bg-transparent shrink-0 w-full">
      <form className="max-w-[960px] w-full mx-auto" onSubmit={handleSubmit}>
        <div className="w-full flex items-stretch gap-2 bg-[#12181b] border border-border-subtle focus-within:border-accent-primary rounded-lg p-1.5 transition-colors">
          <textarea
            ref={textareaRef}
            className="flex-1 bg-transparent text-sm text-content-headline placeholder-content-dim font-mono outline-none resize-none px-3 py-2 leading-6 min-h-[40px] max-h-[112px]"
            value={inputText}
            onChange={(e) => setInputText(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            rows={1}
            disabled={disabled}
          />

          <button
            type="submit"
            className="flex items-center justify-center gap-1.5 px-4 bg-accent-primary hover:bg-accent-primary-hover disabled:opacity-40 disabled:hover:bg-accent-primary text-[#0b0e10] font-semibold text-xs rounded-md transition-colors cursor-pointer shrink-0 min-h-[40px]"
            disabled={!inputText.trim() || isLoading || disabled}
            title="Enviar instrucción"
          >
            <IconSend size={14} />
            <span>Enviar</span>
          </button>
        </div>
      </form>
    </div>
  );
};
