import React, { useState, useEffect } from 'react';
import { SettingsState } from '../../types';
import { AgentTelemetry } from '../../../../types';
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
  IconSearch,
} from '../../../../components/Icons';

interface AISettingsTabProps {
  settings: SettingsState;
  onUpdate: <K extends keyof SettingsState>(key: K, value: SettingsState[K]) => void;
  telemetry?: AgentTelemetry;
}

interface DailyUsageRecord {
  date: string;
  promptTokens: number;
  completionTokens: number;
}

const getStoredDailyUsage = (): DailyUsageRecord => {
  const todayStr = new Date().toISOString().slice(0, 10);
  try {
    const raw = localStorage.getItem('merlin_daily_tokens');
    if (raw) {
      const parsed: DailyUsageRecord = JSON.parse(raw);
      if (parsed.date === todayStr) {
        return parsed;
      }
    }
  } catch {}
  return { date: todayStr, promptTokens: 0, completionTokens: 0 };
};

export const AISettingsTab: React.FC<AISettingsTabProps> = ({
  settings,
  onUpdate,
  telemetry,
}) => {
  const [showApiKey, setShowApiKey] = useState(false);
  const [draftKey, setDraftKey] = useState('');
  const [validationResult, setValidationResult] = useState<{ valid: boolean; message: string } | null>(null);
  const [modelFilter, setModelFilter] = useState('');
  const [keyTab, setKeyTab] = useState<'api' | 'admin'>('api');
  const [showAdminKey, setShowAdminKey] = useState(false);
  const [draftAdminKey, setDraftAdminKey] = useState('');
  const [adminKeySaved, setAdminKeySaved] = useState(false);
  const [testMessage, setTestMessage] = useState('¿Estás funcionando correctamente?');
  const [testMessageResult, setTestMessageResult] = useState<{
    success: boolean;
    message: string;
    responseText?: string;
    model?: string;
  } | null>(null);

  const currentProvider = getProviderConfig(settings.modelProvider);
  const {
    getStatus,
    validateAndSaveKey,
    clearKey,
    validatingProviderId,
    saveAdminKey,
    clearAdminKey,
    fetchUsage,
    getUsage,
    loadingUsageProviderId,
    sendTestMessage,
    sendingTestMessageProviderId,
  } = useAIProviderStatus();
  const currentStatus = getStatus(settings.modelProvider);
  const isValidating = validatingProviderId === settings.modelProvider;
  const currentUsage = getUsage(settings.modelProvider);
  const isLoadingUsage = loadingUsageProviderId === settings.modelProvider;
  const isSendingTestMessage = sendingTestMessageProviderId === settings.modelProvider;

  // Al cambiar de proveedor, limpiar el borrador y el resultado de validación:
  // cada proveedor mantiene su propio estado, totalmente desacoplado del resto
  useEffect(() => {
    setDraftKey('');
    setValidationResult(null);
    setShowApiKey(false);
    setModelFilter('');
    setKeyTab('api');
    setDraftAdminKey('');
    setShowAdminKey(false);
    setAdminKeySaved(false);
    setTestMessageResult(null);
  }, [settings.modelProvider]);

  // Al configurar/verificar la clave, consultar en vivo el consumo/costo o saldo real
  // reportado por el proveedor (si su API lo expone con la clave estándar o la admin key)
  useEffect(() => {
    if (currentStatus.configured && currentStatus.verified) {
      fetchUsage(settings.modelProvider);
    }
  }, [settings.modelProvider, currentStatus.configured, currentStatus.verified, currentStatus.hasAdminKey, fetchUsage]);

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
      if (result.success) {
        fetchUsage(settings.modelProvider);
      }
    } catch (err) {
      setTestMessageResult({
        success: false,
        message: err instanceof Error ? err.message : 'No se pudo enviar el mensaje de prueba.',
      });
    }
  };

  // Persistir tokens en localStorage si la telemetría viva se incrementa
  useEffect(() => {
    if (telemetry && (telemetry.tokensPrompt > 0 || telemetry.tokensCompletion > 0)) {
      const todayStr = new Date().toISOString().slice(0, 10);
      const existing = getStoredDailyUsage();
      const updated: DailyUsageRecord = {
        date: todayStr,
        promptTokens: Math.max(existing.promptTokens, telemetry.tokensPrompt),
        completionTokens: Math.max(existing.completionTokens, telemetry.tokensCompletion),
      };
      try {
        localStorage.setItem('merlin_daily_tokens', JSON.stringify(updated));
      } catch {}
    }
  }, [telemetry?.tokensPrompt, telemetry?.tokensCompletion]);

  // Cálculo de tokens del día de hoy
  const stored = getStoredDailyUsage();
  const livePrompt = telemetry?.tokensPrompt ?? 0;
  const liveCompletion = telemetry?.tokensCompletion ?? 0;

  const todayPrompt = Math.max(stored.promptTokens, livePrompt);
  const todayCompletion = Math.max(stored.completionTokens, liveCompletion);
  const totalTokens = todayPrompt + todayCompletion;

  // Cálculo de dinero en USD según la tarifa del proveedor activo
  const pricing = currentProvider.pricing;
  const promptCostUsd = (todayPrompt * pricing.promptPerMillion) / 1_000_000;
  const completionCostUsd = (todayCompletion * pricing.completionPerMillion) / 1_000_000;
  const totalCostUsd = promptCostUsd + completionCostUsd;

  const formatCost = (val: number) => {
    if (val === 0) return '$0.00 USD';
    if (val < 0.001) return '< $0.001 USD';
    if (val < 0.01) return `$${val.toFixed(4)} USD`;
    return `$${val.toFixed(2)} USD`;
  };

  const promptPct = totalTokens > 0 ? Math.round((todayPrompt / totalTokens) * 100) : 50;
  const completionPct = totalTokens > 0 ? 100 - promptPct : 50;

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

            return (
              <button
                key={provider.id}
                type="button"
                onClick={() => onUpdate('modelProvider', provider.id)}
                className={`relative flex flex-col justify-between p-3.5 rounded-xl border text-left transition-colors cursor-pointer min-h-[102px] ${
                  isSelected
                    ? 'border-accent-primary bg-[#101719]'
                    : 'border-border-subtle/70 bg-[#0d1214] hover:border-border-petrol/60 hover:bg-[#101618]'
                }`}
              >
                <div className="flex items-center justify-between w-full">
                  <div className="w-10 h-10 rounded-lg bg-[#141b1f] border border-border-subtle/50 flex items-center justify-center shrink-0">
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
                  <span className="text-sm font-semibold text-content-headline block leading-tight">
                    {provider.name}
                  </span>
                  <span className="text-xs text-content-dim block mt-1 leading-tight font-mono text-[11px]">
                    {provider.badge}
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
              : `${currentProvider.name} requiere una Admin API Key independiente (a nivel de organización) para reportar el consumo de tokens y costo real de la cuenta. Es opcional: sin ella seguirás viendo una estimación local.`}
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
              {currentStatus.configured && (
                <div className="flex items-center justify-between gap-2 p-2.5 rounded-lg bg-[#0d1214] border border-border-subtle/60">
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
                    className="text-[11px] text-content-dim hover:text-red-400 shrink-0 transition-colors"
                  >
                    Eliminar
                  </button>
                </div>
              )}
              {currentStatus.accountInfo && (
                <p className="text-[11px] text-content-dim leading-relaxed">
                  {currentStatus.accountInfo}
                </p>
              )}

              <div className="relative">
                <input
                  type={showApiKey ? 'text' : 'password'}
                  className="w-full bg-[#0d1214] border border-border-subtle hover:border-border-petrol focus:border-accent-primary rounded-lg pl-3.5 pr-10 py-2.5 text-sm text-content-headline font-mono outline-none transition-colors"
                  value={draftKey}
                  onChange={(e) => {
                    setDraftKey(e.target.value);
                    setValidationResult(null);
                  }}
                  placeholder={currentStatus.configured ? 'Ingresa una nueva clave para reemplazarla' : currentProvider.keyPlaceholder}
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
                  className="inline-flex items-center gap-1.5 text-xs font-semibold px-3 py-1.5 rounded-lg bg-accent-primary text-[#070a0b] hover:brightness-110 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
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
              {currentStatus.hasAdminKey && (
                <div className="flex items-center justify-between gap-2 p-2.5 rounded-lg bg-[#0d1214] border border-border-subtle/60">
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
                      setAdminKeySaved(false);
                    }}
                    className="text-[11px] text-content-dim hover:text-red-400 shrink-0 transition-colors"
                  >
                    Eliminar
                  </button>
                </div>
              )}

              <div className="relative">
                <input
                  type={showAdminKey ? 'text' : 'password'}
                  className="w-full bg-[#0d1214] border border-border-subtle hover:border-border-petrol focus:border-accent-primary rounded-lg pl-3.5 pr-10 py-2.5 text-sm text-content-headline font-mono outline-none transition-colors"
                  value={draftAdminKey}
                  onChange={(e) => {
                    setDraftAdminKey(e.target.value);
                    setAdminKeySaved(false);
                  }}
                  placeholder={currentStatus.hasAdminKey ? 'Ingresa una nueva Admin Key para reemplazarla' : 'sk-ant-admin...'}
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
                    setAdminKeySaved(true);
                    try {
                      await fetchUsage(settings.modelProvider);
                    } finally {
                      setAdminKeySaved(false);
                    }
                  }}
                  disabled={!draftAdminKey.trim()}
                  className="inline-flex items-center gap-1.5 text-xs font-semibold px-3 py-1.5 rounded-lg bg-accent-primary text-[#070a0b] hover:brightness-110 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
                >
                  Guardar Admin Key
                </button>
              </div>

              {adminKeySaved && (
                <div className="flex items-start gap-1.5 text-xs mt-0.5 text-emerald-400">
                  <IconCheck size={13} className="shrink-0 mt-0.5" />
                  <span>Admin Key guardada. Consultando consumo real...</span>
                </div>
              )}
            </div>
          </div>
        </div>
        )}
      </div>

      {/* Sección 2.5: Modelos disponibles para la cuenta configurada */}
      {currentStatus.configured && currentStatus.verified && (
        <div>
          <div className="mb-3 flex items-start justify-between gap-4">
            <div>
              <h3 className="text-xs font-semibold tracking-wider text-content-headline uppercase">
                Modelos Disponibles ({currentProvider.name})
              </h3>
              <p className="text-xs text-content-dim mt-0.5 leading-relaxed">
                Modelos reportados en vivo por la cuenta conectada con esta clave de API.
              </p>
            </div>
            {currentStatus.models && currentStatus.models.length > 0 && (
              <span className="shrink-0 text-[11px] font-mono text-content-dim bg-[#12181a] px-2.5 py-0.5 rounded border border-border-subtle/50 whitespace-nowrap">
                {currentStatus.models.length} modelos
              </span>
            )}
          </div>

          {currentStatus.models && currentStatus.models.length > 0 ? (
            (() => {
              const filteredModels = currentStatus.models.filter((id) =>
                id.toLowerCase().includes(modelFilter.trim().toLowerCase())
              );

              return (
                <div className="rounded-xl border border-border-subtle/60 bg-[#0b1012] overflow-hidden">
                  <div className="relative border-b border-border-subtle/40">
                    <IconSearch
                      size={13}
                      className="absolute left-3 top-1/2 -translate-y-1/2 text-content-dim pointer-events-none"
                    />
                    <input
                      type="text"
                      value={modelFilter}
                      onChange={(e) => setModelFilter(e.target.value)}
                      placeholder="Buscar modelo..."
                      spellCheck={false}
                      className="w-full bg-transparent pl-8 pr-3 py-2.5 text-xs text-content-headline placeholder:text-content-dim outline-none"
                    />
                  </div>

                  <div className="max-h-52 overflow-y-auto p-2.5">
                    {filteredModels.length > 0 ? (
                      <div className="grid grid-cols-1 sm:grid-cols-2 gap-1.5">
                        {filteredModels.map((modelId) => {
                          const isActive = settings.model === modelId;
                          return (
                            <button
                              key={modelId}
                              type="button"
                              onClick={() => onUpdate('model', modelId)}
                              title={modelId}
                              className={`flex items-center gap-1.5 text-xs font-mono px-2.5 py-2 rounded-lg border transition-colors text-left cursor-pointer ${
                                isActive
                                  ? 'border-accent-primary bg-accent-primary/10 text-accent-primary'
                                  : 'border-transparent bg-[#101618] text-content-dim hover:border-border-petrol/60 hover:text-content-headline'
                              }`}
                            >
                              {isActive ? (
                                <IconCheck size={12} className="shrink-0" />
                              ) : (
                                <span className="w-3 shrink-0" />
                              )}
                              <span className="truncate">{modelId}</span>
                            </button>
                          );
                        })}
                      </div>
                    ) : (
                      <p className="text-xs text-content-dim px-1 py-2">
                        Ningún modelo coincide con "{modelFilter}".
                      </p>
                    )}
                  </div>
                </div>
              );
            })()
          ) : (
            <p className="text-xs text-content-dim">
              Esta cuenta no reportó modelos disponibles.
            </p>
          )}
        </div>
      )}

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
                {settings.model || 'selecciona uno en Modelos Disponibles'}
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
                  className="w-full bg-[#0d1214] border border-border-subtle hover:border-border-petrol focus:border-accent-primary rounded-lg px-3.5 py-2.5 text-sm text-content-headline outline-none transition-colors"
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
                      <div className="p-3 rounded-lg bg-[#0d1214] border border-border-subtle/60">
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

      {/* Sección 3: Consumo de Tokens & Costo (real cuando el proveedor lo expone, estimado en caso contrario) */}
      <div>
        <div className="mb-3">
          <div className="flex items-center justify-between">
            <h3 className="text-xs font-semibold tracking-wider text-content-headline uppercase">
              {currentUsage?.available ? 'Consumo & Costo Real de la Cuenta' : 'Consumo de Tokens & Costo Estimado (Hoy)'}
            </h3>
            <span className="text-[11px] font-mono text-content-dim bg-[#12181a] px-2.5 py-0.5 rounded border border-border-subtle/50">
              {new Date().toLocaleDateString('es-ES', { weekday: 'short', day: 'numeric', month: 'short' })}
            </span>
          </div>
          <p className="text-xs text-content-dim mt-0.5 leading-relaxed">
            {currentUsage?.available
              ? `Datos reportados en vivo por la cuenta de ${currentProvider.name}.`
              : `Consumo acumulado de tokens hoy y proyección de costo calculada con las tarifas oficiales de ${currentProvider.name}.`}
          </p>
          {isLoadingUsage && (
            <p className="text-[11px] text-content-dim mt-1 flex items-center gap-1.5">
              <IconRefresh size={11} className="animate-spin" />
              Consultando datos reales de la cuenta...
            </p>
          )}
          {!isLoadingUsage && currentUsage && !currentUsage.available && currentStatus.configured && currentStatus.verified && (
            <p className="text-[11px] text-content-dim mt-1">{currentUsage.message}</p>
          )}
        </div>

        {currentUsage?.available && currentUsage.balanceText ? (
          /* Vista de saldo real (p.ej. DeepSeek): la API solo expone saldo prepagado de la cuenta,
             no un desglose de tokens por día, así que complementamos con la estimación local */
          <div className="flex flex-col gap-3">
            <div className="p-4 rounded-xl bg-[#0c1113] border border-border-subtle/60">
              <div className="flex items-center gap-2 mb-1.5">
                <IconCheck size={13} className="text-emerald-400 shrink-0" />
                <span className="text-xs text-content-dim font-medium">Saldo Real de la Cuenta ({currentProvider.name})</span>
              </div>
              <p className="text-sm font-mono text-content-headline leading-relaxed">{currentUsage.balanceText}</p>
            </div>

            <div>
              <p className="text-[11px] text-content-dim mb-2">
                {currentProvider.name} no expone un desglose de tokens por día vía API; esto es una
                estimación local basada en las peticiones realizadas desde esta app.
              </p>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div className="p-4 rounded-xl bg-[#0c1113] border border-border-subtle/60 flex flex-col justify-between">
                  <span className="text-xs text-content-dim font-medium">Tokens Consumidos Hoy (estimado)</span>
                  <div className="mt-2.5 flex items-baseline gap-1.5">
                    <span className="text-2xl font-bold font-mono text-content-headline">
                      {totalTokens.toLocaleString()}
                    </span>
                    <span className="text-xs font-mono text-accent-primary">tokens</span>
                  </div>
                  <div className="mt-3 pt-2.5 border-t border-border-subtle/40 flex items-center justify-between text-[11px] text-content-dim">
                    <span>Prompt: {todayPrompt.toLocaleString()}</span>
                    <span>Salida: {todayCompletion.toLocaleString()}</span>
                  </div>
                </div>

                <div className="p-4 rounded-xl bg-[#0c1113] border border-border-subtle/60 flex flex-col justify-between">
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-content-dim font-medium">Gasto Estimado (Hoy)</span>
                    <span className="text-[10px] font-mono text-accent-primary bg-accent-primary/10 px-1.5 py-0.5 rounded border border-accent-primary/20">
                      USD
                    </span>
                  </div>
                  <div className="mt-2.5 flex items-baseline gap-1.5">
                    <span className="text-2xl font-bold font-mono text-accent-primary">
                      {formatCost(totalCostUsd)}
                    </span>
                  </div>
                  <div className="mt-3 pt-2.5 border-t border-border-subtle/40 text-[11px] text-content-dim truncate" title={pricing.unitLabel}>
                    Tarifa: {pricing.unitLabel}
                  </div>
                </div>
              </div>
            </div>
          </div>
        ) : currentUsage?.available ? (
          /* Vista de uso real por tokens + costo (p.ej. Anthropic con Admin Key) */
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div className="p-4 rounded-xl bg-[#0c1113] border border-border-subtle/60 flex flex-col justify-between">
              <span className="text-xs text-content-dim font-medium">Tokens Consumidos Hoy (real)</span>
              <div className="mt-2.5 flex items-baseline gap-1.5">
                <span className="text-2xl font-bold font-mono text-content-headline">
                  {((currentUsage.tokensPrompt ?? 0) + (currentUsage.tokensCompletion ?? 0)).toLocaleString()}
                </span>
                <span className="text-xs font-mono text-accent-primary">tokens</span>
              </div>
              <div className="mt-3 pt-2.5 border-t border-border-subtle/40 flex items-center justify-between text-[11px] text-content-dim">
                <span>Prompt: {(currentUsage.tokensPrompt ?? 0).toLocaleString()}</span>
                <span>Salida: {(currentUsage.tokensCompletion ?? 0).toLocaleString()}</span>
              </div>
            </div>

            <div className="p-4 rounded-xl bg-[#0c1113] border border-border-subtle/60 flex flex-col justify-between">
              <div className="flex items-center justify-between">
                <span className="text-xs text-content-dim font-medium">Costo Real (Hoy)</span>
                <span className="text-[10px] font-mono text-accent-primary bg-accent-primary/10 px-1.5 py-0.5 rounded border border-accent-primary/20">
                  USD
                </span>
              </div>
              <div className="mt-2.5 flex items-baseline gap-1.5">
                <span className="text-2xl font-bold font-mono text-accent-primary">
                  {formatCost(currentUsage.costUsd ?? 0)}
                </span>
              </div>
              <div className="mt-3 pt-2.5 border-t border-border-subtle/40 text-[11px] text-content-dim">
                Reportado por el Cost Report de {currentProvider.name}
              </div>
            </div>
          </div>
        ) : (
          /* Fallback: estimación local basada en telemetría de la app y tarifas fijas */
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            {/* Card 1: Tokens Totales */}
            <div className="p-4 rounded-xl bg-[#0c1113] border border-border-subtle/60 flex flex-col justify-between">
              <span className="text-xs text-content-dim font-medium">Tokens Consumidos Hoy</span>
              <div className="mt-2.5 flex items-baseline gap-1.5">
                <span className="text-2xl font-bold font-mono text-content-headline">
                  {totalTokens.toLocaleString()}
                </span>
                <span className="text-xs font-mono text-accent-primary">tokens</span>
              </div>
              <div className="mt-3 pt-2.5 border-t border-border-subtle/40 flex items-center justify-between text-[11px] text-content-dim">
                <span>Prompt: {todayPrompt.toLocaleString()}</span>
                <span>Salida: {todayCompletion.toLocaleString()}</span>
              </div>
            </div>

            {/* Card 2: Costo Estimado en Dinero ($ USD) */}
            <div className="p-4 rounded-xl bg-[#0c1113] border border-border-subtle/60 flex flex-col justify-between">
              <div className="flex items-center justify-between">
                <span className="text-xs text-content-dim font-medium">Gasto Estimado (Hoy)</span>
                <span className="text-[10px] font-mono text-accent-primary bg-accent-primary/10 px-1.5 py-0.5 rounded border border-accent-primary/20">
                  USD
                </span>
              </div>
              <div className="mt-2.5 flex items-baseline gap-1.5">
                <span className="text-2xl font-bold font-mono text-accent-primary">
                  {formatCost(totalCostUsd)}
                </span>
              </div>
              <div className="mt-3 pt-2.5 border-t border-border-subtle/40 text-[11px] text-content-dim truncate" title={pricing.unitLabel}>
                Tarifa: {pricing.unitLabel}
              </div>
            </div>

            {/* Card 3: Distribución de Tráfico (Prompt vs Salida) */}
            <div className="p-4 rounded-xl bg-[#0c1113] border border-border-subtle/60 flex flex-col justify-between">
              <span className="text-xs text-content-dim font-medium">Distribución de Tráfico</span>
              <div className="mt-3">
                <div className="w-full h-2.5 bg-[#151c20] rounded-full overflow-hidden flex">
                  <div
                    className="bg-accent-primary h-full transition-all duration-300"
                    style={{ width: `${promptPct}%` }}
                    title={`Prompt (Entrada): ${promptPct}%`}
                  />
                  <div
                    className="bg-[#3b82f6] h-full transition-all duration-300"
                    style={{ width: `${completionPct}%` }}
                    title={`Respuesta (Salida): ${completionPct}%`}
                  />
                </div>
              </div>
              <div className="mt-3 pt-2.5 border-t border-border-subtle/40 flex items-center justify-between text-[11px] text-content-dim">
                <span className="flex items-center gap-1.5">
                  <span className="w-1.5 h-1.5 rounded-full bg-accent-primary" />
                  Prompt ({promptPct}%)
                </span>
                <span className="flex items-center gap-1.5">
                  <span className="w-1.5 h-1.5 rounded-full bg-[#3b82f6]" />
                  Salida ({completionPct}%)
                </span>
              </div>
            </div>
          </div>
        )}
      </div>

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
                <div className="w-11 h-6 bg-[#1b2328] peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-content-muted peer-checked:after:bg-[#0b0e10] after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-accent-primary"></div>
              </label>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
