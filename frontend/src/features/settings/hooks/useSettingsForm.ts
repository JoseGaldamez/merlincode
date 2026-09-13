import { useState, useEffect } from 'react';
import { AgentTelemetry } from '../../../types';
import { SettingsState } from '../types';
import {
  AI_PROVIDERS,
  DEFAULT_PROVIDER_ID,
  getProviderConfig,
} from '../constants';

const SETTINGS_STORAGE_KEY = 'merlin_app_settings';

const initialProvider = getProviderConfig(DEFAULT_PROVIDER_ID);

const defaultSettings: SettingsState = {
  modelProvider: initialProvider.id,
  model: initialProvider.defaultModel,
  temperature: 0.7,
  streaming: true,
  theme: 'dark-flat',
  language: 'es',
  autoSaveSessions: true,
  telemetryEnabled: false,
};

export function useSettingsForm(
  telemetry: AgentTelemetry,
  onUpdateTelemetry: (updated: Partial<AgentTelemetry>) => void,
  onClose: () => void
) {
  const [settings, setSettings] = useState<SettingsState>(() => {
    try {
      const saved = localStorage.getItem(SETTINGS_STORAGE_KEY);
      if (saved) {
        const parsed = JSON.parse(saved);
        const validProvider = AI_PROVIDERS.some((p) => p.id === parsed.modelProvider)
          ? parsed.modelProvider
          : DEFAULT_PROVIDER_ID;
        const providerConfig = getProviderConfig(validProvider);
        const isModelValid = providerConfig.models.some((m) => m.id === parsed.model);
        const activeModel = isModelValid ? parsed.model : providerConfig.defaultModel;

        return {
          ...defaultSettings,
          ...parsed,
          modelProvider: validProvider,
          model: telemetry.activeModel || activeModel,
          temperature: telemetry.temperature ?? parsed.temperature ?? defaultSettings.temperature,
        };
      }
    } catch {}
    return {
      ...defaultSettings,
      model: telemetry.activeModel || defaultSettings.model,
      temperature: telemetry.temperature ?? defaultSettings.temperature,
    };
  });

  const [savedSuccess, setSavedSuccess] = useState(false);

  // Mantener sincronizado si la telemetría externa cambia
  useEffect(() => {
    if (telemetry.activeModel) {
      setSettings((prev) => ({
        ...prev,
        model: telemetry.activeModel || prev.model,
        temperature: telemetry.temperature ?? prev.temperature,
      }));
    }
  }, [telemetry.activeModel, telemetry.temperature]);

  const updateSetting = <K extends keyof SettingsState>(key: K, value: SettingsState[K]) => {
    setSettings((prev) => {
      // Si cambia el proveedor, sincronizar endpoint oficial y modelo adecuado
      if (key === 'modelProvider') {
        const nextProvider = getProviderConfig(value as string);
        const isCurrentModelValid = nextProvider.models.some((m) => m.id === prev.model);
        return {
          ...prev,
          modelProvider: nextProvider.id,
          model: isCurrentModelValid ? prev.model : nextProvider.defaultModel,
        };
      }
      return { ...prev, [key]: value };
    });
  };

  const handleSave = () => {
    onUpdateTelemetry({
      activeModel: settings.model,
    });

    try {
      localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(settings));
    } catch {}

    setSavedSuccess(true);
    setTimeout(() => {
      setSavedSuccess(false);
      onClose();
    }, 750);
  };

  return {
    settings,
    updateSetting,
    handleSave,
    savedSuccess,
  };
}
