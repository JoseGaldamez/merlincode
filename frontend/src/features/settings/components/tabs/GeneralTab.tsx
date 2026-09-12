import React from 'react';
import { SettingsState } from '../../types';

interface GeneralTabProps {
  settings: SettingsState;
  onUpdate: <K extends keyof SettingsState>(key: K, value: SettingsState[K]) => void;
}

export const GeneralTab: React.FC<GeneralTabProps> = ({ settings, onUpdate }) => {
  return (
    <div className="flex flex-col gap-9 max-w-3xl">
      {/* Sección 1: Apariencia */}
      <div>
        <div className="mb-4">
          <h3 className="text-xs font-mono font-semibold tracking-wider text-accent-primary uppercase">
            Apariencia
          </h3>
          <p className="text-xs text-content-dim mt-1 leading-relaxed">
            Personaliza el tema visual y la paleta de colores de la interfaz.
          </p>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
          {/* Opción 1: Dark Flat */}
          <button
            type="button"
            onClick={() => onUpdate('theme', 'dark-flat')}
            className={`p-4 rounded-lg border text-left transition-all cursor-pointer flex flex-col justify-between ${
              settings.theme === 'dark-flat'
                ? 'border-accent-primary bg-[#132228] shadow-sm'
                : 'border-border-subtle hover:border-border-petrol bg-[#0d1214]'
            }`}
          >
            <div className="flex items-center justify-between mb-2.5">
              <span className="text-sm font-semibold text-content-headline">
                Oscuro Flat (Predeterminado)
              </span>
              <span
                className={`w-3.5 h-3.5 rounded-full border flex items-center justify-center ${
                  settings.theme === 'dark-flat'
                    ? 'border-accent-primary'
                    : 'border-content-dim'
                }`}
              >
                {settings.theme === 'dark-flat' && (
                  <span className="w-2 h-2 rounded-full bg-accent-primary" />
                )}
              </span>
            </div>
            <p className="text-xs text-content-dim leading-relaxed">
              Superficies de pizarra mate con detalles cian y bordes sutiles de bajo contraste.
            </p>
          </button>

          {/* Opción 2: Deep Black */}
          <button
            type="button"
            onClick={() => onUpdate('theme', 'deep-black')}
            className={`p-4 rounded-lg border text-left transition-all cursor-pointer flex flex-col justify-between ${
              settings.theme === 'deep-black'
                ? 'border-accent-primary bg-[#132228] shadow-sm'
                : 'border-border-subtle hover:border-border-petrol bg-[#0d1214]'
            }`}
          >
            <div className="flex items-center justify-between mb-2.5">
              <span className="text-sm font-semibold text-content-headline">
                Negro Profundo (OLED)
              </span>
              <span
                className={`w-3.5 h-3.5 rounded-full border flex items-center justify-center ${
                  settings.theme === 'deep-black'
                    ? 'border-accent-primary'
                    : 'border-content-dim'
                }`}
              >
                {settings.theme === 'deep-black' && (
                  <span className="w-2 h-2 rounded-full bg-accent-primary" />
                )}
              </span>
            </div>
            <p className="text-xs text-content-dim leading-relaxed">
              Fondos ultra-oscuros con contraste absoluto pensados para pantallas OLED.
            </p>
          </button>
        </div>
      </div>

      {/* Sección 2: Idioma & Región */}
      <div>
        <div className="mb-4">
          <h3 className="text-xs font-mono font-semibold tracking-wider text-accent-primary uppercase">
            Idioma & Región
          </h3>
          <p className="text-xs text-content-dim mt-1 leading-relaxed">
            Selecciona el idioma principal para los textos y menús de la aplicación.
          </p>
        </div>

        <div className="divide-y divide-border-subtle/40 border-t border-b border-border-subtle/40">
          <div className="py-4 flex flex-col md:flex-row md:items-center justify-between gap-4">
            <div className="max-w-md">
              <label className="text-sm font-medium text-content-headline block">
                Idioma de la Interfaz
              </label>
              <span className="text-xs text-content-dim block mt-0.5 leading-relaxed">
                Cambia los textos descriptivos y opciones de todo el sistema.
              </span>
            </div>
            <div className="w-full md:w-80 shrink-0">
              <select
                className="w-full bg-[#0d1214] border border-border-subtle hover:border-border-petrol focus:border-accent-primary rounded-lg px-3.5 py-2 text-sm text-content-headline outline-none transition-colors cursor-pointer"
                value={settings.language}
                onChange={(e) => onUpdate('language', e.target.value)}
              >
                <option value="es">Español (Latinoamérica)</option>
                <option value="en">English (US)</option>
              </select>
            </div>
          </div>
        </div>
      </div>

      {/* Sección 3: Ventana y Paneles */}
      <div>
        <div className="mb-4">
          <h3 className="text-xs font-mono font-semibold tracking-wider text-accent-primary uppercase">
            Ventana y Paneles
          </h3>
          <p className="text-xs text-content-dim mt-1 leading-relaxed">
            Comportamiento de la interfaz de usuario en el escritorio.
          </p>
        </div>

        <div className="divide-y divide-border-subtle/40 border-t border-b border-border-subtle/40">
          <div className="py-4 flex items-center justify-between gap-4">
            <div className="max-w-md">
              <span className="text-sm font-medium text-content-headline block">
                Recordar estado de paneles laterales
              </span>
              <span className="text-xs text-content-dim block mt-0.5 leading-relaxed">
                Persistir automáticamente paneles colapsados o expandidos entre inicios.
              </span>
            </div>
            <span className="text-xs font-mono px-2.5 py-1 rounded bg-[#152329] text-accent-primary font-medium shrink-0 border border-border-petrol/50">
              Activo
            </span>
          </div>

          <div className="py-4 flex items-center justify-between gap-4">
            <div className="max-w-md">
              <span className="text-sm font-medium text-content-headline block">
                Guardar dimensiones de ventana
              </span>
              <span className="text-xs text-content-dim block mt-0.5 leading-relaxed">
                Almacena el ancho, alto y modo maximizado al redimensionar en tiempo real.
              </span>
            </div>
            <span className="text-xs font-mono px-2.5 py-1 rounded bg-[#152329] text-accent-primary font-medium shrink-0 border border-border-petrol/50">
              Activo
            </span>
          </div>
        </div>
      </div>
    </div>
  );
};
