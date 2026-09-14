import { describe, expect, it } from 'vitest';
import {
  LEGACY_SETTINGS_STORAGE_KEY,
  SETTINGS_STORAGE_KEY,
  loadAndMigrateSettings,
  serializeAllowedSettings,
} from './useSettingsForm';

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

  it('elimina una configuración corrupta en vez de conservar posibles secretos', () => {
    const storage = new MemoryStorage({
      [LEGACY_SETTINGS_STORAGE_KEY]: '{"apiKey":"SENTINEL_BROKEN"',
    });

    const settings = loadAndMigrateSettings(storage);

    expect(settings.modelProvider).toBe('anthropic');
    expect(storage.getItem(LEGACY_SETTINGS_STORAGE_KEY)).toBeNull();
  });
});
