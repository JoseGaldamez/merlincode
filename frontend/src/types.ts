export interface ToolApprovalRequest {
  requestId: string;
  toolName: string;
  path: string;
  newContent: string;
  oldContent?: string;
  expiresAt: number; // epoch ms — tras este momento el backend rechaza automáticamente la solicitud
}

export interface ToolActivityEntry {
  toolCallId: string;
  toolName: string;
  argsSummary?: string;
  status: 'running' | 'awaiting_approval' | 'done' | 'error' | 'rejected';
  approvalRequest?: ToolApprovalRequest;
}

export type MessageAction =
  | { type: 'open_folder'; label: string }
  | { type: 'tool_approval'; label: string; request: ToolApprovalRequest };

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
  status?: 'idle' | 'thinking' | 'synthesizing' | 'done' | 'error' | 'tool_running' | 'awaiting_approval';
  action?: MessageAction;
  modelId?: string;
  startedAt?: number;
  durationSeconds?: number;
  tokensPrompt?: number;
  tokensCompletion?: number;
  feedback?: 'like' | 'dislike' | null;
  toolActivity?: ToolActivityEntry[];
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
  type:
    | 'status'
    | 'thinking'
    | 'content'
    | 'done'
    | 'error'
    | 'tool_call'
    | 'tool_approval_required'
    | 'tool_approval_resolved';
  content?: string;
  thinking?: string;
  error?: string;
  statusText?: string;
  providerId?: string;
  modelId?: string;
  tokensPrompt?: number;
  tokensCompletion?: number;
  toolName?: string;
  toolCallId?: string;
  toolArgsSummary?: string;
  approvalRequest?: ToolApprovalRequest;
}

