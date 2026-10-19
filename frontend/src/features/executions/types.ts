export interface RunRequest {
  directory: string;
  entry?: string;
}

export interface RunProposal {
  id: string;
  project: string;
  directory: string;
  title: string;
  command: string;
  reason: string;
  request: RunRequest;
}

export interface Execution {
  proposal: RunProposal;
  status: 'starting' | 'running' | 'stopping' | 'stopped' | 'exited' | 'error' | 'removed';
  logs: string;
  url?: string;
  error?: string;
  exitCode?: number;
  revision: number;
}

export interface ExecutionFocus {
  id: string;
  command: string;
  source?: DOMRect;
  nonce: number;
}

export const isExecutionActive = (status?: Execution['status']) =>
  status === 'starting' || status === 'running' || status === 'stopping';
