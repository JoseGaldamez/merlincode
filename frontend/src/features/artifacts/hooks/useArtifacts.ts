import { useState, useCallback } from 'react';
import { Artifact } from '../../../types';

export function useArtifacts() {
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [selectedArtifactId, setSelectedArtifactId] = useState<string | undefined>();

  const addArtifact = useCallback((artifact: Artifact) => {
    setArtifacts((prev) => [artifact, ...prev]);
  }, []);

  const clearArtifacts = useCallback(() => {
    setArtifacts([]);
    setSelectedArtifactId(undefined);
  }, []);

  return {
    artifacts,
    setArtifacts,
    selectedArtifactId,
    setSelectedArtifactId,
    addArtifact,
    clearArtifacts,
  };
}
