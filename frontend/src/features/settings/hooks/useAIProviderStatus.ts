import { useCallback, useEffect, useState } from 'react';
import {
  ListAIProviderStatuses,
  ValidateAndSaveAIProviderKey,
  ClearAIProviderKey,
  SaveAIProviderAdminKey,
  ClearAIProviderAdminKey,
  GetAIProviderUsage,
} from '../../../../wailsjs/go/main/App';
import { domain } from '../../../../wailsjs/go/models';

export type ProviderStatusMap = Record<string, domain.ProviderStatus>;
export type ProviderUsageMap = Record<string, domain.ProviderUsageResult>;

/**
 * Gestiona el estado real (backend Go) de las credenciales de cada proveedor de IA:
 * carga los estados persistidos, y permite validar/guardar o borrar la key de un proveedor
 * concreto sin afectar a los demás (cada proveedor vive de forma independiente en el backend).
 */
export function useAIProviderStatus() {
  const [statuses, setStatuses] = useState<ProviderStatusMap>({});
  const [loadingStatuses, setLoadingStatuses] = useState(true);
  const [validatingProviderId, setValidatingProviderId] = useState<string | null>(null);
  const [usages, setUsages] = useState<ProviderUsageMap>({});
  const [loadingUsageProviderId, setLoadingUsageProviderId] = useState<string | null>(null);

  const refreshStatuses = useCallback(async () => {
    try {
      const list = await ListAIProviderStatuses();
      const map: ProviderStatusMap = {};
      for (const status of list) {
        map[status.providerId] = status;
      }
      setStatuses(map);
    } catch {
      // Si el backend aún no está disponible (p.ej. en dev sin Wails), no rompemos la UI
    } finally {
      setLoadingStatuses(false);
    }
  }, []);

  useEffect(() => {
    refreshStatuses();
  }, [refreshStatuses]);

  const validateAndSaveKey = useCallback(
    async (providerId: string, apiKey: string): Promise<domain.ProviderValidationResult> => {
      setValidatingProviderId(providerId);
      try {
        const result = await ValidateAndSaveAIProviderKey(providerId, apiKey);
        if (result.valid) {
          await refreshStatuses();
        }
        return result;
      } finally {
        setValidatingProviderId((current) => (current === providerId ? null : current));
      }
    },
    [refreshStatuses]
  );

  const clearKey = useCallback(
    async (providerId: string) => {
      await ClearAIProviderKey(providerId);
      await refreshStatuses();
    },
    [refreshStatuses]
  );

  const getStatus = useCallback(
    (providerId: string): domain.ProviderStatus =>
      statuses[providerId] ?? {
        providerId,
        configured: false,
        verified: false,
        supportsAdminKey: false,
        hasAdminKey: false,
      },
    [statuses]
  );

  const saveAdminKey = useCallback(
    async (providerId: string, adminKey: string) => {
      await SaveAIProviderAdminKey(providerId, adminKey);
      await refreshStatuses();
    },
    [refreshStatuses]
  );

  const clearAdminKey = useCallback(
    async (providerId: string) => {
      await ClearAIProviderAdminKey(providerId);
      await refreshStatuses();
    },
    [refreshStatuses]
  );

  const fetchUsage = useCallback(async (providerId: string) => {
    setLoadingUsageProviderId(providerId);
    try {
      const result = await GetAIProviderUsage(providerId);
      setUsages((prev) => ({ ...prev, [providerId]: result }));
      return result;
    } finally {
      setLoadingUsageProviderId((current) => (current === providerId ? null : current));
    }
  }, []);

  const getUsage = useCallback(
    (providerId: string): domain.ProviderUsageResult | undefined => usages[providerId],
    [usages]
  );

  return {
    statuses,
    loadingStatuses,
    validatingProviderId,
    getStatus,
    validateAndSaveKey,
    clearKey,
    saveAdminKey,
    clearAdminKey,
    fetchUsage,
    getUsage,
    loadingUsageProviderId,
  };
}
