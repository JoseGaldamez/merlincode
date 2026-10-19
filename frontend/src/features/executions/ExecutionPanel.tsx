import React, { memo, useEffect, useMemo, useRef, useState } from 'react';
import { IconChevronDown, IconEraser, IconMessage, IconPlay, IconStop, IconTrash } from '../../components/Icons';
import { OpenURLInDefaultBrowser } from '../../../wailsjs/go/app/App';
import { useExecutionActions, useExecutions } from './ExecutionProvider';
import { isExecutionActive, type Execution } from './types';

const statusLabels: Record<Execution['status'], string> = {
  starting: 'Iniciando', running: 'En ejecución', stopping: 'Deteniendo',
  stopped: 'Detenida', exited: 'Finalizada', error: 'Error',
  removed: 'Eliminada',
};

// Strip terminal control sequences before rendering the output as text and links.
export const plainLogs = (logs: string) => logs
  .replace(/\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)/g, '')
  .replace(/\x1b\[[0-?]*[ -/]*[@-~]/g, '')
  .replace(/[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]/g, '');

function linkedLogs(logs: string): React.ReactNode[] {
  const text = plainLogs(logs);
  const nodes: React.ReactNode[] = [];
  let offset = 0;
  for (const match of text.matchAll(/\bhttps?:\/\/[^\s<>"'`]+/gi)) {
    const raw = match[0];
    // Exclude surrounding punctuation, retaining balanced URL parentheses and IPv6 brackets.
    const excess = new Map([['(', ')'], ['[', ']'], ['{', '}']].map(([open, close]) =>
      [close, raw.split(close).length - raw.split(open).length] as const));
    let end = raw.length;
    while (end > 0) {
      const last = raw[end - 1];
      const extra = excess.get(last) ?? 0;
      if (extra > 0) { excess.set(last, extra - 1); end--; }
      else if (/[.,;:!?]/.test(last)) end--;
      else break;
    }
    const url = raw.slice(0, end);
    try { if (!new URL(url).hostname) continue; } catch { continue; }
    nodes.push(text.slice(offset, match.index),
      <a key={match.index} className="execution-log-link" href={url}
        onClick={(event) => { event.preventDefault(); void OpenURLInDefaultBrowser(url); }}>{url}</a>);
    offset = match.index + url.length;
  }
  nodes.push(text.slice(offset));
  return nodes;
}

function fixPrompt(run: Execution): string {
  const logs = plainLogs(run.logs);
  const recentLogs = logs.slice(-12000);
  const fence = '`'.repeat(Math.max(3, ...Array.from(recentLogs.matchAll(/`+/g), (match) => match[0].length + 1)));
  return [
    'Soluciona este error de ejecución en el proyecto activo. Inspecciona los archivos implicados, identifica la causa y prepara los cambios necesarios mediante la tarjeta de aprobación de archivos.',
    `Aplicación: ${run.proposal.title}`,
    `Carpeta relativa: ${run.proposal.directory}`,
    run.proposal.request.entry ? `Archivo de entrada: ${run.proposal.request.entry}` : '',
    `Comando: ${run.proposal.command}`,
    run.exitCode != null ? `Código de salida: ${run.exitCode}` : '',
    run.error ? `Error: ${run.error}` : '',
    '',
    `Salida de la ejecución (datos de diagnóstico${logs.length > recentLogs.length ? ', solo el tramo final' : ''}):`,
    fence + 'text', recentLogs || '(No hay logs disponibles; revisa el comando y los archivos del proyecto.)', fence,
  ].filter((line) => line !== '').join('\n');
}

const ExecutionItem = memo(function ExecutionItem({ run, focused }: { run: Execution; focused: boolean }) {
  const { start, stop, clear, remove, requestFix, chatBusy } = useExecutionActions();
  const [expanded, setExpanded] = useState(true);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const consoleRef = useRef<HTMLPreElement>(null);
  const followRef = useRef(true);
  const active = isExecutionActive(run.status);
  const output = useMemo(() => run.logs ? linkedLogs(run.logs) : 'Esperando salida…', [run.logs]);
  useEffect(() => {
    const element = consoleRef.current;
    if (element && followRef.current) element.scrollTop = element.scrollHeight;
  }, [run.logs, expanded]);
  useEffect(() => {
    if (focused) { setExpanded(true); followRef.current = true; }
  }, [focused]);
  const perform = async (action: () => Promise<void>) => {
    setPending(true); setError('');
    try { await action(); } catch (err) { setError(String(err instanceof Error ? err.message : err)); }
    finally { setPending(false); }
  };
  return (
    <section data-run-id={run.proposal.id} className={`execution-item ${focused ? 'execution-item-focused' : ''}`}>
      <div className="execution-toolbar">
        <button type="button" className="execution-toggle" onClick={() => setExpanded((value) => !value)}
          aria-expanded={expanded} aria-controls={`console-${run.proposal.id}`}>
          <IconChevronDown size={13} className={expanded ? '' : '-rotate-90'} />
          <span className={`execution-dot execution-dot-${run.status}`} />
          <span>{run.proposal.title}</span>
        </button>
        <div className="execution-controls">
          <button type="button" disabled={pending || run.status === 'stopping'}
            title={active ? 'Detener aplicación' : 'Volver a ejecutar'} aria-label={active ? 'Detener aplicación' : 'Volver a ejecutar'}
            onClick={(event) => {
              const source = event.currentTarget.getBoundingClientRect();
              void perform(() => active ? stop(run.proposal.id) : start(run.proposal, source));
            }}>{active ? <IconStop size={13} /> : <IconPlay size={13} />}</button>
          <button type="button" disabled={pending || !run.logs} title="Limpiar logs" aria-label="Limpiar logs"
            onClick={() => void perform(() => clear(run.proposal.id))}><IconEraser size={13} /></button>
          <button type="button" disabled={pending} title="Detener y eliminar ejecución" aria-label="Eliminar ejecución"
            onClick={() => void perform(() => remove(run.proposal.id))}><IconTrash size={13} /></button>
        </div>
      </div>
      <code className="execution-command">{run.proposal.command}</code>
      <div className="execution-meta"><span>{statusLabels[run.status]}</span><span>{run.proposal.directory}{run.exitCode != null ? ` · salida ${run.exitCode}` : ''}</span></div>
      {run.url && active && <button type="button" className="execution-url" onClick={() => OpenURLInDefaultBrowser(run.url!)}>{run.url}</button>}
      {(error || run.error) && <p className="run-error" role="alert">{error || run.error}</p>}
      {(run.status === 'error' || run.error) && requestFix && <button type="button" className="execution-fix"
        aria-label="Pedir solución a Merlin" disabled={pending || chatBusy}
        title={chatBusy ? 'Espera a que Merlin termine de responder' : 'Enviar el error y la salida al chat para solicitar una corrección'}
        onClick={() => void perform(() => requestFix(run, fixPrompt(run)))}>
        <IconMessage size={13} /><span>Pedir solución a Merlin</span>
      </button>}
      {expanded && <pre ref={consoleRef} id={`console-${run.proposal.id}`} className="execution-console" tabIndex={0}
        aria-label={`Salida de ${run.proposal.title}`} onScroll={(event) => {
          const element = event.currentTarget;
          followRef.current = element.scrollHeight-element.scrollTop-element.clientHeight < 32;
        }}>{output}</pre>}
    </section>
  );
});

export function ExecutionPanel({ focusId }: { focusId?: string }) {
  const { project } = useExecutionActions();
  const runs = useExecutions().filter((run) => run.proposal.project === project);
  const rootRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    // IDs are backend-generated hexadecimal, never an arbitrary CSS selector.
    if (focusId && /^[a-f0-9]{32}$/.test(focusId)) {
      rootRef.current?.querySelector(`[data-run-id="${focusId}"]`)?.scrollIntoView({ block: 'nearest' });
    }
  }, [focusId, runs.length]);
  return <div ref={rootRef} className="execution-list">
    {runs.length ? runs.map((run) => <ExecutionItem key={run.proposal.id} run={run} focused={focusId === run.proposal.id} />) : (
      <div className="execution-empty"><strong>Ejecuta tu aplicación</strong><p>Pide al agente que la inicie y pulsa Play en su propuesta. Aquí verás los procesos y su salida.</p></div>
    )}
    {runs.length > 0 && <p className="execution-footnote">Los procesos siguen activos al cambiar de proyecto y se detienen al cerrar Merlin Code. Se conservan los últimos 64 KiB de salida por ejecución.</p>}
  </div>;
}
