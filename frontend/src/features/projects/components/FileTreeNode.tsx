import React from 'react';
import { domain } from '../../../../wailsjs/go/models';
import {
  IconFolder,
  IconFolderOpen,
  IconChevronRight,
  IconChevronDown,
} from '../../../components/Icons';
import { FileIcon } from './FileIcon';

interface FileTreeNodeProps {
  node: domain.FileNode;
  depth?: number;
  isSelected: boolean;
  isExpanded: boolean;
  onSelect: (path: string) => void;
  onToggleExpand: (path: string) => void;
  onDoubleClick: (fullPath: string) => void;
  renderChild: (child: domain.FileNode, depth: number) => React.ReactNode;
}

export const FileTreeNode: React.FC<FileTreeNodeProps> = ({
  node,
  depth = 0,
  isSelected,
  isExpanded,
  onSelect,
  onToggleExpand,
  onDoubleClick,
  renderChild,
}) => {
  const isDir = node.isDir;

  return (
    <div className="flex flex-col">
      <div
        className={`flex items-center gap-1.5 py-1 px-2 rounded text-xs cursor-pointer select-none transition-colors border ${
          isSelected
            ? 'bg-[#18262c] text-accent-primary border-border-petrol'
            : 'text-content-muted hover:bg-[#131a1e] hover:text-content-headline border-transparent'
        }`}
        style={{ paddingLeft: `${depth * 14 + 8}px` }}
        onClick={() => {
          onSelect(node.relPath);
          if (isDir) {
            onToggleExpand(node.relPath);
          }
        }}
        onDoubleClick={() => {
          if (node.fullPath) {
            onDoubleClick(node.fullPath);
          }
        }}
        title={`${node.relPath}${node.size ? ` (${(node.size / 1024).toFixed(1)} KB)` : ''}`}
      >
        {isDir ? (
          <>
            <span className="text-content-dim shrink-0">
              {isExpanded ? <IconChevronDown size={12} /> : <IconChevronRight size={12} />}
            </span>
            <span className="text-accent-primary/80 shrink-0">
              {isExpanded ? <IconFolderOpen size={13} /> : <IconFolder size={13} />}
            </span>
            <span className="truncate font-mono text-[11px] font-medium text-content-body">
              {node.name}
            </span>
          </>
        ) : (
          <>
            <span className="w-3 shrink-0" />
            <FileIcon name={node.name} />
            <span className="truncate font-mono text-[11px] text-content-muted">
              {node.name}
            </span>
          </>
        )}
      </div>

      {isDir && isExpanded && node.children && (
        <div className="flex flex-col">
          {node.children.length === 0 ? (
            <div
              className="text-[10px] font-mono text-content-dim italic py-0.5"
              style={{ paddingLeft: `${(depth + 1) * 14 + 20}px` }}
            >
              (vacío)
            </div>
          ) : (
            node.children.map((child: domain.FileNode) => renderChild(child, depth + 1))
          )}
        </div>
      )}
    </div>
  );
};
