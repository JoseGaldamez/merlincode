import React from 'react';
import { Artifact } from '../types';
import { IconFileCode, IconPanelRight, IconCopy, IconCheck } from './Icons';

interface RightSidebarProps {
  isOpen: boolean;
  onToggle: () => void;
  artifacts: Artifact[];
  selectedArtifactId?: string;
  onSelectArtifact?: (id: string) => void;
}

export const RightSidebar: React.FC<RightSidebarProps> = ({
  isOpen,
  onToggle,
  artifacts,
  selectedArtifactId,
  onSelectArtifact,
}) => {
  const [copiedId, setCopiedId] = React.useState<string | null>(null);

  const handleCopy = (e: React.MouseEvent, artifact: Artifact) => {
    e.stopPropagation();
    if (!artifact.content) return;
    navigator.clipboard.writeText(artifact.content);
    setCopiedId(artifact.id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  return (
    <aside
      className={`h-full bg-surface border-l border-border-subtle flex flex-col select-none shrink-0 transition-all duration-200 overflow-hidden ${
        isOpen ? 'w-72' : 'w-11'
      }`}
    >
      {/* Botón en el mismo panel para expandir cuando está colapsado */}
      {!isOpen ? (
        <button
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
        <div className="h-full flex flex-col">
          {/* Encabezado del panel con botón de colapso */}
          <div className="flex items-center justify-between px-3.5 py-2.5 border-b border-border-subtle shrink-0">
            <button
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
                  {artifacts.map((art) => {
                    const isSelected = art.id === selectedArtifactId;
                    return (
                      <div
                        key={art.id}
                        className={`group p-2.5 rounded bg-[#13191d] border transition-all cursor-pointer ${
                          isSelected
                            ? 'border-accent-primary bg-[#162329]'
                            : 'border-border-subtle hover:border-border-petrol'
                        }`}
                        onClick={() => onSelectArtifact?.(art.id)}
                      >
                        <div className="flex items-center justify-between gap-2 mb-1.5">
                          <div className="flex items-center gap-2 min-w-0 flex-1">
                            <IconFileCode
                              size={14}
                              className={`shrink-0 ${
                                isSelected ? 'text-accent-primary' : 'text-content-dim'
                              }`}
                            />
                            <span
                              className="text-xs font-mono text-content-body truncate"
                              title={art.title}
                            >
                              {art.title}
                            </span>
                          </div>
                          {art.content && (
                            <button
                              className="p-1 text-content-dim hover:text-accent-primary hover:bg-[#1b252a] rounded transition-colors cursor-pointer"
                              onClick={(e) => handleCopy(e, art)}
                              title="Copiar contenido"
                            >
                              {copiedId === art.id ? (
                                <IconCheck size={12} className="text-status-success" />
                              ) : (
                                <IconCopy size={12} />
                              )}
                            </button>
                          )}
                        </div>

                        <div className="flex items-center gap-2 text-[10px] font-mono text-content-dim">
                          {art.language && (
                            <span className="px-1.5 py-0.5 rounded bg-[#1a2327] text-content-muted font-medium">
                              {art.language.toUpperCase()}
                            </span>
                          )}
                          {art.size && <span>{art.size}</span>}
                          <span className="ml-auto text-content-dim">{art.timestamp}</span>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </aside>
  );
};
