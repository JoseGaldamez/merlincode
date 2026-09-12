import { useState, useEffect, useCallback } from 'react';
import { domain } from '../../../../wailsjs/go/models';
import { GetProjectTree, SetActiveProject } from '../../../../wailsjs/go/main/App';

export function useProjectTree(projectPath?: string) {
  const [nodes, setNodes] = useState<domain.FileNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [expandedPaths, setExpandedPaths] = useState<Record<string, boolean>>({});
  const [selectedPath, setSelectedPath] = useState<string | null>(null);

  const loadTree = useCallback(async () => {
    if (!projectPath) {
      setNodes([]);
      return;
    }
    setLoading(true);
    try {
      // Asegurar que el backend de Go tenga este directorio configurado como activo
      // antes de solicitar el árbol de archivos (evita condición de carrera al iniciar la app)
      await SetActiveProject(projectPath);
      const tree = await GetProjectTree();
      setNodes(tree || []);
    } catch {
      setNodes([]);
    } finally {
      setLoading(false);
    }
  }, [projectPath]);

  useEffect(() => {
    loadTree();
  }, [loadTree]);

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
    refreshTree: loadTree,
  };
}
