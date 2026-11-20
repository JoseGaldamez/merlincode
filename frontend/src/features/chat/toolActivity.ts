import type { domain } from '../../../wailsjs/go/models';
import type { ToolActivityEntry } from '../../types';

function summarizeSavedArguments(raw: string): string | undefined {
  try {
    const args: unknown = JSON.parse(raw);
    if (!args || typeof args !== 'object') return;
    if ('path' in args && typeof args.path === 'string' && args.path) return args.path;
    if ('directory' in args && typeof args.directory === 'string') return args.directory;
  } catch {
    // Some legacy entries have missing or malformed arguments.
  }
}

export function activityFromTrace(trace?: readonly domain.ToolTraceEntry[]): ToolActivityEntry[] | undefined {
  return trace?.map((entry, index) => ({
    toolCallId: `saved-${index}`,
    toolName: entry.toolName,
    argsSummary: summarizeSavedArguments(entry.arguments),
    status: entry.approved === false ? 'rejected' : entry.isError ? 'error' : 'done',
    error: entry.isError ? entry.result : undefined,
  }));
}
