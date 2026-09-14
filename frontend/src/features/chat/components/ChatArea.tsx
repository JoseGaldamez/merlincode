import React, { useRef, useEffect } from 'react';
import { Message } from '../../../types';
import { EmptyChatState } from './EmptyChatState';
import { MessageItem } from './MessageItem';
import { ChatInput } from './ChatInput';

export interface ChatAreaProps {
  messages: Message[];
  onSendMessage: (text: string) => void;
  isLoading: boolean;
  onCancelStream?: () => void;
  streamingStatusText?: string;
  activeProjectName?: string;
  onOpenFolder?: () => void;
  onFeedback?: (messageId: string, feedback: 'like' | 'dislike' | null) => void;
}

export const ChatArea: React.FC<ChatAreaProps> = ({
  messages,
  onSendMessage,
  isLoading,
  onCancelStream,
  streamingStatusText,
  activeProjectName,
  onOpenFolder,
  onFeedback,
}) => {
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages, isLoading, streamingStatusText]);

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
                onFeedback={onFeedback}
              />
            ))
          )}

          <div ref={messagesEndRef} />
        </div>
      </div>

      {/* Floating Input Prompt Section */}
      <ChatInput
        onSendMessage={onSendMessage}
        isLoading={isLoading}
        onCancelStream={onCancelStream}
        streamingStatusText={streamingStatusText}
      />
    </main>
  );
};
