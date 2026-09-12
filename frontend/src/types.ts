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
  status?: 'idle' | 'thinking' | 'done' | 'error';
  action?: MessageAction;
}

export interface Session {
  id: string;
  title: string;
  date: string;
  messagesCount: number;
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

