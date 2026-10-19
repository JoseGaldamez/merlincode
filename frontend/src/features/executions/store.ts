import { isExecutionActive, type Execution } from './types';

// Snapshots include a monotonic backend revision. Delayed events cannot restore
// cleared logs or overwrite a more recent IPC response.
export class ExecutionStore {
  private runs: Execution[] = [];
  private listeners = new Set<() => void>();
  private removed = new Map<string, number>();

  getSnapshot = () => this.runs;
  subscribe = (listener: () => void) => {
    this.listeners.add(listener);
    return () => { this.listeners.delete(listener); };
  };
  receive = (run: Execution) => {
    if (!run?.proposal?.id) return;
    if ((this.removed.get(run.proposal.id) ?? -1) >= run.revision) return;
    const previous = this.runs.find((item) => item.proposal.id === run.proposal.id);
    if (previous && previous.revision >= run.revision) return;
    if (run.status === 'removed') {
      this.removed.set(run.proposal.id, run.revision);
      this.runs = this.runs.filter((item) => item.proposal.id !== run.proposal.id);
      this.listeners.forEach((listener) => listener());
      return;
    }
    this.removed.delete(run.proposal.id);
    this.runs = [...this.runs.filter((item) => item.proposal.id !== run.proposal.id), run];
    if (this.runs.length > 24) {
      const finished = this.runs.findIndex((item) => !isExecutionActive(item.status));
      if (finished >= 0) this.runs.splice(finished, 1);
    }
    this.listeners.forEach((listener) => listener());
  };
}
