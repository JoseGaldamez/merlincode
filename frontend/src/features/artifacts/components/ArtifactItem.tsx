import React from 'react';
import { Artifact } from '../../../types';
import { IconFileCode, IconCopy, IconCheck } from '../../../components/Icons';
import { useClipboard } from '../../../hooks/useClipboard';

interface ArtifactItemProps {
  artifact: Artifact;
  isSelected: boolean;
  onSelect?: (id: string) => void;
}

export const ArtifactItem: React.FC<ArtifactItemProps> = ({
  artifact,
  isSelected,
  onSelect,
}) => {
  const { copy, isCopied } = useClipboard();

  const handleCopy = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (!artifact.content) return;
    copy(artifact.id, artifact.content);
  };

  return (
    <div
      className={`group p-2.5 rounded bg-[#13191d] border transition-all cursor-pointer ${
        isSelected
          ? 'border-accent-primary bg-[#162329]'
          : 'border-border-subtle hover:border-border-petrol'
      }`}
      onClick={() => onSelect?.(artifact.id)}
    >
      <div className="flex items-center justify-between gap-2 mb-1.5">
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <IconFileCode
            size={14}
            className={`shrink-0 ${isSelected ? 'text-accent-primary' : 'text-content-dim'}`}
          />
          <span className="text-xs font-mono text-content-body truncate" title={artifact.title}>
            {artifact.title}
          </span>
        </div>

        {artifact.content && (
          <button
            type="button"
            className="p-1 text-content-dim hover:text-accent-primary hover:bg-[#1b252a] rounded transition-colors cursor-pointer"
            onClick={handleCopy}
            title="Copiar contenido"
          >
            {isCopied(artifact.id) ? (
              <IconCheck size={12} className="text-status-success" />
            ) : (
              <IconCopy size={12} />
            )}
          </button>
        )}
      </div>

      <div className="flex items-center gap-2 text-[10px] font-mono text-content-dim">
        {artifact.language && (
          <span className="px-1.5 py-0.5 rounded bg-[#1a2327] text-content-muted font-medium">
            {artifact.language.toUpperCase()}
          </span>
        )}
        {artifact.size && <span>{artifact.size}</span>}
        <span className="ml-auto text-content-dim">{artifact.timestamp}</span>
      </div>
    </div>
  );
};
