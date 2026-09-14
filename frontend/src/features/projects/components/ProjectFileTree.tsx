import React, { useState, useCallback } from 'react';
import { domain } from '../../../../wailsjs/go/models';
import { OpenDirectoryInExplorer } from '../../../../wailsjs/go/app/App';
import {
  IconFolder,
  IconFolderOpen,
  IconChevronDown,
  IconChevronUp,
  IconRefresh,
} from '../../../components/Icons';
import { useProjectTree } from '../hooks/useProjectTree';
import { FileTreeNode } from './FileTreeNode';

export interface ProjectFileTreeProps {
  projectPath?: string;
  projectName?: string;
  onOpenFolder?: () => void;
}

const STORAGE_KEY_TREE_HEIGHT = 'merlin_project_tree_height';
const STORAGE_KEY_TREE_COLLAPSED = 'merlin_project_tree_collapsed';

function getInitialTreeHeight(): number {
  try {
    const saved = localStorage.getItem(STORAGE_KEY_TREE_HEIGHT);
    if (saved !== null) {
      const val = parseInt(saved, 10);
      if (!isNaN(val) && val >= 360) {
        return val;
      }
    }
  } catch {}
  return 360;
}

function getInitialTreeCollapsed(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY_TREE_COLLAPSED) === 'true';
  } catch {}
  return false;
}

