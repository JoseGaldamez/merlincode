import React, { useEffect, useId, useRef, useState } from 'react';
import { IconChevronDown, IconPlay } from '../../components/Icons';
import type { Project } from '../../types';
import { useExecutionActions, useExecutions } from './ExecutionProvider';
import { isExecutionActive, type Execution, type ExecutionFocus } from './types';

export function RunningProjectsIndicator({ projects, onSelect }: {
  projects: Project[];
  onSelect: (path: string, focus: ExecutionFocus) => Promise<void>;
}) {
  const runs = useExecutions();
  const { project } = useExecutionActions();
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState<string>();
  const [error, setError] = useState('');
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const popupId = useId();
  const groups = new Map<string, Execution[]>();
  for (const run of runs) {
    if (!isExecutionActive(run.status)) continue;
    const path = run.proposal.project;
    const group = groups.get(path) || [];
    group.push(run);
    groups.set(path, group);
  }
  const total = [...groups.values()].reduce((sum, group) => sum + group.length, 0);
  const hasActiveRuns = total > 0;
  const entries = [...groups.entries()].sort(([a], [b]) => a.localeCompare(b));

  useEffect(() => {
    if (!hasActiveRuns) { setOpen(false); return; }
    if (!open) return;
    root.current?.querySelector<HTMLButtonElement>('.running-project-row')?.focus();
    const outside = (event: MouseEvent) => {
      if (!root.current?.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', outside);
    return () => document.removeEventListener('mousedown', outside);
  }, [open, hasActiveRuns]);

  const select = async (path: string, run: Execution) => {
    setPending(path); setError('');
    try {
      await onSelect(path, { id: run.proposal.id, command: run.proposal.command, nonce: performance.now() });
      setOpen(false);
      trigger.current?.focus();
    } catch {
      setError('No se pudo abrir el proyecto. Comprueba que su carpeta esté disponible.');
    } finally { setPending(undefined); }
  };

  if (!hasActiveRuns) return null;

  return <div className="running-projects wails-no-drag" ref={root}
    onBlur={(event) => { if (!pending && !event.currentTarget.contains(event.relatedTarget as Node | null)) setOpen(false); }}
    onKeyDown={(event) => {
      if (event.key === 'Escape') { event.stopPropagation(); setOpen(false); trigger.current?.focus(); }
    }}>
    <button ref={trigger} type="button" className={`running-projects-trigger ${total ? 'has-active-runs' : ''}`}
      aria-label={`Ejecuciones activas: ${total}`} aria-expanded={open} aria-controls={popupId}
      title="Ver ejecuciones por proyecto" onClick={() => { setOpen((value) => !value); setError(''); }}>
      <IconPlay size={13} /><span>{total}</span><IconChevronDown size={11} />
    </button>
    {open && <div id={popupId} className="running-projects-popup" role="region" aria-label="Ejecuciones por proyecto" aria-busy={!!pending}>
      <div className="running-projects-heading">Ejecuciones por proyecto</div>
      {entries.length ? entries.map(([path, group]) => {
        const name = projects.find((item) => item.path === path)?.name || path.split(/[\\/]/).filter(Boolean).at(-1) || path;
        const first = [...group].sort((a, b) => a.proposal.id.localeCompare(b.proposal.id))[0];
        return <button key={path} type="button" className="running-project-row" disabled={!!pending}
          aria-label={`${name}: ${group.length} en curso`} aria-current={path === project ? 'true' : undefined}
          title={path} onClick={() => void select(path, first)}>
          <span className="running-project-info"><strong>{name}</strong><span>{pending === path ? 'Abriendo…' : path}</span></span>
          {path === project && <span className="running-project-current">Actual</span>}
          <span className="running-project-count">{group.length}</span>
        </button>;
      }) : <p className="running-projects-empty">No hay ejecuciones activas.</p>}
      {error && <p className="run-error" role="alert">{error}</p>}
    </div>}
  </div>;
}
