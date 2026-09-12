import React, { useState, useEffect } from 'react';
import {
  IconClose,
  IconSettings,
  IconFolder,
  IconCpu,
  IconCheck,
  IconShield,
  IconKeyboard,
  IconTrash,
  IconPlus,
} from './Icons';
import { Project, AgentTelemetry } from '../types';

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  projects: Project[];
  activeProjectId: string;
  onSelectProject: (id: string) => void;
  onAddProject: (name: string) => void;
  onDeleteProject?: (id: string) => void;
  telemetry: AgentTelemetry;
  onUpdateTelemetry: (updated: Partial<AgentTelemetry>) => void;
}

type CategoryType = 'general' | 'ai' | 'projects' | 'shortcuts' | 'privacy';

interface CategoryItem {
  id: CategoryType;
  label: string;
  description: string;
  icon: React.ReactNode;
}

export const SettingsModal: React.FC<SettingsModalProps> = ({
  isOpen,
  onClose,
  projects,
  activeProjectId,
  onSelectProject,
  onAddProject,
  onDeleteProject,
  telemetry,
  onUpdateTelemetry,
}) => {
  const [activeCategory, setActiveCategory] = useState<CategoryType>('general');
  const [savedSuccess, setSavedSuccess] = useState(false);

  // Estados de configuración
  const [modelProvider, setModelProvider] = useState('merlin');
  const [model, setModel] = useState(telemetry.activeModel || 'Merlin');
  const [temperature, setTemperature] = useState(telemetry.temperature ?? 0.7);
  const [apiKey, setApiKey] = useState('');
  const [apiEndpoint, setApiEndpoint] = useState('https://api.merlincode.ai/v1');
  const [streaming, setStreaming] = useState(true);

  const [theme, setTheme] = useState('dark-flat');
  const [language, setLanguage] = useState('es');
  const [autoSaveSessions, setAutoSaveSessions] = useState(true);
  const [telemetryEnabled, setTelemetryEnabled] = useState(false);

  const [newProjectName, setNewProjectName] = useState('');

  // Escuchar tecla Escape para cerrar modal
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const handleSave = () => {
    onUpdateTelemetry({
      activeModel: model,
      temperature: Number(temperature),
    });
    setSavedSuccess(true);
    setTimeout(() => {
      setSavedSuccess(false);
      onClose();
    }, 750);
  };

  const handleCreateProject = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newProjectName.trim()) return;
    onAddProject(newProjectName.trim());
    setNewProjectName('');
  };

  const categories: CategoryItem[] = [
    {
      id: 'general',
      label: 'General',
      description: 'Apariencia, interfaz y comportamiento del sistema.',
      icon: <IconSettings size={15} />,
    },
    {
      id: 'ai',
      label: 'Modelos & IA',
      description: 'Proveedores de IA, modelos, temperatura y claves de API.',
      icon: <IconCpu size={15} />,
    },
    {
      id: 'projects',
      label: 'Proyectos',
      description: 'Gestión y conmutación de espacios de trabajo.',
      icon: <IconFolder size={15} />,
    },
    {
      id: 'shortcuts',
      label: 'Atajos de Teclado',
      description: 'Combinaciones de teclas para agilizar la interacción.',
      icon: <IconKeyboard size={15} />,
    },
    {
      id: 'privacy',
      label: 'Privacidad & Datos',
      description: 'Almacenamiento local, caché y recopilación de datos.',
      icon: <IconShield size={15} />,
    },
  ];

  const currentCategoryObj = categories.find((c) => c.id === activeCategory)!;

  return (
    <div
      className="fixed inset-0 bg-black/75 backdrop-blur-[2px] flex items-center justify-center z-50 animate-fadeIn"
      onClick={onClose}
    >
      <div
        className="bg-[#111619] border border-border-subtle rounded-lg w-[90vw] max-w-[1280px] h-[88vh] max-h-[800px] min-h-[540px] flex flex-col shadow-2xl overflow-hidden animate-scaleIn select-none"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        {/* Modal Top Header */}
        <div className="flex items-center justify-between px-5 py-3 border-b border-border-subtle bg-[#0e1315] shrink-0">
          <div className="flex items-center gap-2.5 text-accent-primary">
            <IconSettings size={16} />
            <h3 className="text-sm font-semibold text-content-headline tracking-tight font-sans">
              Configuración
            </h3>
          </div>
          <button
            className="p-1 text-content-dim hover:text-content-headline hover:bg-[#182328] rounded transition-colors cursor-pointer"
            onClick={onClose}
            title="Cerrar configuración"
            aria-label="Cerrar"
          >
            <IconClose size={15} />
          </button>
        </div>

        {/* Modal Main Split: Left Sidebar & Right Content */}
        <div className="flex flex-1 overflow-hidden min-h-0">
          {/* Left Categories Sidebar */}
          <nav
            className="w-[250px] min-w-[250px] bg-[#0d1214] border-r border-border-subtle flex flex-col p-3 overflow-y-auto shrink-0"
            aria-label="Categorías de configuración"
          >
            <div className="px-2 py-1 mb-1">
              <span className="text-[10px] font-mono font-bold tracking-widest text-content-dim uppercase">
                CATEGORÍAS
              </span>
            </div>

            <ul className="flex flex-col gap-1 list-none p-0 m-0">
              {categories.map((cat) => {
                const isActive = cat.id === activeCategory;
                return (
                  <li key={cat.id}>
                    <button
                      className={`w-full flex items-center gap-2.5 px-3 py-2 rounded text-xs transition-all cursor-pointer text-left border ${
                        isActive
                          ? 'bg-[#18262c] border-border-petrol text-accent-primary font-medium shadow-sm'
                          : 'text-content-muted hover:bg-[#141b1f] hover:text-content-body border-transparent'
                      }`}
                      onClick={() => setActiveCategory(cat.id)}
                    >
                      <span className="shrink-0">{cat.icon}</span>
                      <span className="flex-1 truncate">{cat.label}</span>
                      {cat.id === 'projects' && (
                        <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-[#1f3037] text-accent-primary">
                          {projects.length}
                        </span>
                      )}
                    </button>
                  </li>
                );
              })}
            </ul>
          </nav>

          {/* Right Content Panel */}
          <main className="flex-1 flex flex-col bg-[#111619] overflow-y-auto p-6 md:p-8 min-w-0">
            <div className="mb-6 pb-3 border-b border-border-subtle shrink-0">
              <h2 className="text-lg font-semibold text-content-headline mb-1 font-sans">
                {currentCategoryObj.label}
              </h2>
              <p className="text-xs text-content-dim leading-relaxed">
                {currentCategoryObj.description}
              </p>
            </div>

            <div className="flex flex-col gap-4">
              {/* CATEGORÍA 1: GENERAL */}
              {activeCategory === 'general' && (
                <div className="flex flex-col gap-4">
                  <div className="p-4 rounded bg-[#13191d] border border-border-subtle flex flex-col gap-3 shadow-sm">
                    <div>
                      <h4 className="text-xs font-semibold text-content-headline">Tema Visual</h4>
                      <p className="text-[11px] text-content-dim leading-normal mt-0.5">
                        Estilo estético para los paneles y la barra superior frameless.
                      </p>
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-2.5">
                      <label
                        className={`flex items-start gap-2.5 p-2.5 rounded bg-[#0f1416] border transition-all cursor-pointer ${
                          theme === 'dark-flat'
                            ? 'border-accent-primary bg-[#132126]'
                            : 'border-border-subtle hover:border-border-petrol'
                        }`}
                      >
                        <input
                          type="radio"
                          name="theme"
                          value="dark-flat"
                          checked={theme === 'dark-flat'}
                          onChange={() => setTheme('dark-flat')}
                          className="mt-0.5 accent-accent-primary"
                        />
                        <div>
                          <div className="text-xs font-medium text-content-headline">
                            Oscuro Flat (DESIGN.md)
                          </div>
                          <div className="text-[11px] text-content-dim mt-0.5">
                            Superficies pizarra, bordes mínimos y acento cian.
                          </div>
                        </div>
                      </label>

                      <label
                        className={`flex items-start gap-2.5 p-2.5 rounded bg-[#0f1416] border transition-all cursor-pointer ${
                          theme === 'deep-black'
                            ? 'border-accent-primary bg-[#132126]'
                            : 'border-border-subtle hover:border-border-petrol'
                        }`}
                      >
                        <input
                          type="radio"
                          name="theme"
                          value="deep-black"
                          checked={theme === 'deep-black'}
                          onChange={() => setTheme('deep-black')}
                          className="mt-0.5 accent-accent-primary"
                        />
                        <div>
                          <div className="text-xs font-medium text-content-headline">
                            Negro Profundo (OLED)
                          </div>
                          <div className="text-[11px] text-content-dim mt-0.5">
                            Fondos ultra-oscuros con contraste absoluto.
                          </div>
                        </div>
                      </label>
                    </div>
                  </div>

                  <div className="p-4 rounded bg-[#13191d] border border-border-subtle flex flex-col gap-3 shadow-sm">
                    <div>
                      <h4 className="text-xs font-semibold text-content-headline">Idioma de la Interfaz</h4>
                      <p className="text-[11px] text-content-dim leading-normal mt-0.5">
                        Selecciona el idioma principal para los textos de la aplicación.
                      </p>
                    </div>
                    <select
                      className="w-full max-w-xs bg-[#0f1416] border border-border-subtle rounded px-2.5 py-1.5 text-xs text-content-headline outline-none focus:border-accent-primary transition-colors cursor-pointer"
                      value={language}
                      onChange={(e) => setLanguage(e.target.value)}
                    >
                      <option value="es">Español (Latinoamérica)</option>
                      <option value="en">English (US)</option>
                    </select>
                  </div>

                  <div className="p-4 rounded bg-[#13191d] border border-border-subtle flex flex-col gap-3 shadow-sm">
                    <div>
                      <h4 className="text-xs font-semibold text-content-headline">Ventana y Paneles</h4>
                      <p className="text-[11px] text-content-dim leading-normal mt-0.5">
                        Comportamiento de la interfaz de usuario en el escritorio.
                      </p>
                    </div>
                    <div className="flex flex-col gap-2">
                      <div className="flex items-center justify-between p-2.5 bg-[#0f1416] rounded border border-border-subtle">
                        <div>
                          <span className="text-xs font-medium text-content-headline block">
                            Recordar estado de paneles laterales
                          </span>
                          <span className="text-[11px] text-content-dim block">
                            Persistir automáticamente paneles colapsados entre inicios.
                          </span>
                        </div>
                        <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-[#16242a] text-accent-primary">
                          Activo
                        </span>
                      </div>

                      <div className="flex items-center justify-between p-2.5 bg-[#0f1416] rounded border border-border-subtle">
                        <div>
                          <span className="text-xs font-medium text-content-headline block">
                            Guardar dimensiones de ventana
                          </span>
                          <span className="text-[11px] text-content-dim block">
                            Guarda el tamaño de la ventana al redimensionar en tiempo real.
                          </span>
                        </div>
                        <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-[#16242a] text-accent-primary">
                          Activo
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {/* CATEGORÍA 2: MODELOS & IA */}
              {activeCategory === 'ai' && (
                <div className="flex flex-col gap-4">
                  <div className="p-4 rounded bg-[#13191d] border border-border-subtle flex flex-col gap-3 shadow-sm">
                    <div>
                      <h4 className="text-xs font-semibold text-content-headline">Proveedor de Inteligencia</h4>
                      <p className="text-[11px] text-content-dim leading-normal mt-0.5">
                        El motor o backend que procesa las instrucciones y solicitudes.
                      </p>
                    </div>
                    <select
                      className="w-full max-w-xs bg-[#0f1416] border border-border-subtle rounded px-2.5 py-1.5 text-xs text-content-headline outline-none focus:border-accent-primary transition-colors cursor-pointer"
                      value={modelProvider}
                      onChange={(e) => setModelProvider(e.target.value)}
                    >
                      <option value="merlin">Merlin Neural Core (Nativo)</option>
                      <option value="openai">OpenAI (GPT-4o, o1, etc.)</option>
                      <option value="claude">Anthropic (Claude 3.5 Sonnet)</option>
                      <option value="ollama">Local LLM / Ollama (Offline)</option>
                    </select>
                  </div>

                  <div className="p-4 rounded bg-[#13191d] border border-border-subtle flex flex-col gap-3 shadow-sm">
                    <div>
                      <h4 className="text-xs font-semibold text-content-headline">Identificador del Modelo</h4>
                      <p className="text-[11px] text-content-dim leading-normal mt-0.5">
                        Nombre del checkpoint o identificador exacto del modelo activo.
                      </p>
                    </div>
                    <input
                      type="text"
                      className="w-full max-w-md bg-[#0f1416] border border-border-subtle rounded px-2.5 py-1.5 text-xs text-content-headline font-mono outline-none focus:border-accent-primary transition-colors"
                      value={model}
                      onChange={(e) => setModel(e.target.value)}
                      placeholder="e.g. Merlin-Pro, gpt-4o, claude-3-5-sonnet"
                    />
                  </div>

                  <div className="p-4 rounded bg-[#13191d] border border-border-subtle flex flex-col gap-3 shadow-sm">
                    <div>
                      <h4 className="text-xs font-semibold text-content-headline">Clave de API (API Key)</h4>
                      <p className="text-[11px] text-content-dim leading-normal mt-0.5">
                        Tu credencial de autenticación (se almacena cifrada en almacenamiento local).
                      </p>
                    </div>
                    <input
                      type="password"
                      className="w-full max-w-md bg-[#0f1416] border border-border-subtle rounded px-2.5 py-1.5 text-xs text-content-headline font-mono outline-none focus:border-accent-primary transition-colors"
                      value={apiKey}
                      onChange={(e) => setApiKey(e.target.value)}
                      placeholder="sk-••••••••••••••••••••••••"
                    />
                  </div>

                  <div className="p-4 rounded bg-[#13191d] border border-border-subtle flex flex-col gap-3 shadow-sm">
                    <div>
                      <h4 className="text-xs font-semibold text-content-headline">Endpoint / Base URL</h4>
                      <p className="text-[11px] text-content-dim leading-normal mt-0.5">
                        Dirección del servicio API para inferencia personalizada o servidores locales.
                      </p>
                    </div>
                    <input
                      type="text"
                      className="w-full max-w-md bg-[#0f1416] border border-border-subtle rounded px-2.5 py-1.5 text-xs text-content-headline font-mono outline-none focus:border-accent-primary transition-colors"
                      value={apiEndpoint}
                      onChange={(e) => setApiEndpoint(e.target.value)}
                      placeholder="https://api.merlincode.ai/v1"
                    />
                  </div>

                  <div className="p-4 rounded bg-[#13191d] border border-border-subtle flex flex-col gap-3 shadow-sm">
                    <div className="flex items-center justify-between">
                      <div>
                        <h4 className="text-xs font-semibold text-content-headline">Temperatura de Generación</h4>
                        <p className="text-[11px] text-content-dim leading-normal mt-0.5">
                          Valores bajos aumentan el determinismo; valores altos la creatividad.
                        </p>
                      </div>
                      <span className="text-xs font-mono font-bold px-2 py-0.5 rounded bg-[#1b272d] text-accent-primary">
                        {temperature}
                      </span>
                    </div>
                    <input
                      type="range"
                      min="0"
                      max="1"
                      step="0.05"
                      value={temperature}
                      onChange={(e) => setTemperature(parseFloat(e.target.value))}
                      className="w-full accent-accent-primary cursor-pointer mt-1"
                    />
                  </div>

                  <div className="p-4 rounded bg-[#13191d] border border-border-subtle flex flex-col gap-3 shadow-sm">
                    <div className="flex items-center justify-between">
                      <div>
                        <h4 className="text-xs font-semibold text-content-headline">Streaming de Tokens</h4>
                        <p className="text-[11px] text-content-dim leading-normal mt-0.5">
                          Mostrar la respuesta palabra por palabra a medida que el modelo genera.
                        </p>
                      </div>
                      <label className="relative inline-flex items-center cursor-pointer">
                        <input
                          type="checkbox"
                          checked={streaming}
                          onChange={(e) => setStreaming(e.target.checked)}
                          className="sr-only peer"
                        />
                        <div className="w-9 h-5 bg-[#1f292e] peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-content-muted peer-checked:after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-accent-primary"></div>
                      </label>
                    </div>
                  </div>
                </div>
              )}

              {/* CATEGORÍA 3: PROYECTOS */}
              {activeCategory === 'projects' && (
                <div className="flex flex-col gap-4">
                  <div className="p-4 rounded bg-[#13191d] border border-border-subtle flex flex-col gap-3 shadow-sm">
                    <div>
                      <h4 className="text-xs font-semibold text-content-headline">Espacios de Trabajo Guardados</h4>
                      <p className="text-[11px] text-content-dim leading-normal mt-0.5">
                        Selecciona el proyecto activo o administra tus espacios de desarrollo.
                      </p>
                    </div>
                    <div className="flex flex-col gap-1.5">
                      {projects.map((proj) => {
                        const isCurrent = proj.id === activeProjectId;
                        return (
                          <div
                            key={proj.id}
                            className={`flex items-center justify-between p-2.5 rounded border transition-colors ${
                              isCurrent
                                ? 'bg-[#18262c] border-border-petrol'
                                : 'bg-[#0f1416] border-border-subtle hover:border-border-petrol'
                            }`}
                          >
                            <div className="flex items-center gap-2.5">
                              <IconFolder
                                size={14}
                                className={isCurrent ? 'text-accent-primary' : 'text-content-dim'}
                              />
                              <span className="text-xs font-mono font-medium text-content-headline">
                                {proj.name}
                              </span>
                              {isCurrent && (
                                <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-[#1c282e] text-accent-primary">
                                  Activo
                                </span>
                              )}
                            </div>

                            <div className="flex items-center gap-2">
                              {!isCurrent && (
                                <button
                                  className="text-xs text-accent-primary hover:underline cursor-pointer"
                                  onClick={() => onSelectProject(proj.id)}
                                >
                                  Activar
                                </button>
                              )}
                              {onDeleteProject && projects.length > 1 && (
                                <button
                                  className="p-1 text-content-dim hover:text-status-error hover:bg-[#25181a] rounded transition-colors cursor-pointer"
                                  onClick={() => onDeleteProject(proj.id)}
                                  title="Eliminar proyecto"
                                >
                                  <IconTrash size={13} />
                                </button>
                              )}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  </div>

                  <div className="p-4 rounded bg-[#13191d] border border-border-subtle flex flex-col gap-3 shadow-sm">
                    <div>
                      <h4 className="text-xs font-semibold text-content-headline">Crear Nuevo Proyecto</h4>
                      <p className="text-[11px] text-content-dim leading-normal mt-0.5">
                        Agrega un nuevo contexto o espacio de trabajo a tu entorno de desarrollo.
                      </p>
                    </div>
                    <form onSubmit={handleCreateProject} className="flex gap-2 max-w-md">
                      <input
                        type="text"
                        className="flex-1 bg-[#0f1416] border border-border-subtle rounded px-2.5 py-1.5 text-xs text-content-headline font-mono outline-none focus:border-accent-primary transition-colors"
                        value={newProjectName}
                        onChange={(e) => setNewProjectName(e.target.value)}
                        placeholder="Nombre del proyecto..."
                      />
                      <button
                        type="submit"
                        disabled={!newProjectName.trim()}
                        className="flex items-center gap-1.5 px-3 py-1.5 bg-accent-primary hover:bg-accent-primary-hover disabled:opacity-40 text-[#0b0e10] font-semibold text-xs rounded transition-colors cursor-pointer shrink-0"
                      >
                        <IconPlus size={13} />
                        <span>Crear</span>
                      </button>
                    </form>
                  </div>
                </div>
              )}

              {/* CATEGORÍA 4: ATAJOS DE TECLADO */}
              {activeCategory === 'shortcuts' && (
                <div className="flex flex-col gap-4">
                  <div className="p-4 rounded bg-[#13191d] border border-border-subtle flex flex-col gap-3 shadow-sm">
                    <div>
                      <h4 className="text-xs font-semibold text-content-headline">Comandos del Teclado</h4>
                      <p className="text-[11px] text-content-dim leading-normal mt-0.5">
                        Atajos directos para interactuar con el flujo de trabajo de Merlin Code.
                      </p>
                    </div>
                    <div className="flex flex-col gap-1.5">
                      <div className="flex items-center justify-between p-2 rounded bg-[#0f1416] border border-border-subtle text-xs">
                        <span className="text-content-body">Enviar mensaje / instrucción</span>
                        <kbd className="px-2 py-0.5 rounded bg-[#172227] border border-[#283941] text-accent-primary text-[11px] font-mono shadow-sm">
                          Enter
                        </kbd>
                      </div>

                      <div className="flex items-center justify-between p-2 rounded bg-[#0f1416] border border-border-subtle text-xs">
                        <span className="text-content-body">Salto de línea en el campo de texto</span>
                        <kbd className="px-2 py-0.5 rounded bg-[#172227] border border-[#283941] text-accent-primary text-[11px] font-mono shadow-sm">
                          Shift + Enter
                        </kbd>
                      </div>

                      <div className="flex items-center justify-between p-2 rounded bg-[#0f1416] border border-border-subtle text-xs">
                        <span className="text-content-body">Buscar en el historial de sesiones</span>
                        <kbd className="px-2 py-0.5 rounded bg-[#172227] border border-[#283941] text-accent-primary text-[11px] font-mono shadow-sm">
                          Ctrl + F
                        </kbd>
                      </div>

                      <div className="flex items-center justify-between p-2 rounded bg-[#0f1416] border border-border-subtle text-xs">
                        <span className="text-content-body">Abrir Configuración</span>
                        <kbd className="px-2 py-0.5 rounded bg-[#172227] border border-[#283941] text-accent-primary text-[11px] font-mono shadow-sm">
                          Ctrl + ,
                        </kbd>
                      </div>

                      <div className="flex items-center justify-between p-2 rounded bg-[#0f1416] border border-border-subtle text-xs">
                        <span className="text-content-body">Cerrar diálogo o modal</span>
                        <kbd className="px-2 py-0.5 rounded bg-[#172227] border border-[#283941] text-accent-primary text-[11px] font-mono shadow-sm">
                          Esc
                        </kbd>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {/* CATEGORÍA 5: PRIVACIDAD & DATOS */}
              {activeCategory === 'privacy' && (
                <div className="flex flex-col gap-4">
                  <div className="p-4 rounded bg-[#13191d] border border-border-subtle flex flex-col gap-3 shadow-sm">
                    <div>
                      <h4 className="text-xs font-semibold text-content-headline">Almacenamiento Local</h4>
                      <p className="text-[11px] text-content-dim leading-normal mt-0.5">
                        Tus datos se mantienen en tu máquina local dentro del directorio de usuario:
                      </p>
                      <code className="inline-block font-mono text-[11px] px-2 py-1 bg-[#0b0e10] border border-border-subtle text-accent-primary rounded mt-1.5">
                        %APPDATA%/merlincode/window.json
                      </code>
                    </div>
                  </div>

                  <div className="p-4 rounded bg-[#13191d] border border-border-subtle flex flex-col gap-3 shadow-sm">
                    <div className="flex items-center justify-between">
                      <div>
                        <h4 className="text-xs font-semibold text-content-headline">Auto-guardado de Sesiones</h4>
                        <p className="text-[11px] text-content-dim leading-normal mt-0.5">
                          Guardar automáticamente los mensajes de cada sesión localmente.
                        </p>
                      </div>
                      <label className="relative inline-flex items-center cursor-pointer">
                        <input
                          type="checkbox"
                          checked={autoSaveSessions}
                          onChange={(e) => setAutoSaveSessions(e.target.checked)}
                          className="sr-only peer"
                        />
                        <div className="w-9 h-5 bg-[#1f292e] peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-content-muted peer-checked:after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-accent-primary"></div>
                      </label>
                    </div>
                  </div>

                  <div className="p-4 rounded bg-[#13191d] border border-border-subtle flex flex-col gap-3 shadow-sm">
                    <div className="flex items-center justify-between">
                      <div>
                        <h4 className="text-xs font-semibold text-content-headline">Telemetría y Diagnósticos</h4>
                        <p className="text-[11px] text-content-dim leading-normal mt-0.5">
                          Enviar reportes anónimos de errores para mejorar la estabilidad.
                        </p>
                      </div>
                      <label className="relative inline-flex items-center cursor-pointer">
                        <input
                          type="checkbox"
                          checked={telemetryEnabled}
                          onChange={(e) => setTelemetryEnabled(e.target.checked)}
                          className="sr-only peer"
                        />
                        <div className="w-9 h-5 bg-[#1f292e] peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-content-muted peer-checked:after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-accent-primary"></div>
                      </label>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </main>
        </div>

        {/* Modal Bottom Footer */}
        <div className="flex items-center justify-between px-6 py-3 border-t border-border-subtle bg-[#0e1315] shrink-0">
          <div className="flex items-center">
            {savedSuccess && (
              <span className="flex items-center gap-1.5 text-xs text-status-success font-medium animate-fadeIn">
                <IconCheck size={14} />
                <span>¡Cambios guardados con éxito!</span>
              </span>
            )}
          </div>
          <div className="flex items-center gap-2.5">
            <button
              className="px-3.5 py-1.5 rounded border border-border-subtle hover:border-border-petrol text-xs text-content-muted hover:text-content-headline transition-colors cursor-pointer"
              onClick={onClose}
            >
              Cancelar
            </button>
            <button
              className="flex items-center gap-1.5 px-4 py-1.5 rounded bg-accent-primary hover:bg-accent-primary-hover text-[#0b0e10] font-semibold text-xs transition-colors cursor-pointer shadow-sm"
              onClick={handleSave}
            >
              <IconCheck size={14} />
              <span>Guardar Cambios</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