export const ProjectFileTree: React.FC<ProjectFileTreeProps> = ({
  projectPath,
  projectName,
  onOpenFolder,
}) => {
  const {
    nodes,
    loading,
    selectedPath,
    setSelectedPath,
    toggleDirectory,
    isExpanded,
    refreshTree,
  } = useProjectTree(projectPath);

  const [height, setHeight] = useState<number>(() => getInitialTreeHeight());
  const [isCollapsed, setIsCollapsed] = useState<boolean>(() => getInitialTreeCollapsed());

  const toggleCollapse = useCallback((e?: React.MouseEvent) => {
    if (e) e.stopPropagation();
    setIsCollapsed((prev) => {
      const next = !prev;
      try {
        localStorage.setItem(STORAGE_KEY_TREE_COLLAPSED, String(next));
      } catch {}
      return next;
    });
  }, []);

  const handleMouseDownResize = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    document.body.style.cursor = 'row-resize';
    document.body.style.userSelect = 'none';

    const startY = e.clientY;
    const startHeight = height;

    const onMouseMove = (moveEvent: MouseEvent) => {
      const delta = startY - moveEvent.clientY;
      const maxHeight = Math.max(360, Math.floor(window.innerHeight * 0.5));
      const minHeight = 360;
      const effectiveMax = maxHeight;
      const effectiveMin = Math.min(minHeight, effectiveMax);
      const newHeight = Math.max(effectiveMin, Math.min(effectiveMax, startHeight + delta));
      setHeight(newHeight);
      try {
        localStorage.setItem(STORAGE_KEY_TREE_HEIGHT, String(newHeight));
      } catch {}
    };

    const onMouseUp = () => {
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
    };

    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);
  }, [height]);

  const handleOpenSystemExplorer = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (projectPath) {
      OpenDirectoryInExplorer(projectPath).catch(() => {});
    }
  };

  const handleDoubleClickNode = (fullPath: string) => {
    if (fullPath) {
      OpenDirectoryInExplorer(fullPath).catch(() => {});
    }
  };

  const renderRecursiveNode = (node: domain.FileNode, depth: number): React.ReactNode => (
    <FileTreeNode
      key={node.relPath}
      node={node}
      depth={depth}
      isSelected={selectedPath === node.relPath}
      isExpanded={isExpanded(node.relPath)}
      onSelect={setSelectedPath}
      onToggleExpand={toggleDirectory}
      onDoubleClick={handleDoubleClickNode}
      renderChild={renderRecursiveNode}
    />
  );

  if (!projectPath) {
    return (
      <div className="flex flex-col p-3 text-center items-center justify-center gap-2 bg-[#0c1012] border-t border-border-subtle shrink-0">
        <span className="text-[11px] font-mono text-content-dim">
          Sin carpeta vinculada
        </span>
        {onOpenFolder && (
          <button
            type="button"
            onClick={onOpenFolder}
            className="flex items-center gap-1.5 px-2.5 py-1 bg-[#152329] hover:bg-[#1a2d35] text-accent-primary text-[11px] font-mono font-medium rounded border border-border-petrol transition-colors cursor-pointer"
          >
            <IconFolder size={12} />
            <span>Abrir carpeta</span>
          </button>
        )}
      </div>
    );
  }

  // Si está colapsado, se muestra únicamente la cabecera
  if (isCollapsed) {
    return (
      <div className="flex flex-col border-t border-border-subtle bg-[#0f1416] shrink-0 select-none">
        <div
          className="flex items-center justify-between px-3 py-2 cursor-pointer hover:bg-[#131a1e] transition-colors"
          onClick={() => toggleCollapse()}
          title="Hacer clic para desplegar panel de archivos"
        >
          <div className="flex items-center gap-1.5 min-w-0 flex-1 pr-2">
            <IconFolder size={13} className="text-accent-primary shrink-0" />
            <span
              className="text-[11px] font-mono font-semibold tracking-wider text-content-dim truncate"
              title={projectPath}
            >
              {projectName || 'PROYECTO'}
            </span>
          </div>

          <div className="flex items-center gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
            <button
              type="button"
              className="p-1 text-content-dim hover:text-accent-primary hover:bg-[#152329] rounded transition-colors cursor-pointer"
              onClick={handleOpenSystemExplorer}
              title="Abrir ubicación en el explorador de archivos del sistema"
              aria-label="Abrir en explorador del sistema"
            >
              <IconFolderOpen size={13} />
            </button>

            <button
              type="button"
              className="p-1 text-content-dim hover:text-accent-primary hover:bg-[#152329] rounded transition-colors cursor-pointer"
              onClick={toggleCollapse}
              title="Desplegar panel de archivos"
              aria-label="Desplegar panel"
            >
              <IconChevronUp size={14} />
            </button>
          </div>
        </div>
      </div>
    );
  }

  const windowH = typeof window !== 'undefined' ? window.innerHeight : 800;
  const maxAllowedHeight = Math.max(360, Math.floor(windowH * 0.5));

  return (
    <div
      className="relative flex flex-col border-t border-border-subtle bg-[#0c1012] shrink-0 overflow-hidden"
      style={{
        height: `${height}px`,
        minHeight: '360px',
        maxHeight: `${maxAllowedHeight}px`,
      }}
    >
      {/* Tirador de redimensionamiento vertical */}
      <div
        className="absolute top-0 left-0 right-0 h-1.5 cursor-row-resize hover:bg-accent-primary/60 active:bg-accent-primary transition-colors z-20 select-none"
        onMouseDown={handleMouseDownResize}
        title="Arrastrar para redimensionar panel (mínimo 360px, máximo mitad de la pantalla)"
      />

      {/* Subpanel Header */}
      <div
        className="flex items-center justify-between px-3 py-2 bg-[#0f1416] border-b border-border-subtle shrink-0 select-none cursor-pointer"
        onClick={() => toggleCollapse()}
      >
        <div className="flex items-center gap-1.5 min-w-0 flex-1 pr-2">
          <IconFolder size={13} className="text-accent-primary shrink-0" />
          <span
            className="text-[11px] font-mono font-semibold tracking-wider text-accent-primary truncate"
            title={projectPath}
          >
            {projectName || 'PROYECTO'}
          </span>
        </div>

        <div className="flex items-center gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
          <button
            type="button"
            className="p-1 text-content-dim hover:text-accent-primary hover:bg-[#152329] rounded transition-colors cursor-pointer"
            onClick={(e) => {
              e.stopPropagation();
              refreshTree();
            }}
            title="Recargar árbol de archivos"
            aria-label="Recargar archivos"
          >
            <IconRefresh size={12} className={loading ? 'animate-spin' : ''} />
          </button>

          <button
            type="button"
            className="p-1 text-content-dim hover:text-accent-primary hover:bg-[#152329] rounded transition-colors cursor-pointer"
            onClick={handleOpenSystemExplorer}
            title="Abrir ubicación en el explorador de archivos del sistema"
            aria-label="Abrir en explorador del sistema"
          >
            <IconFolderOpen size={13} />
          </button>

          <button
            type="button"
            className="p-1 text-content-dim hover:text-accent-primary hover:bg-[#152329] rounded transition-colors cursor-pointer"
            onClick={toggleCollapse}
            title="Ocultar panel (bajar todo)"
            aria-label="Ocultar panel"
          >
            <IconChevronDown size={14} />
          </button>
        </div>
      </div>

      {/* Subpanel Content / Tree List */}
      <div className="flex-1 overflow-y-auto px-1.5 py-1.5 flex flex-col min-h-0">
        {loading && nodes.length === 0 ? (
          <div className="flex items-center justify-center p-4 text-[11px] font-mono text-content-dim">
            <span>Cargando archivos...</span>
          </div>
        ) : nodes.length === 0 ? (
          <div className="flex flex-col items-center justify-center p-4 text-[11px] font-mono text-content-dim text-center">
            <span>Carpeta vacía o sin archivos visibles</span>
          </div>
        ) : (
          nodes.map((node) => renderRecursiveNode(node, 0))
        )}
      </div>
    </div>
  );
};
