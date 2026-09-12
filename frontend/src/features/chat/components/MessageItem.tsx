import React from 'react';
import { Message } from '../../../types';
import logoImg from '../../../assets/images/logo.png';
import { IconFolder } from '../../../components/Icons';
import { ThoughtChain } from './ThoughtChain';
import { CodeSnippetView } from './CodeSnippetView';

interface MessageItemProps {
  message: Message;
  onOpenFolder?: () => void;
}

export const MessageItem: React.FC<MessageItemProps> = ({ message, onOpenFolder }) => {
  const isAssistant = message.role === 'assistant';

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
                <span className="text-xs font-mono font-bold text-content-dim tracking-wider">
                  MERLIN
                </span>
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
          <p className="whitespace-pre-wrap">{message.content}</p>

          {/* Optional Code Block */}
          {message.codeSnippet && (
            <CodeSnippetView
              code={message.codeSnippet.code}
              language={message.codeSnippet.language}
              filename={message.codeSnippet.filename}
              messageId={message.id}
            />
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
        </div>
      </div>
    </div>
  );
};
