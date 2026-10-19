import { describe, expect, it } from 'vitest';
import { ExecutionStore } from './store';
import { proposalsFromTrace } from './proposals';
import type { Execution, RunProposal } from './types';

const proposal: RunProposal = { id: 'a'.repeat(32), project: '/app', directory: '.', title: 'dev', command: 'pnpm run dev', reason: 'package.json', request: { directory: '.' } };
const run = (revision: number, logs: string): Execution => ({ proposal, status: 'running', revision, logs });

describe('execution snapshots', () => {
  it('preserves silent active servers when the finished execution history fills up', () => {
    const store = new ExecutionStore();
    store.receive(run(1, 'silent server'));
    for (let index = 1; index <= 24; index++) {
      store.receive({ ...run(index + 1, 'finished'), proposal: { ...proposal, id: `finished-${index}` }, status: 'exited' });
    }
    expect(store.getSnapshot()).toHaveLength(24);
    expect(store.getSnapshot().some((item) => item.proposal.id === proposal.id && item.status === 'running')).toBe(true);
  });
  it('does not restore cleared logs from a delayed event', () => {
    const store = new ExecutionStore();
    store.receive(run(1, 'old'));
    store.receive(run(3, ''));
    store.receive(run(2, 'old plus pending'));
    expect(store.getSnapshot()[0].logs).toBe('');
    store.receive(run(4, 'new'));
    expect(store.getSnapshot()[0].logs).toBe('new');
  });
  it('rehydrates validated cards from successful tool traces only', () => {
    expect(proposalsFromTrace([{ toolName: 'propose_app_run', result: JSON.stringify([proposal]) }])).toEqual([proposal]);
    expect(proposalsFromTrace([{ toolName: 'propose_app_run', result: '{invalid' }])).toEqual([]);
    expect(proposalsFromTrace([{ toolName: 'propose_app_run', result: JSON.stringify([proposal]), isError: true }])).toEqual([]);
    expect(proposalsFromTrace([{ toolName: 'propose_app_run', result: JSON.stringify([{ command: 'rm -rf /' }]) }])).toEqual([]);
  });
  it('ignores queued updates after deletion and allows a newer explicit launch', () => {
    const store = new ExecutionStore();
    store.receive(run(1,'old'));
    store.receive({...run(5,''),status:'removed'});
    store.receive(run(4,'queued logs'));
    expect(store.getSnapshot()).toEqual([]);
    store.receive(run(6,'new launch'));
    expect(store.getSnapshot()[0].logs).toBe('new launch');
  });
});
