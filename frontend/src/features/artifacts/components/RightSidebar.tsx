import React from 'react';
import { Artifact } from '../../../types';
import { IconFileCode, IconPanelRight } from '../../../components/Icons';
import { useResizablePanel } from '../../../hooks/useResizablePanel';
import { ArtifactItem } from './ArtifactItem';

export interface RightSidebarProps {
  isOpen: boolean;
  onToggle: () => void;
  width: number;
  onWidthChange: (width: number) => void;
  artifacts: Artifact[];
  selectedArtifactId?: string;
  onSelectArtifact?: (id: string) => void;
}

export const RightSidebar: React.FC<RightSidebarProps> = ({
  isOpen,
  onToggle,
  width,
  onWidthChange,
  artifacts,
  selectedArtifactId,
  onSelectArtifact,
}) => {
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
      {/* Botón para expandir cuando está colapsado */}
      {!isOpen ? (
        <button
          type="button"
          className="w-full h-full flex flex-col items-center justify-start pt-3 gap-3 text-content-dim hover:text-accent-primary transition-colors cursor-pointer"
          onClick={onToggle}
          title="Expandir panel de artefactos"
          aria-label="Expandir panel derecho"
        >
          <IconFileCode size={16} />
          <span className="[writing-mode:vertical-lr] text-[10px] tracking-widest font-mono text-content-dim font-semibold mt-1">
            ARTEFACTOS
          </span>
        </button>
      ) : (
        <div className="h-full flex flex-col w-full relative">
          {/* Resize Handle en el borde izquierdo */}
          <div
            className="absolute top-0 left-0 w-1.5 h-full cursor-col-resize hover:bg-accent-primary/60 active:bg-accent-primary transition-colors z-20 select-none"
            onMouseDown={handleMouseDown}
            title="Arrastrar para redimensionar panel (máximo 360px)"
          />

          {/* Encabezado del panel con botón de colapso */}
          <div className="flex items-center justify-between px-3.5 py-2.5 border-b border-border-subtle shrink-0">
            <button
              type="button"
              className="p-1 text-content-dim hover:text-content-headline hover:bg-[#192226] rounded transition-colors cursor-pointer"
              onClick={onToggle}
              title="Colapsar panel"
              aria-label="Colapsar panel derecho"
            >
              <IconPanelRight size={15} />
            </button>
            <div className="flex items-center gap-2">
              <span className="text-[11px] font-semibold tracking-wider text-content-dim font-mono">
                ARTEFACTOS
              </span>
              <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-[#1c282e] text-accent-primary font-medium">
                {artifacts.length}
              </span>
            </div>
          </div>

          <div className="flex-1 overflow-y-auto px-2.5 py-2">
            {artifacts.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 px-3 text-center">
                <div className="p-3 rounded-full bg-[#141b1f] text-content-dim mb-3 border border-border-subtle">
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
        </div>
      )}
    </aside>
  );
};
