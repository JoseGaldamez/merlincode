import React, { useRef, useEffect } from 'react';
import { Message } from '../../../types';
import logoImg from '../../../assets/images/logo.png';
import { EmptyChatState } from './EmptyChatState';
import { MessageItem } from './MessageItem';
import { ChatInput } from './ChatInput';

export interface ChatAreaProps {
  messages: Message[];
  onSendMessage: (text: string) => void;
  isLoading: boolean;
  activeProjectName?: string;
  onOpenFolder?: () => void;
}

export const ChatArea: React.FC<ChatAreaProps> = ({
  messages,
  onSendMessage,
  isLoading,
  activeProjectName,
  onOpenFolder,
}) => {
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages, isLoading]);

  return (
    <main className="flex-1 h-full flex flex-col bg-canvas overflow-hidden min-w-0">
      {/* Messages Scroll Area */}
      <div className="flex-1 overflow-y-auto px-4 md:px-8 py-6">
        <div className="max-w-[960px] w-full mx-auto flex flex-col gap-6 min-h-full">
          {messages.length === 0 ? (
            <EmptyChatState
              activeProjectName={activeProjectName}
              onOpenFolder={onOpenFolder}
            />
          ) : (
            messages.map((msg) => (
              <MessageItem
                key={msg.id}
                message={msg}
                onOpenFolder={onOpenFolder}
              />
            ))
          )}

          {isLoading && (
            <div className="w-full flex flex-col items-start">
              <div className="w-full max-w-[960px] mr-auto">
                <div className="flex items-center gap-2 mb-1.5 px-1">
                  <img src={logoImg} alt="Merlin" className="w-5 h-5 object-contain select-none" />
                  <span className="text-xs font-mono font-bold text-content-dim tracking-wider">
                    MERLIN
                  </span>
                </div>
                <div className="flex items-center gap-2.5 py-1.5 px-1 w-fit">
                  <span className="w-2 h-2 rounded-full bg-accent-primary animate-pulse" />
                  <span className="text-xs text-content-dim font-mono">Procesando...</span>
                </div>
              </div>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>
      </div>

      {/* Floating Input Prompt Section */}
      <ChatInput
        onSendMessage={onSendMessage}
        isLoading={isLoading}
      />
    </main>
  );
};
