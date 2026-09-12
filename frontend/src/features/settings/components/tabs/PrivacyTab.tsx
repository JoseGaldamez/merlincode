import React from 'react';
import { SettingsState } from '../../types';

interface PrivacyTabProps {
  settings: SettingsState;
  onUpdate: <K extends keyof SettingsState>(key: K, value: SettingsState[K]) => void;
}

export const PrivacyTab: React.FC<PrivacyTabProps> = ({ settings, onUpdate }) => {
  return (
    <div className="flex flex-col gap-9 max-w-3xl">
      <div>
        <div className="mb-4">
          <h3 className="text-xs font-mono font-semibold tracking-wider text-accent-primary uppercase">
            Almacenamiento & Telemetría
          </h3>
          <p className="text-xs text-content-dim mt-1 leading-relaxed">
            Control de la persistencia local de datos y diagnóstico del sistema.
          </p>
        </div>

        <div className="divide-y divide-border-subtle/40 border-t border-b border-border-subtle/40">
          {/* Ubicación de datos */}
          <div className="py-4 flex flex-col md:flex-row md:items-center justify-between gap-4">
            <div className="max-w-md">
              <label className="text-sm font-medium text-content-headline block">
                Almacenamiento Local
              </label>
              <span className="text-xs text-content-dim block mt-0.5 leading-relaxed">
                Tus configuraciones, historial y estado se guardan en tu equipo.
              </span>
            </div>
            <div className="shrink-0">
              <code className="inline-block font-mono text-xs px-3 py-1.5 bg-[#0d1214] border border-border-subtle text-accent-primary rounded-lg">
                %APPDATA%/merlincode/window.json
              </code>
            </div>
          </div>

          {/* Auto-guardado */}
          <div className="py-4 flex items-center justify-between gap-4">
            <div className="max-w-md">
              <label className="text-sm font-medium text-content-headline block">
                Auto-guardado de Sesiones
              </label>
              <span className="text-xs text-content-dim block mt-0.5 leading-relaxed">
                Almacenar automáticamente los mensajes y el historial en tu disco local.
              </span>
            </div>
            <label className="relative inline-flex items-center cursor-pointer shrink-0">
              <input
                type="checkbox"
                checked={settings.autoSaveSessions}
                onChange={(e) => onUpdate('autoSaveSessions', e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-[#1b2328] peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-content-muted peer-checked:after:bg-[#0b0e10] after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-accent-primary"></div>
            </label>
          </div>

          {/* Telemetría */}
          <div className="py-4 flex items-center justify-between gap-4">
            <div className="max-w-md">
              <label className="text-sm font-medium text-content-headline block">
                Telemetría y Diagnósticos
              </label>
              <span className="text-xs text-content-dim block mt-0.5 leading-relaxed">
                Enviar métricas anónimas para mejorar el rendimiento y la estabilidad.
              </span>
            </div>
            <label className="relative inline-flex items-center cursor-pointer shrink-0">
              <input
                type="checkbox"
                checked={settings.telemetryEnabled}
                onChange={(e) => onUpdate('telemetryEnabled', e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-[#1b2328] peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-content-muted peer-checked:after:bg-[#0b0e10] after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-accent-primary"></div>
            </label>
          </div>
        </div>
      </div>
    </div>
  );
};
