import React, { useState, useEffect } from 'react';
import { SettingsState } from '../../types';
import { DailyUsageCards } from '../DailyUsageCards';
import { AI_PROVIDERS, getProviderConfig } from '../../constants';
import { BrowserOpenURL } from '../../../../../wailsjs/runtime/runtime';
import { useAIProviderStatus } from '../../hooks/useAIProviderStatus';
import {
  IconAnthropic,
  IconOpenAI,
  IconGoogle,
  IconDeepSeek,
  IconEye,
  IconEyeOff,
  IconExternalLink,
  IconShield,
  IconCheck,
  IconClose,
  IconRefresh,
} from '../../../../components/Icons';

interface AISettingsTabProps {
  settings: SettingsState;
  onUpdate: <K extends keyof SettingsState>(key: K, value: SettingsState[K]) => void;
}

export const AISettingsTab: React.FC<AISettingsTabProps> = ({
  settings,
  onUpdate,
}) => {
  const [showApiKey, setShowApiKey] = useState(false);
  const [draftKey, setDraftKey] = useState('');
  const [validationResult, setValidationResult] = useState<{ valid: boolean; message: string } | null>(null);
  const [keyTab, setKeyTab] = useState<'api' | 'admin'>('api');
  const [showAdminKey, setShowAdminKey] = useState(false);
  const [draftAdminKey, setDraftAdminKey] = useState('');
  const [testMessage, setTestMessage] = useState('¿Estás funcionando correctamente?');
  const [testMessageResult, setTestMessageResult] = useState<{
    success: boolean;
    message: string;
    responseText?: string;
    model?: string;
  } | null>(null);

  const currentProvider = getProviderConfig(settings.modelProvider);
  const {
    statuses,
    getStatus,
    validateAndSaveKey,
    clearKey,
    validatingProviderId,
    saveAdminKey,
    clearAdminKey,
    sendTestMessage,
    sendingTestMessageProviderId,
  } = useAIProviderStatus();
  const currentStatus = getStatus(settings.modelProvider);
  const isValidating = validatingProviderId === settings.modelProvider;
  const isSendingTestMessage = sendingTestMessageProviderId === settings.modelProvider;

  // Al cambiar de proveedor, limpiar el borrador y el resultado de validación:
  // cada proveedor mantiene su propio estado, totalmente desacoplado del resto
  useEffect(() => {
    setDraftKey('');
    setValidationResult(null);
    setShowApiKey(false);
    setKeyTab('api');
    setDraftAdminKey('');
    setShowAdminKey(false);
    setTestMessageResult(null);
  }, [settings.modelProvider]);

  const handleValidateAndSave = async () => {
    if (!draftKey.trim()) return;
    setValidationResult(null);
    try {
      const result = await validateAndSaveKey(settings.modelProvider, draftKey.trim());
      setValidationResult({ valid: result.valid, message: result.message });
      if (result.valid) {
        setDraftKey('');
      }
    } catch (err) {
      setValidationResult({
        valid: false,
        message: err instanceof Error ? err.message : 'No se pudo validar la clave de API.',
      });
    }
  };

  const handleClearKey = async () => {
    await clearKey(settings.modelProvider);
    setValidationResult(null);
    setDraftKey('');
  };

  const handleSendTestMessage = async () => {
    setTestMessageResult(null);
    try {
      const result = await sendTestMessage(settings.modelProvider, testMessage.trim(), settings.model);
      setTestMessageResult(result);
    } catch (err) {
      setTestMessageResult({
        success: false,
        message: err instanceof Error ? err.message : 'No se pudo enviar el mensaje de prueba.',
      });
    }
  };

  const renderOfficialLogo = (id: string, size = 22) => {
    switch (id) {
      case 'anthropic':
        return <IconAnthropic size={size} className="text-[#CC785C]" />;
      case 'openai':
        return <IconOpenAI size={size} className="text-white" />;
      case 'google':
        return <IconGoogle size={size} />;
      case 'deepseek':
        return <IconDeepSeek size={size} className="text-[#4D6BFE]" />;
      default:
        return null;
    }
  };

  return (
    <div className="flex flex-col gap-7 max-w-3xl">
      {/* Sección 1: Selección de Proveedores */}
      <div>
        <div className="mb-3">
          <h3 className="text-xs font-semibold tracking-wider text-content-headline uppercase">
            Proveedor de Inteligencia Artificial
          </h3>
          <p className="text-xs text-content-dim mt-0.5 leading-relaxed">
            Selecciona el motor neuronal oficial para la generación y asistencia de código.
          </p>
        </div>

        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {AI_PROVIDERS.map((provider) => {
            const isSelected = settings.modelProvider === provider.id;
            const isUnconfigured = statuses[provider.id]?.configured === false;

            return (
              <button
                key={provider.id}
                type="button"
                aria-pressed={isSelected}
                onClick={() => onUpdate('modelProvider', provider.id)}
                className={`relative flex flex-col justify-between p-3.5 rounded-xl border text-left transition-colors cursor-pointer min-h-[102px] ${
                  isSelected
                    ? 'border-accent-primary bg-surface-neutral'
                    : 'border-border-subtle/70 bg-surface-lowest hover:border-border-petrol/60 hover:bg-surface-neutral'
                } ${isUnconfigured ? 'grayscale' : ''}`}
              >
                <div className="flex items-center justify-between w-full">
                  <div className="w-10 h-10 rounded-lg bg-surface-card border border-border-subtle/50 flex items-center justify-center shrink-0">
                    {renderOfficialLogo(provider.id, 22)}
                  </div>

                  {isSelected && (
                    <div className="w-4 h-4 rounded-full bg-accent-primary flex items-center justify-center text-[#070a0b] shrink-0">
                      <svg
                        width="9"
                        height="9"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="3.5"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      >
                        <polyline points="20 6 9 17 4 12" />
                      </svg>
                    </div>
                  )}
                </div>

                <div className="mt-2.5">
                  <span className={`text-sm font-semibold block leading-tight ${isUnconfigured ? 'text-content-muted' : 'text-content-headline'}`}>
                    {provider.name}
                  </span>
                  <span className="text-xs text-content-dim block mt-1 leading-tight font-mono text-[11px]">
                    {isUnconfigured ? 'Sin clave' : provider.badge}
                  </span>
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {/* Sección 2: Claves (API estándar y Admin, en tabs) */}
      <div>
        <div className="mb-3">
          <h3 className="text-xs font-semibold tracking-wider text-content-headline uppercase">
            Credenciales ({currentProvider.name})
          </h3>
          <p className="text-xs text-content-dim mt-0.5 leading-relaxed">
            {keyTab === 'api'
              ? `Credencial requerida para autenticar peticiones directamente con ${currentProvider.name}.`
              : `Clave opcional de organización de ${currentProvider.name} para consultar sus reportes administrativos.`}
          </p>
        </div>

        <div className="flex items-center gap-1 border-b border-border-subtle/60 mb-4">
          <button
            type="button"
            onClick={() => setKeyTab('api')}
            className={`px-4 py-2 text-xs font-semibold whitespace-nowrap transition-colors cursor-pointer -mb-px border-b-2 ${
              keyTab === 'api'
                ? 'border-accent-primary text-content-headline'
                : 'border-transparent text-content-dim hover:text-content-headline hover:border-border-petrol/60'
            }`}
          >
            Clave de API
          </button>
          {currentStatus.supportsAdminKey && (
            <button
              type="button"
              onClick={() => setKeyTab('admin')}
              className={`px-4 py-2 text-xs font-semibold whitespace-nowrap transition-colors cursor-pointer -mb-px border-b-2 ${
                keyTab === 'admin'
                  ? 'border-accent-primary text-content-headline'
                  : 'border-transparent text-content-dim hover:text-content-headline hover:border-border-petrol/60'
              }`}
            >
              Clave de Administrador
            </button>
          )}
        </div>

        {keyTab === 'api' && (
        <div className="divide-y divide-border-subtle/40 border-t border-b border-border-subtle/40">
          <div className="py-4 grid grid-cols-1 md:grid-cols-12 gap-6 items-start">
            <div className="md:col-span-5">
              <label className="text-sm font-semibold text-content-headline block">
                Clave de API
              </label>
              <p className="text-xs text-content-dim mt-1 leading-relaxed">
                Tu clave se almacena de forma local en tu máquina y nunca se comparte con terceros.
              </p>
              <div className="flex items-center gap-1.5 mt-2.5 text-content-muted text-xs">
                <IconShield size={13} className="text-accent-primary/90 shrink-0" />
                <span>Almacenada exclusivamente en tu equipo.</span>
              </div>
            </div>

            <div className="md:col-span-7 flex flex-col gap-2">
              {currentStatus.configured ? (
                <>
                  <div className="flex items-center justify-between gap-2 p-2.5 rounded-lg bg-surface-lowest border border-border-subtle/60">
                    <div className="flex items-center gap-2 min-w-0">
                      {currentStatus.verified ? (
                        <IconCheck size={14} className="text-emerald-400 shrink-0" />
                      ) : (
                        <IconClose size={14} className="text-amber-400 shrink-0" />
                      )}
                      <span className="text-xs font-mono text-content-headline truncate">
                        {currentStatus.maskedKey}
                      </span>
                    </div>
                    <button
                      type="button"
                      onClick={handleClearKey}
                      className="text-[11px] text-content-dim hover:text-red-400 shrink-0 transition-colors cursor-pointer"
                    >
                      Eliminar
                    </button>
                  </div>
                  {currentStatus.accountInfo && (
                    <p className="text-[11px] text-content-dim leading-relaxed">
                      {currentStatus.accountInfo}
                    </p>
                  )}
                </>
              ) : (
                <>
                  <div className="relative">
                    <input
                      type={showApiKey ? 'text' : 'password'}
                      className="w-full bg-surface-lowest border border-border-subtle hover:border-border-petrol focus:border-accent-primary rounded-lg pl-3.5 pr-10 py-2.5 text-sm text-content-headline font-mono outline-none transition-colors"
                      value={draftKey}
                      onChange={(e) => {
                        setDraftKey(e.target.value);
                        setValidationResult(null);
                      }}
                      placeholder={currentProvider.keyPlaceholder}
                      spellCheck={false}
                    />
                    <button
                      type="button"
                      onClick={() => setShowApiKey(!showApiKey)}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-content-dim hover:text-content-headline p-1 rounded transition-colors"
                      title={showApiKey ? 'Ocultar clave' : 'Mostrar clave'}
                    >
                      {showApiKey ? <IconEyeOff size={16} /> : <IconEye size={16} />}
                    </button>
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={handleValidateAndSave}
                      disabled={!draftKey.trim() || isValidating}
                      className="inline-flex items-center gap-1.5 text-xs font-semibold px-3 py-1.5 rounded-lg bg-accent-primary text-[#070a0b] hover:brightness-110 disabled:opacity-40 disabled:cursor-not-allowed transition-all cursor-pointer"
                    >
                      {isValidating && <IconRefresh size={12} className="animate-spin" />}
                      {isValidating ? 'Verificando conexión...' : 'Verificar y guardar'}
                    </button>

                    <button
                      type="button"
                      onClick={() => {
                        try {
                          BrowserOpenURL(currentProvider.keyHelpUrl);
                        } catch {
                          window.open(currentProvider.keyHelpUrl, '_blank', 'noopener,noreferrer');
                        }
                      }}
                      className="inline-flex items-center gap-1.5 text-xs text-accent-primary hover:underline transition-colors cursor-pointer"
                    >
                      <span>Obtener clave en {currentProvider.name} Console</span>
                      <IconExternalLink size={12} />
                    </button>
                  </div>

                  {validationResult && (
                    <div
                      className={`flex items-start gap-1.5 text-xs mt-0.5 ${
                        validationResult.valid ? 'text-emerald-400' : 'text-red-400'
                      }`}
                    >
                      {validationResult.valid ? (
                        <IconCheck size={13} className="shrink-0 mt-0.5" />
                      ) : (
                        <IconClose size={13} className="shrink-0 mt-0.5" />
                      )}
                      <span>{validationResult.message}</span>
                    </div>
                  )}
                </>
              )}
            </div>
          </div>
        </div>
        )}

        {keyTab === 'admin' && currentStatus.supportsAdminKey && (
        <div className="divide-y divide-border-subtle/40 border-t border-b border-border-subtle/40">
          <div className="py-4 grid grid-cols-1 md:grid-cols-12 gap-6 items-start">
            <div className="md:col-span-5">
              <label className="text-sm font-semibold text-content-headline block">
                Admin API Key
              </label>
              <p className="text-xs text-content-dim mt-1 leading-relaxed">
                Se almacena localmente, igual que tu clave de API, y solo se usa para consultar reportes de uso/costo.
              </p>
            </div>

            <div className="md:col-span-7 flex flex-col gap-2">
              {currentStatus.hasAdminKey ? (
                <div className="flex items-center justify-between gap-2 p-2.5 rounded-lg bg-surface-lowest border border-border-subtle/60">
                  <div className="flex items-center gap-2 min-w-0">
                    <IconShield size={14} className="text-accent-primary/90 shrink-0" />
                    <span className="text-xs font-mono text-content-headline truncate">
                      {currentStatus.maskedAdminKey}
                    </span>
                  </div>
                  <button
                    type="button"
                    onClick={async () => {
                      await clearAdminKey(settings.modelProvider);
                    }}
                    className="text-[11px] text-content-dim hover:text-red-400 shrink-0 transition-colors cursor-pointer"
                  >
                    Eliminar
                  </button>
                </div>
              ) : (
                <>
                  <div className="relative">
                    <input
                      type={showAdminKey ? 'text' : 'password'}
                      className="w-full bg-surface-lowest border border-border-subtle hover:border-border-petrol focus:border-accent-primary rounded-lg pl-3.5 pr-10 py-2.5 text-sm text-content-headline font-mono outline-none transition-colors"
                      value={draftAdminKey}
                      onChange={(e) => setDraftAdminKey(e.target.value)}
                      placeholder="sk-ant-admin..."
                      spellCheck={false}
                    />
                    <button
                      type="button"
                      onClick={() => setShowAdminKey(!showAdminKey)}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-content-dim hover:text-content-headline p-1 rounded transition-colors"
                      title={showAdminKey ? 'Ocultar clave' : 'Mostrar clave'}
                    >
                      {showAdminKey ? <IconEyeOff size={16} /> : <IconEye size={16} />}
                    </button>
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={async () => {
                        if (!draftAdminKey.trim()) return;
                        await saveAdminKey(settings.modelProvider, draftAdminKey.trim());
                        setDraftAdminKey('');
                      }}
                      disabled={!draftAdminKey.trim()}
                      className="inline-flex items-center gap-1.5 text-xs font-semibold px-3 py-1.5 rounded-lg bg-accent-primary text-[#070a0b] hover:brightness-110 disabled:opacity-40 disabled:cursor-not-allowed transition-all cursor-pointer"
                    >
                      Guardar Admin Key
                    </button>
                  </div>

                </>
              )}
            </div>
          </div>
        </div>
        )}
      </div>

      {/* Sección 2.0: Probar Conexión (envía un mensaje real usando la clave de API estándar, no la Admin Key) */}
      {currentStatus.configured && currentStatus.verified && (
        <div>
          <div className="mb-3">
            <h3 className="text-xs font-semibold tracking-wider text-content-headline uppercase">
              Probar Conexión ({currentProvider.name})
            </h3>
            <p className="text-xs text-content-dim mt-0.5 leading-relaxed">
              Envía un mensaje real al modelo para confirmar que la clave funciona de punta a punta. Se usa
              tu clave de API estándar, no la Admin Key (la Admin Key solo consulta reportes de organización
              y no tiene permiso para generar respuestas).
            </p>
            <p className="text-xs text-content-dim mt-1.5">
              Modelo a probar:{' '}
              <span className="font-mono text-content-headline">
                {settings.model || currentProvider.defaultModel}
              </span>
            </p>
          </div>

          <div className="divide-y divide-border-subtle/40 border-t border-b border-border-subtle/40">
            <div className="py-4 grid grid-cols-1 md:grid-cols-12 gap-6 items-start">
              <div className="md:col-span-5">
                <label className="text-sm font-semibold text-content-headline block">
                  Mensaje de prueba
                </label>
                <p className="text-xs text-content-dim mt-1 leading-relaxed">
                  El consumo generado debería reflejarse en el reporte de uso real en unos minutos.
                </p>
              </div>

              <div className="md:col-span-7 flex flex-col gap-2">
                <input
                  type="text"
                  className="w-full bg-surface-lowest border border-border-subtle hover:border-border-petrol focus:border-accent-primary rounded-lg px-3.5 py-2.5 text-sm text-content-headline outline-none transition-colors"
                  value={testMessage}
                  onChange={(e) => setTestMessage(e.target.value)}
                  placeholder="Escribe un mensaje para probar la conexión..."
                  spellCheck={false}
                />

                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={handleSendTestMessage}
                    disabled={!testMessage.trim() || isSendingTestMessage}
                    className="inline-flex items-center gap-1.5 text-xs font-semibold px-3 py-1.5 rounded-lg bg-accent-primary text-[#070a0b] hover:brightness-110 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
                  >
                    {isSendingTestMessage && <IconRefresh size={12} className="animate-spin" />}
                    {isSendingTestMessage ? 'Enviando mensaje...' : 'Enviar mensaje de prueba'}
                  </button>
                </div>

                {testMessageResult && (
                  <div className="flex flex-col gap-1.5 mt-0.5">
                    <div
                      className={`flex items-start gap-1.5 text-xs ${
                        testMessageResult.success ? 'text-emerald-400' : 'text-red-400'
                      }`}
                    >
                      {testMessageResult.success ? (
                        <IconCheck size={13} className="shrink-0 mt-0.5" />
                      ) : (
                        <IconClose size={13} className="shrink-0 mt-0.5" />
                      )}
                      <span>{testMessageResult.message}</span>
                    </div>
                    {testMessageResult.responseText && (
                      <div className="p-3 rounded-lg bg-surface-lowest border border-border-subtle/60">
                        {testMessageResult.model && (
                          <p className="text-[10px] font-mono text-content-dim mb-1.5">
                            {testMessageResult.model}
                          </p>
                        )}
                        <p className="text-xs text-content-headline whitespace-pre-wrap leading-relaxed">
                          {testMessageResult.responseText}
                        </p>
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      <DailyUsageCards key={settings.modelProvider} providerId={settings.modelProvider} />
      {/* Sección 4: Opciones de Inferencia */}
      <div>
        <div className="mb-3">
          <h3 className="text-xs font-semibold tracking-wider text-content-headline uppercase">
            Inferencia & Flujo de Respuesta
          </h3>
          <p className="text-xs text-content-dim mt-0.5 leading-relaxed">
            Comportamiento de transmisión en tiempo real de los tokens de código.
          </p>
        </div>

        <div className="divide-y divide-border-subtle/40 border-t border-b border-border-subtle/40">
          <div className="py-4 flex items-center justify-between gap-4 md:grid md:grid-cols-12 md:gap-6">
            <div className="md:col-span-5">
              <label className="text-sm font-semibold text-content-headline block">
                Streaming de Tokens
              </label>
              <p className="text-xs text-content-dim mt-1 leading-relaxed">
                Transmitir la respuesta en tiempo real a medida que el modelo genera el código.
              </p>
            </div>
            <div className="md:col-span-7 flex items-center justify-end">
              <label className="relative inline-flex items-center cursor-pointer shrink-0">
                <input
                  type="checkbox"
                  checked={settings.streaming}
                  onChange={(e) => onUpdate('streaming', e.target.checked)}
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-surface-container peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-content-muted peer-checked:after:bg-canvas after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-accent-primary"></div>
              </label>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
