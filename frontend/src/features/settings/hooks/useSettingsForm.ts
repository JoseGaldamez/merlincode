import { useState } from 'react';
import { SettingsState } from '../types';
import {
  AI_PROVIDERS,
  DEFAULT_PROVIDER_ID,
  getProviderConfig,
} from '../constants';
import type { ProviderStatusMap } from './useAIProviderStatus';

export const SETTINGS_STORAGE_KEY = 'merlin_app_settings:v3';
export const PREVIOUS_SETTINGS_STORAGE_KEY = 'merlin_app_settings:v2';
export const LEGACY_SETTINGS_STORAGE_KEY = 'merlin_app_settings';

type SettingsStorage = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>;

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

/**
 * Serializa estrictamente el estado a un objeto con lista permitida (allowlist) de propiedades
 * conocidas, garantizando que ningún campo ajeno o secreto heredado se escriba a localStorage.
 */
export function serializeAllowedSettings(state: SettingsState): Record<string, unknown> {
  return {
    modelProvider: String(state.modelProvider || DEFAULT_PROVIDER_ID),
    model: String(state.model || ''),
    temperature:
      typeof state.temperature === 'number' && !Number.isNaN(state.temperature)
        ? Math.max(0, Math.min(1, state.temperature))
        : defaultSettings.temperature,
    streaming: Boolean(state.streaming),
    theme: String(state.theme || defaultSettings.theme),
    language: String(state.language || defaultSettings.language),
    autoSaveSessions: Boolean(state.autoSaveSessions),
    telemetryEnabled: Boolean(state.telemetryEnabled),
  };
}

/**
 * Deserializa un objeto JSON arbitrario inspeccionando únicamente campos conocidos
 * y detectando campos heredados sensibles (apiKey, adminApiKey, apiEndpoint, etc.).
 */
export function parseAndSanitizeSettings(raw: unknown, migrateAutomaticModel = false): {
  settings: SettingsState;
  hasLegacySecrets: boolean;
  hasUnknownKeys: boolean;
} {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
    return { settings: { ...defaultSettings }, hasLegacySecrets: false, hasUnknownKeys: false };
  }

  const record = raw as Record<string, unknown>;
  const legacyKeys = ['apiKey', 'adminApiKey', 'apiEndpoint', 'key', 'token', 'secret'];
  const hasLegacySecrets = legacyKeys.some((k) => k in record);

  const allowedKeySet = new Set([
    'modelProvider',
    'model',
    'temperature',
    'streaming',
    'theme',
    'language',
    'autoSaveSessions',
    'telemetryEnabled',
  ]);
  const hasUnknownKeys = Object.keys(record).some((k) => !allowedKeySet.has(k));

  const rawProvider = typeof record.modelProvider === 'string' ? record.modelProvider : DEFAULT_PROVIDER_ID;
  const validProvider = AI_PROVIDERS.some((p) => p.id === rawProvider) ? rawProvider : DEFAULT_PROVIDER_ID;
  const providerConfig = getProviderConfig(validProvider);

  const rawModel = typeof record.model === 'string' ? record.model : '';
  // DeepSeek V4 Pro sigue en el catálogo, pero era el modelo automático de versiones anteriores.
  const isPreviousDefault = migrateAutomaticModel && validProvider === 'deepseek' && rawModel === 'deepseek-v4-pro';
  const isModelValid = providerConfig.models.some((m) => m.id === rawModel);
  const activeModel = isModelValid && !isPreviousDefault ? rawModel : providerConfig.defaultModel;

  const temp =
    typeof record.temperature === 'number' && !Number.isNaN(record.temperature)
      ? Math.max(0, Math.min(1, record.temperature))
      : defaultSettings.temperature;

  const sanitized: SettingsState = {
    modelProvider: validProvider,
    model: activeModel,
    temperature: temp,
    streaming: typeof record.streaming === 'boolean' ? record.streaming : defaultSettings.streaming,
    theme: typeof record.theme === 'string' ? record.theme : defaultSettings.theme,
    language: typeof record.language === 'string' ? record.language : defaultSettings.language,
    autoSaveSessions:
      typeof record.autoSaveSessions === 'boolean' ? record.autoSaveSessions : defaultSettings.autoSaveSessions,
    telemetryEnabled:
      typeof record.telemetryEnabled === 'boolean' ? record.telemetryEnabled : defaultSettings.telemetryEnabled,
  };

  return { settings: sanitized, hasLegacySecrets, hasUnknownKeys };
}

/**
 * Lee la versión actual o migra la clave heredada sin permitir que campos secretos
 * entren al estado React. El origen heredado se elimina incluso si está corrupto.
 */
export function loadAndMigrateSettings(storage: SettingsStorage): SettingsState {
  try {
    const current = storage.getItem(SETTINGS_STORAGE_KEY);
    const saved = current || storage.getItem(PREVIOUS_SETTINGS_STORAGE_KEY) || storage.getItem(LEGACY_SETTINGS_STORAGE_KEY);

    if (!saved) {
      return { ...defaultSettings };
    }

    const parsed: unknown = JSON.parse(saved);
    const { settings } = parseAndSanitizeSettings(parsed, !current);
    storage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(serializeAllowedSettings(settings)));
    storage.removeItem(PREVIOUS_SETTINGS_STORAGE_KEY);
    storage.removeItem(LEGACY_SETTINGS_STORAGE_KEY);
    return settings;
  } catch {
    try {
      storage.removeItem(LEGACY_SETTINGS_STORAGE_KEY);
      storage.removeItem(PREVIOUS_SETTINGS_STORAGE_KEY);
      storage.removeItem(SETTINGS_STORAGE_KEY);
    } catch {}
    return { ...defaultSettings };
  }
}

export function selectSettingsProvider(settings: SettingsState, providerId: string): SettingsState {
  const provider = getProviderConfig(providerId);
  return {
    ...settings,
    modelProvider: provider.id,
    model: provider.models.some((model) => model.id === settings.model) ? settings.model : provider.defaultModel,
  };
}

export function useSettingsForm(statuses: ProviderStatusMap) {
  const [savedSettings, setSavedSettings] = useState(() => loadAndMigrateSettings(localStorage));
  const isVerified = (id: string) => statuses[id]?.configured && statuses[id]?.verified;
  const fallback = isVerified(savedSettings.modelProvider) ? undefined : AI_PROVIDERS.find((provider) => isVerified(provider.id));
  const settings = fallback ? selectSettingsProvider(savedSettings, fallback.id) : savedSettings;

  const saveSettings = (next: SettingsState) => {
    const { settings: sanitized } = parseAndSanitizeSettings(next);
    setSavedSettings(sanitized);
    try {
      localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(serializeAllowedSettings(sanitized)));
    } catch {}
  };

  return {
    settings,
    selectProvider: (providerId: string) => saveSettings(selectSettingsProvider(settings, providerId)),
    saveSettings,
  };
}
