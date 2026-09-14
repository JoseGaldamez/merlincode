import React, { useState, useRef, useEffect } from 'react';
import {
  IconAnthropic,
  IconOpenAI,
  IconGoogle,
  IconDeepSeek,
  IconChevronDown,
  IconCheck,
  IconCpu,
  IconSettings,
} from '../../../components/Icons';
import { AI_PROVIDERS } from '../../settings/constants';
import { ProviderStatusMap } from '../../settings/hooks/useAIProviderStatus';

interface ProviderSelectorProps {
  statuses?: ProviderStatusMap;
  selectedProviderId?: string;
  onSelectProvider?: (providerId: string) => void;
  onOpenSettings?: () => void;
  disabled?: boolean;
}

export const ProviderSelector: React.FC<ProviderSelectorProps> = ({
  statuses = {},
  selectedProviderId = 'google',
  onSelectProvider,
  onOpenSettings,
  disabled = false,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // Filtrar solo proveedores que tienen credencial configurada y verificada en el llavero
  const configuredProviders = AI_PROVIDERS.filter(
    (p) => statuses[p.id]?.configured && statuses[p.id]?.verified
  );

  const activeProvider =
    configuredProviders.find((p) => p.id === selectedProviderId) ??
    configuredProviders[0] ??
    AI_PROVIDERS.find((p) => p.id === selectedProviderId) ??
    AI_PROVIDERS[0];

  // Cerrar dropdown al hacer click afuera
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isOpen]);

  const renderLogo = (providerId: string, size = 13) => {
    switch (providerId) {
      case 'anthropic':
        return <IconAnthropic size={size} className="text-[#CC785C] shrink-0" />;
      case 'openai':
        return <IconOpenAI size={size} className="text-white shrink-0" />;
      case 'google':
        return <IconGoogle size={size} className="shrink-0" />;
      case 'deepseek':
        return <IconDeepSeek size={size} className="text-[#4D6BFE] shrink-0" />;
      default:
        return <IconCpu size={size} className="text-accent-primary shrink-0" />;
    }
  };

  // CASO 1: Ningún proveedor configurado
  if (configuredProviders.length === 0) {
    return (
      <div className="wails-no-drag flex items-center">
        <button
          type="button"
          onClick={onOpenSettings}
          className="inline-flex items-center gap-1.5 px-2 py-1 rounded bg-amber-950/40 border border-amber-500/40 text-amber-300 hover:bg-amber-950/70 text-xs font-mono transition-colors cursor-pointer"
          title="Configurar clave de API en Ajustes"
        >
          <span>⚠️ Configurar IA</span>
        </button>
      </div>
    );
  }

  // CASO 2: Exactamente 1 proveedor configurado
  if (configuredProviders.length === 1) {
    const single = configuredProviders[0];
    return (
      <div className="wails-no-drag flex items-center">
        <div
          className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded bg-[#141b1f] border border-border-subtle text-xs font-mono text-content-headline"
          title={`Proveedor activo: ${single.name}`}
        >
          {renderLogo(single.id, 13)}
          <span className="font-medium text-content-headline">{single.name}</span>
        </div>
      </div>
    );
  }

  // CASO 3: 2 o más proveedores configurados (Dropdown selector)
  return (
    <div className="wails-no-drag relative flex items-center" ref={containerRef}>
      <button
        type="button"
        onClick={() => setIsOpen((prev) => !prev)}
        disabled={disabled}
        className="inline-flex items-center gap-2 px-2.5 py-1 rounded bg-[#141b1f] border border-border-subtle hover:border-border-petrol focus:border-accent-primary text-xs font-mono text-content-headline transition-colors cursor-pointer disabled:opacity-50"
        title="Cambiar proveedor de IA"
        aria-expanded={isOpen}
      >
        {renderLogo(activeProvider.id, 13)}
        <span className="font-medium text-content-headline">{activeProvider.name}</span>
        <IconChevronDown size={11} className={`text-content-dim transition-transform duration-150 ${isOpen ? 'rotate-180' : ''}`} />
      </button>

      {isOpen && (
        <div className="absolute top-full left-0 mt-1 w-48 bg-[#12181b] border border-border-subtle rounded-md shadow-2xl py-1 z-50 flex flex-col gap-0.5">
          <div className="px-3 py-1 text-[10px] font-mono text-content-dim uppercase tracking-wider border-b border-border-subtle/60">
            Proveedor de IA
          </div>
          {configuredProviders.map((p) => {
            const isSelected = p.id === activeProvider.id;
            return (
              <button
                key={p.id}
                type="button"
                onClick={() => {
                  if (onSelectProvider) {
                    onSelectProvider(p.id);
                  }
                  setIsOpen(false);
                }}
                className={`w-full flex items-center justify-between px-3 py-1.5 text-left text-xs font-mono transition-colors cursor-pointer ${
                  isSelected
                    ? 'bg-accent-primary/10 text-accent-primary font-semibold'
                    : 'text-content-body hover:text-content-headline hover:bg-[#182328]'
                }`}
              >
                <div className="flex items-center gap-2 min-w-0">
                  {renderLogo(p.id, 13)}
                  <span className="truncate">{p.name}</span>
                </div>
                {isSelected && <IconCheck size={13} className="shrink-0 text-accent-primary" />}
              </button>
            );
          })}

          <div className="border-t border-border-subtle/60 mt-1 pt-1">
            <button
              type="button"
              onClick={() => {
                setIsOpen(false);
                if (onOpenSettings) onOpenSettings();
              }}
              className="w-full flex items-center gap-2 px-3 py-1.5 text-left text-xs font-mono text-content-dim hover:text-accent-primary hover:bg-[#182328] transition-colors cursor-pointer"
            >
              <IconSettings size={12} />
              <span>Ajustes de IA...</span>
            </button>
          </div>
        </div>
      )}
    </div>
  );
};
