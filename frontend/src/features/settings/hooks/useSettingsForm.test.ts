// @vitest-environment jsdom
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { describe, expect, it, vi } from 'vitest';
import {
  LEGACY_SETTINGS_STORAGE_KEY,
  PREVIOUS_SETTINGS_STORAGE_KEY,
  SETTINGS_STORAGE_KEY,
  loadAndMigrateSettings,
  serializeAllowedSettings,
  useSettingsForm,
} from './useSettingsForm';
import type { ProviderStatusMap } from './useAIProviderStatus';

class MemoryStorage {
  private readonly values = new Map<string, string>();

  constructor(initial: Record<string, string> = {}) {
    for (const [key, value] of Object.entries(initial)) {
      this.values.set(key, value);
    }
  }

  getItem(key: string): string | null {
    return this.values.get(key) ?? null;
  }

  setItem(key: string, value: string): void {
    this.values.set(key, value);
  }

  removeItem(key: string): void {
    this.values.delete(key);
  }
}

describe('preferencias seguras', () => {
  it('migra preferencias heredadas y elimina secretos antes de crear el estado', () => {
    const sentinel = 'SENTINEL_BROWSER_SECRET_12345';
    const storage = new MemoryStorage({
      [LEGACY_SETTINGS_STORAGE_KEY]: JSON.stringify({
        modelProvider: 'google',
        model: 'gemini-2.5-pro',
        temperature: 0.4,
        streaming: true,
        theme: 'dark-flat',
        language: 'es',
        autoSaveSessions: true,
        telemetryEnabled: false,
        apiKey: sentinel,
        adminApiKey: `${sentinel}_ADMIN`,
        apiEndpoint: `https://example.invalid?key=${sentinel}`,
        unknownField: sentinel,
      }),
    });

    const settings = loadAndMigrateSettings(storage);
    const persisted = storage.getItem(SETTINGS_STORAGE_KEY);

    expect(settings.modelProvider).toBe('google');
    expect(settings).not.toHaveProperty('apiKey');
    expect(settings).not.toHaveProperty('adminApiKey');
    expect(storage.getItem(LEGACY_SETTINGS_STORAGE_KEY)).toBeNull();
    expect(persisted).not.toBeNull();
    expect(persisted).not.toContain(sentinel);
    expect(persisted).not.toContain('apiKey');
    expect(persisted).not.toContain('apiEndpoint');
  });

  it('usa la misma lista permitida al volver a guardar', () => {
    const storage = new MemoryStorage({
      [SETTINGS_STORAGE_KEY]: JSON.stringify({
        modelProvider: 'openai',
        model: 'gpt-4o-mini',
        temperature: 4,
        streaming: false,
        theme: 'dark-flat',
        language: 'es',
        autoSaveSessions: false,
        telemetryEnabled: false,
        token: 'SENTINEL_REAPPEAR',
      }),
    });

    const settings = loadAndMigrateSettings(storage);
    storage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(serializeAllowedSettings(settings)));
    const persisted = storage.getItem(SETTINGS_STORAGE_KEY) ?? '';

    expect(settings.temperature).toBe(1);
    expect(persisted).not.toContain('SENTINEL_REAPPEAR');
    expect(persisted).not.toContain('token');
  });

  it.each([
    ['anthropic', 'claude-sonnet-5', 'claude-sonnet-5-5'],
    ['openai', 'gpt-5.6-terra', 'gpt-6.2-sol'],
    ['openai', 'gpt-6-sol', 'gpt-6.2-sol'],
    ['google', 'gemini-3.8-flash', 'gemini-4-hydrogen'],
    ['deepseek', 'deepseek-v4-pro', 'deepseek-flash'],
  ])('actualiza el modelo automático anterior de %s', (provider, previous, current) => {
    const storage = new MemoryStorage({
      [PREVIOUS_SETTINGS_STORAGE_KEY]: JSON.stringify({ modelProvider: provider, model: previous }),
    });

    const settings = loadAndMigrateSettings(storage);

    expect(settings.modelProvider).toBe(provider);
    expect(settings.model).toBe(current);
    expect(storage.getItem(SETTINGS_STORAGE_KEY)).toContain(current);
    expect(storage.getItem(PREVIOUS_SETTINGS_STORAGE_KEY)).toBeNull();
  });

  it('corrige un modelo guardado que pertenece a otro proveedor', () => {
    const storage = new MemoryStorage({
      [PREVIOUS_SETTINGS_STORAGE_KEY]: JSON.stringify({ modelProvider: 'openai', model: 'claude-sonnet-5-5' }),
    });
    expect(loadAndMigrateSettings(storage)).toMatchObject({ modelProvider: 'openai', model: 'gpt-6.2-sol' });
  });

  it('elimina una configuración corrupta en vez de conservar posibles secretos', () => {
    const storage = new MemoryStorage({
      [LEGACY_SETTINGS_STORAGE_KEY]: '{"apiKey":"SENTINEL_BROKEN"',
    });

    const settings = loadAndMigrateSettings(storage);

    expect(settings.modelProvider).toBe('anthropic');
    expect(storage.getItem(LEGACY_SETTINGS_STORAGE_KEY)).toBeNull();
  });
});

it('purges legacy secrets even when current storage exists or is corrupt', () => {
  for (const current of ['{}', '{invalid']) {
    const storage = new MemoryStorage({
      [SETTINGS_STORAGE_KEY]: current,
      [LEGACY_SETTINGS_STORAGE_KEY]: '{"apiKey":"SENTINEL"}',
    });
    expect(JSON.stringify(loadAndMigrateSettings(storage))).not.toContain('SENTINEL');
    expect(storage.getItem(LEGACY_SETTINGS_STORAGE_KEY)).toBeNull();
  }
});

it('guarda el proveedor de la barra y restaura el modelo elegido al volver a abrir la aplicación', async () => {
  const providerStatus = (providerId: string) => ({
    providerId, configured: true, verified: true, supportsAdminKey: false, hasAdminKey: false,
  });
  const statuses: ProviderStatusMap = { openai: providerStatus('openai'), deepseek: providerStatus('deepseek') };
  let current!: ReturnType<typeof useSettingsForm>;
  function Harness({ providers }: { providers: ProviderStatusMap }) {
    current = useSettingsForm(providers);
    return null;
  }

  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  localStorage.clear();
  const root = createRoot(document.createElement('div'));
  try {
    await act(async () => root.render(createElement(Harness, { providers: { openai: statuses.openai } })));
    expect(current.settings).toMatchObject({ modelProvider: 'openai', model: 'gpt-6.2-sol' });
    expect(localStorage.getItem(SETTINGS_STORAGE_KEY)).toBeNull();

    await act(async () => root.render(createElement(Harness, { providers: statuses })));
    await act(async () => current.selectProvider('deepseek'));
    expect(loadAndMigrateSettings(localStorage)).toMatchObject({ modelProvider: 'deepseek', model: 'deepseek-flash' });

    await act(async () => current.saveSettings({ ...current.settings, model: 'deepseek-v4-pro' }));
    await act(async () => root.render(null));
    await act(async () => root.render(createElement(Harness, { providers: statuses })));
    expect(current.settings).toMatchObject({ modelProvider: 'deepseek', model: 'deepseek-v4-pro' });
  } finally {
    await act(async () => root.unmount());
    localStorage.clear();
    vi.unstubAllGlobals();
  }
});
