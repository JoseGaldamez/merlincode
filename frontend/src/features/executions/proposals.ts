import type { RunProposal } from './types';

export function proposalsFromTrace(trace?: readonly { toolName: string; result: string; isError?: boolean }[]): RunProposal[] {
  const result = new Map<string, RunProposal>();
  for (const entry of trace || []) {
    if (entry.toolName !== 'propose_app_run' || entry.isError) continue;
    try {
      const value: unknown = JSON.parse(entry.result);
      if (!Array.isArray(value)) continue;
      for (const item of value) {
        if (item && typeof item === 'object' && typeof item.id === 'string' && /^[a-f0-9]{32}$/.test(item.id)
          && typeof item.command === 'string' && typeof item.project === 'string' && typeof item.title === 'string'
          && typeof item.reason === 'string' && typeof item.directory === 'string'
          && item.request && typeof item.request.directory === 'string'
          && (item.request.entry === undefined || typeof item.request.entry === 'string')) {
          result.set(item.id, item as RunProposal);
        }
      }
    } catch { /* Older or malformed trace entries do not become executable cards. */ }
  }
  return [...result.values()];
}
