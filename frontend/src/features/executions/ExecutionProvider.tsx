import React, { createContext, useContext, useEffect, useMemo, useState, useSyncExternalStore } from 'react';
import { ClearAppRunLogs, ListAppRuns, PrepareAppRun, RemoveAppRun, StartAppRun, StopAppRun } from '../../../wailsjs/go/app/App';
import { domain } from '../../../wailsjs/go/models';
import { EventsOn } from '../../../wailsjs/runtime/runtime';
import { ExecutionStore } from './store';
import type { Execution, ExecutionFocus, RunProposal } from './types';

interface ExecutionContextValue {
  store: ExecutionStore;
  project: string;
  start: (proposal: RunProposal, source?: DOMRect) => Promise<void>;
  stop: (id: string) => Promise<void>;
  clear: (id: string) => Promise<void>;
  remove: (id: string) => Promise<void>;
  requestFix?: (run: Execution, prompt: string) => Promise<void>;
  chatBusy: boolean;
}

const ExecutionContext = createContext<ExecutionContextValue | null>(null);

export function ExecutionProvider({ project, onReveal, onRequestFix, chatBusy = false, children }: React.PropsWithChildren<{
  project: string;
  onReveal: (focus: ExecutionFocus) => void;
  onRequestFix?: (prompt: string) => Promise<void>;
  chatBusy?: boolean;
}>) {
  const [store] = useState(() => new ExecutionStore());
  useEffect(() => {
    let alive = true;
    const unsubscribe = EventsOn('execution:update', (run: Execution) => {
      if (alive) store.receive(run);
    });
    ListAppRuns().then((runs) => { if (alive) runs.forEach((run) => store.receive(run as Execution)); }).catch(() => {});
    return () => { alive = false; unsubscribe(); };
  }, [store]);

  const value = useMemo<ExecutionContextValue>(() => ({
    store, project, chatBusy,
    requestFix: onRequestFix ? async (run, prompt) => {
      if (run.proposal.project !== project) throw new Error('Activa el proyecto de esta ejecución antes de solicitar la solución.');
      if (chatBusy) throw new Error('Espera a que Merlin termine de responder antes de enviar la consulta.');
      await onRequestFix(prompt);
    } : undefined,
    start: async (proposal, source) => {
      // File edits change the evidence ID. Accept refreshed evidence only for
      // the same visible launch recipe; changed launchers still require review.
      const proposals = await PrepareAppRun(new domain.RunRequest(proposal.request));
      const current = proposals.find((item) => item.project === proposal.project
        && item.directory === proposal.directory && item.command === proposal.command
        && item.reason === proposal.reason && item.request.directory === proposal.request.directory
        && (item.request.entry || '') === (proposal.request.entry || ''));
      if (!current) {
        throw new Error('Cambió el proyecto. Solicita una nueva propuesta para revisar el comando actualizado.');
      }
      if (current.id !== proposal.id) store.receive(await RemoveAppRun(proposal.id) as Execution);
      onReveal({ id: current.id, command: current.command, source, nonce: performance.now() });
      store.receive(await StartAppRun(current.id) as Execution);
    },
    stop: async (id) => { await StopAppRun(id); },
    clear: async (id) => { store.receive(await ClearAppRunLogs(id) as Execution); },
    remove: async (id) => { store.receive(await RemoveAppRun(id) as Execution); },
  }), [store, project, onReveal, onRequestFix, chatBusy]);

  return <ExecutionContext.Provider value={value}>{children}</ExecutionContext.Provider>;
}

export function useExecutionActions() {
  const value = useContext(ExecutionContext);
  if (!value) throw new Error('ExecutionProvider no está disponible');
  return value;
}

export function useExecutions() {
  const { store } = useExecutionActions();
  return useSyncExternalStore(store.subscribe, store.getSnapshot, store.getSnapshot);
}
