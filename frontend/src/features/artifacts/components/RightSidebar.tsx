import React, { useCallback, useEffect, useId, useRef, useState } from 'react';
import { Artifact } from '../../../types';
import { IconChevronDown, IconChevronUp, IconFileCode, IconPanelRight, IconPlay } from '../../../components/Icons';
import { useResizablePanel } from '../../../hooks/useResizablePanel';
import { ArtifactItem } from './ArtifactItem';
import { ExecutionPanel } from '../../executions/ExecutionPanel';
import type { ExecutionFocus } from '../../executions/types';

const HEIGHT_KEY = 'merlin_executions_panel_height';
const COLLAPSED_KEY = 'merlin_executions_panel_collapsed';
const MIN_SECTION_HEIGHT = 120;

function initialHeight() {
  try {
    const saved = Number(localStorage.getItem(HEIGHT_KEY));
    if (Number.isFinite(saved) && saved >= MIN_SECTION_HEIGHT) return saved;
  } catch {}
  return 360;
}

function savePreference(key: string, value: string) {
  try { localStorage.setItem(key, value); } catch {}
}

export interface RightSidebarProps {
  isOpen: boolean;
  onToggle: () => void;
  width: number;
  onWidthChange: (width: number) => void;
  artifacts: Artifact[];
  selectedArtifactId?: string;
  onSelectArtifact?: (id: string) => void;
  executionFocus?: ExecutionFocus;
}

