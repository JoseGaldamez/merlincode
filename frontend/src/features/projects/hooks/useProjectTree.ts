import { useState, useEffect, useCallback, useRef } from 'react';
import { domain } from '../../../../wailsjs/go/models';
import { GetProjectTree, SetActiveProject } from '../../../../wailsjs/go/app/App';
import { EventsOn } from '../../../../wailsjs/runtime/runtime';

export function useProjectTree(projectPath?: string) {
  const [nodes, setNodes] = useState<domain.FileNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [expandedPaths, setExpandedPaths] = useState<Record<string, boolean>>({});
  const [selectedPath, setSelectedPath] = useState<string | null>(null);

  const refreshRef = useRef<() => Promise<void>>(async () => {});
  const refreshTree = useCallback(() => refreshRef.current(), []);

  useEffect(() => {
    let disposed = false;
    let requestId = 0;
    // Activar solo al cambiar de proyecto. Una actualización del árbol nunca
    // debe reactivar una carpeta anterior mientras se cambia de proyecto.
    const ready = projectPath ? SetActiveProject(projectPath) : Promise.resolve();
    const loadTree = async () => {
      const currentRequest = ++requestId;
      if (!projectPath) {
        setNodes([]);
        setLoading(false);
        return;
      }
      setLoading(true);
      try {
        await ready;
        if (disposed || currentRequest !== requestId) return;
        const tree = await GetProjectTree();
        if (!disposed && currentRequest === requestId) setNodes(tree || []);
      } catch {
        if (!disposed && currentRequest === requestId) setNodes([]);
      } finally {
        if (!disposed && currentRequest === requestId) setLoading(false);
      }
    };
    refreshRef.current = loadTree;
    setNodes([]);
    const unsubscribe = EventsOn('workspace:changed', () => { void loadTree(); });
    void loadTree();
    return () => {
      disposed = true;
      unsubscribe();
    };
  }, [projectPath]);

  const toggleDirectory = useCallback((relPath: string) => {
    setExpandedPaths((prev) => ({
      ...prev,
      [relPath]: prev[relPath] === undefined ? false : !prev[relPath],
    }));
  }, []);

  const isExpanded = useCallback(
    (relPath: string) => expandedPaths[relPath] ?? true,
    [expandedPaths]
  );

  return {
    nodes,
    loading,
    selectedPath,
    setSelectedPath,
    toggleDirectory,
    isExpanded,
    refreshTree,
  };
}
