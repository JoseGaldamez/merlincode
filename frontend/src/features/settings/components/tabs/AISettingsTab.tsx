import React, { useState, useEffect } from 'react';
import { SettingsState } from '../../types';
import { AgentTelemetry } from '../../../../types';
import { AI_PROVIDERS, getProviderConfig } from '../../constants';
import { BrowserOpenURL } from '../../../../../wailsjs/runtime/runtime';
import {
  IconAnthropic,
  IconOpenAI,
  IconGoogle,
  IconDeepSeek,
  IconEye,
  IconEyeOff,
  IconExternalLink,
  IconShield,
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

  const currentProvider = getProviderConfig(settings.modelProvider);

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

      {/* Sección 2: Clave de API (Subida directamente debajo de los proveedores) */}
      <div>
        <div className="mb-3">
          <h3 className="text-xs font-semibold tracking-wider text-content-headline uppercase">
            Clave de API ({currentProvider.name})
          </h3>
          <p className="text-xs text-content-dim mt-0.5 leading-relaxed">
            Credencial requerida para autenticar peticiones directamente con {currentProvider.name}.
          </p>
        </div>

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
              <div className="relative">
                <input
                  type={showApiKey ? 'text' : 'password'}
                  className="w-full bg-[#0d1214] border border-border-subtle hover:border-border-petrol focus:border-accent-primary rounded-lg pl-3.5 pr-10 py-2.5 text-sm text-content-headline font-mono outline-none transition-colors"
                  value={settings.apiKey}
                  onChange={(e) => onUpdate('apiKey', e.target.value)}
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

              <button
                type="button"
                onClick={() => {
                  try {
                    BrowserOpenURL(currentProvider.keyHelpUrl);
                  } catch {
                    window.open(currentProvider.keyHelpUrl, '_blank', 'noopener,noreferrer');
                  }
                }}
                className="inline-flex items-center gap-1.5 text-xs text-accent-primary hover:underline transition-colors self-start mt-0.5 cursor-pointer"
              >
                <span>Obtener clave en {currentProvider.name} Console</span>
                <IconExternalLink size={12} />
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Sección 3: Consumo de Tokens & Costo Estimado (Usage de Hoy) */}
      <div>
        <div className="mb-3">
          <div className="flex items-center justify-between">
            <h3 className="text-xs font-semibold tracking-wider text-content-headline uppercase">
              Consumo de Tokens & Costo Estimado (Hoy)
            </h3>
            <span className="text-[11px] font-mono text-content-dim bg-[#12181a] px-2.5 py-0.5 rounded border border-border-subtle/50">
              {new Date().toLocaleDateString('es-ES', { weekday: 'short', day: 'numeric', month: 'short' })}
            </span>
          </div>
          <p className="text-xs text-content-dim mt-0.5 leading-relaxed">
            Consumo acumulado de tokens hoy y proyección de costo calculada con las tarifas oficiales de {currentProvider.name}.
          </p>
        </div>

        {/* Tarjetas de métricas de Usage */}
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
