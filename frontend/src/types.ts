export interface MessageAction {
  type: 'open_folder';
  label: string;
}

export interface Message {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: string;
  thoughtChain?: string;
  codeSnippet?: {
    language: string;
    code: string;
    filename?: string;
  };
  status?: 'idle' | 'thinking' | 'synthesizing' | 'done' | 'error';
  action?: MessageAction;
  startedAt?: number;
  durationSeconds?: number;
  tokensPrompt?: number;
  tokensCompletion?: number;
  feedback?: 'like' | 'dislike' | null;
}

export interface Session {
  id: string;
  title: string;
  date: string;
  messagesCount: number;
  projectId?: string;
  projectPath?: string;
}

export interface Artifact {
  id: string;
  title: string;
  type: 'code' | 'markdown' | 'diff' | 'file';
  language?: string;
  content?: string;
  timestamp: string;
  size?: string;
}

export interface AgentTelemetry {
  status: 'idle' | 'synthesizing' | 'human_input' | 'error';
  activeModel: string;
  tokensPrompt: number;
  tokensCompletion: number;
  latencyMs: number;
  activeFile: string;
  temperature: number;
}

export interface Project {
  id: string;
  name: string;
  path: string;
}

export interface ChatStreamEvent {
  sessionId?: string;
  messageId: string;
  type: 'status' | 'thinking' | 'content' | 'done' | 'error';
  content?: string;
  thinking?: string;
  error?: string;
  statusText?: string;
  providerId?: string;
  modelId?: string;
  tokensPrompt?: number;
  tokensCompletion?: number;
}