export const RightSidebar: React.FC<RightSidebarProps> = ({
  isOpen,
  onToggle,
  width,
  onWidthChange,
  artifacts,
  selectedArtifactId,
  onSelectArtifact,
  executionFocus,
}) => {
  const [executionHeight, setExecutionHeight] = useState(initialHeight);
  const [executionsCollapsed, setExecutionsCollapsed] = useState(() => {
    try { return localStorage.getItem(COLLAPSED_KEY) === 'true'; } catch { return false; }
  });
  const containerRef = useRef<HTMLDivElement>(null);
  const executionsRef = useRef<HTMLElement>(null);
  const resizeCleanup = useRef<() => void>();
  const executionsId = useId();
  const maxHeight = Math.max(MIN_SECTION_HEIGHT, (containerRef.current?.clientHeight || window.innerHeight) - MIN_SECTION_HEIGHT);
  const clampHeight = useCallback((height: number) => Math.max(MIN_SECTION_HEIGHT,
    Math.min((containerRef.current?.clientHeight || window.innerHeight) - MIN_SECTION_HEIGHT, height)), []);

  useEffect(() => {
    if (executionFocus) {
      setExecutionsCollapsed(false);
      savePreference(COLLAPSED_KEY, 'false');
    }
  }, [executionFocus]);

  useEffect(() => {
    if (!isOpen || executionsCollapsed) return;
    const resize = () => setExecutionHeight((height) => clampHeight(height));
    resize();
    window.addEventListener('resize', resize);
    return () => { window.removeEventListener('resize', resize); resizeCleanup.current?.(); };
  }, [isOpen, executionsCollapsed, clampHeight]);

  const resizeExecutions = (event: React.MouseEvent) => {
    if (event.button !== 0) return;
    event.preventDefault();
    resizeCleanup.current?.();
    const startY = event.clientY;
    const startHeight = executionsRef.current?.getBoundingClientRect().height || executionHeight;
    const cursor = document.body.style.cursor;
    const userSelect = document.body.style.userSelect;
    let height = startHeight;
    document.body.style.cursor = 'row-resize';
    document.body.style.userSelect = 'none';
    const move = (moveEvent: MouseEvent) => {
      height = clampHeight(startHeight + startY - moveEvent.clientY);
      setExecutionHeight(height);
    };
    const finish = () => {
      savePreference(HEIGHT_KEY, String(height));
      document.body.style.cursor = cursor;
      document.body.style.userSelect = userSelect;
      window.removeEventListener('mousemove', move);
      window.removeEventListener('mouseup', finish);
      window.removeEventListener('blur', finish);
      resizeCleanup.current = undefined;
    };
    resizeCleanup.current = finish;
    window.addEventListener('mousemove', move);
    window.addEventListener('mouseup', finish);
    window.addEventListener('blur', finish);
  };
  const { handleMouseDown } = useResizablePanel({
    minWidth: 180,
    maxWidth: 360,
    side: 'right',
    onWidthChange,
  });

  return (
    <aside
      className={`relative h-full bg-surface border-l border-border-subtle flex flex-col select-none shrink-0 overflow-hidden ${
        !isOpen ? 'w-11 transition-all duration-200' : ''
      }`}
      style={{
        width: isOpen ? `${width}px` : undefined,
        minWidth: isOpen ? `${width}px` : undefined,
        maxWidth: isOpen ? '360px' : undefined,
      }}
    >
      {/* El margen exterior deja libre el borde de redimensionado de Wails. */}
      {!isOpen ? (
        <button
          type="button"
          className="mx-1.5 h-full flex flex-col items-center justify-start pt-3 gap-3 text-content-dim hover:text-accent-primary transition-colors cursor-pointer"
          onClick={onToggle}
          title="Expandir panel de artefactos y ejecuciones"
          aria-label="Expandir panel derecho"
        >
          <IconFileCode size={16} />
          <span className="[writing-mode:vertical-lr] text-[10px] tracking-widest font-mono text-content-dim font-semibold mt-1">
            Artefactos | Ejecuciones
          </span>
        </button>
      ) : (
        <div ref={containerRef} className="h-full min-h-0 flex flex-col w-full relative">
          {/* Resize Handle en el borde izquierdo */}
          <div
            className="absolute top-0 left-0 w-1.5 h-full cursor-col-resize hover:bg-accent-primary/60 active:bg-accent-primary transition-colors z-20 select-none"
            onMouseDown={handleMouseDown}
            title="Arrastrar para redimensionar panel (máximo 360px)"
          />

          <section className="flex-1 min-h-[120px] flex flex-col overflow-hidden" aria-label="Artefactos">
          <div className="flex items-center justify-between px-3.5 py-2.5 border-b border-border-subtle shrink-0">
            <div className="flex items-center gap-2">
              <span className="text-[11px] font-semibold tracking-wider text-content-dim font-mono">ARTEFACTOS</span>
              <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-surface-selected text-accent-primary font-medium">{artifacts.length}</span>
            </div>
            <button
              type="button"
              className="p-1 text-content-dim hover:text-content-headline hover:bg-surface-hover rounded transition-colors cursor-pointer"
              onClick={onToggle}
              title="Colapsar panel"
              aria-label="Colapsar panel derecho"
            >
              <IconPanelRight size={15} />
            </button>
          </div>

          <div className="flex-1 min-h-0 overflow-y-auto px-2.5 py-2">
            {artifacts.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 px-3 text-center">
                <div className="p-3 rounded-full bg-surface-card text-content-dim mb-3 border border-border-subtle">
                  <IconFileCode size={20} />
                </div>
                <span className="text-xs font-semibold text-content-headline mb-1">
                  Sin artefactos aún
                </span>
                <span className="text-[11px] text-content-dim leading-relaxed max-w-[200px]">
                  Los archivos, esquemas y códigos generados por el agente aparecerán aquí.
                </span>
              </div>
            ) : (
              <div className="flex flex-col">
                <div className="px-1 py-1.5 text-[10px] font-mono font-semibold tracking-wider text-content-dim">
                  ARCHIVOS GENERADOS
                </div>
                <div className="flex flex-col gap-2 mt-1">
                  {artifacts.map((art) => (
                    <ArtifactItem
                      key={art.id}
                      artifact={art}
                      isSelected={art.id === selectedArtifactId}
                      onSelect={onSelectArtifact}
                    />
                  ))}
                </div>
              </div>
            )}
          </div>
          </section>

          <section ref={executionsRef} className="relative flex flex-col shrink-0 overflow-hidden border-t border-border-subtle bg-surface-lowest"
            aria-label="Ejecuciones" data-execution-destination
            style={executionsCollapsed ? undefined : { height: `${executionHeight}px`, minHeight: MIN_SECTION_HEIGHT, maxHeight: `calc(100% - ${MIN_SECTION_HEIGHT}px)` }}>
            {!executionsCollapsed && <div role="separator" tabIndex={0} aria-label="Ajustar altura de Ejecuciones"
              aria-orientation="horizontal" aria-controls={executionsId} aria-valuemin={MIN_SECTION_HEIGHT}
              aria-valuemax={maxHeight} aria-valuenow={executionHeight}
              className="absolute top-0 left-0 right-0 h-1.5 z-20 cursor-row-resize hover:bg-accent-primary/60 focus-visible:bg-accent-primary focus-visible:outline-none transition-colors"
              title="Arrastrar para ajustar la altura de Ejecuciones" onMouseDown={resizeExecutions}
              onKeyDown={(event) => {
                const heights: Record<string, number> = { ArrowUp: executionHeight + 24, ArrowDown: executionHeight - 24, Home: MIN_SECTION_HEIGHT, End: maxHeight };
                if (!(event.key in heights)) return;
                event.preventDefault();
                const height = clampHeight(heights[event.key]);
                setExecutionHeight(height);
                savePreference(HEIGHT_KEY, String(height));
              }} />}
            <button type="button" className="flex items-center gap-1.5 px-3 py-2 w-full min-w-0 bg-surface-neutral border-b border-border-subtle hover:bg-surface-card transition-colors cursor-pointer shrink-0 focus-visible:outline-2 focus-visible:outline-accent-primary focus-visible:-outline-offset-2"
              aria-label={executionsCollapsed ? 'Desplegar Ejecuciones' : 'Colapsar Ejecuciones'} aria-expanded={!executionsCollapsed} aria-controls={executionsId}
              onClick={() => {
                const next = !executionsCollapsed;
                setExecutionsCollapsed(next);
                savePreference(COLLAPSED_KEY, String(next));
              }}>
              <IconPlay size={13} className="text-accent-primary shrink-0" />
              <span className={`flex-1 text-left text-[11px] font-semibold tracking-wider font-mono truncate ${executionsCollapsed ? 'text-content-dim' : 'text-accent-primary'}`}>EJECUCIONES</span>
              <span className="p-1 text-content-dim shrink-0">{executionsCollapsed ? <IconChevronUp size={14} /> : <IconChevronDown size={14} />}</span>
            </button>
            <div id={executionsId} hidden={executionsCollapsed} className="flex-1 min-h-0 overflow-y-auto px-1.5 py-1.5">
              {!executionsCollapsed && <ExecutionPanel focusId={executionFocus?.id} />}
            </div>
          </section>
        </div>
      )}
    </aside>
  );
};
