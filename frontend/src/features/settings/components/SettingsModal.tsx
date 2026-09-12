import React, { useState, useEffect } from 'react';
import {
  IconClose,
  IconSettings,
  IconFolder,
  IconCpu,
  IconCheck,
  IconShield,
  IconKeyboard,
} from '../../../components/Icons';
import { CategoryItem, SettingsCategoryType, SettingsModalProps } from '../types';
import { useSettingsForm } from '../hooks/useSettingsForm';
import { SettingsNav } from './SettingsNav';
import { GeneralTab } from './tabs/GeneralTab';
import { AISettingsTab } from './tabs/AISettingsTab';
import { ProjectsTab } from './tabs/ProjectsTab';
import { ShortcutsTab } from './tabs/ShortcutsTab';
import { PrivacyTab } from './tabs/PrivacyTab';

export const SettingsModal: React.FC<SettingsModalProps> = ({
  isOpen,
  onClose,
  projects,
  activeProjectId,
  onSelectProject,
  onAddProject,
  onDeleteProject,
  onOpenFolder,
  telemetry,
  onUpdateTelemetry,
}) => {
  const [activeCategory, setActiveCategory] = useState<SettingsCategoryType>('ai');
  const { settings, updateSetting, handleSave, savedSuccess } = useSettingsForm(
    telemetry,
    onUpdateTelemetry,
    onClose
  );

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

  const categories: CategoryItem[] = [
    {
      id: 'ai',
      label: 'Modelos & IA',
      description: 'Configuración de proveedores de IA, checkpoints y parámetros de inferencia.',
      icon: <IconCpu size={18} />,
    },
    {
      id: 'general',
      label: 'General',
      description: 'Tema visual, idioma y comportamiento de la interfaz de usuario.',
      icon: <IconSettings size={18} />,
    },
    {
      id: 'projects',
      label: 'Proyectos',
      description: 'Gestión y conmutación de espacios de trabajo locales.',
      icon: <IconFolder size={18} />,
    },
    {
      id: 'shortcuts',
      label: 'Atajos de Teclado',
      description: 'Comandos rápidos para optimizar la navegación.',
      icon: <IconKeyboard size={18} />,
    },
    {
      id: 'privacy',
      label: 'Privacidad & Datos',
      description: 'Gestión de persistencia local y reportes anónimos.',
      icon: <IconShield size={18} />,
    },
  ];

  const currentCategoryObj = categories.find((c) => c.id === activeCategory) || categories[0];

  return (
    <div
      className="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center z-50 animate-fadeIn p-4 md:p-6"
      onClick={onClose}
    >
      <div
        className="bg-[#0f1416] border border-border-subtle rounded-xl w-full max-w-[1080px] h-[86vh] max-h-[820px] min-h-[560px] flex flex-col shadow-2xl overflow-hidden animate-scaleIn select-none"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        {/* Modal Top Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border-subtle/50 bg-[#0c1012] shrink-0">
          <div className="flex items-center gap-3 text-accent-primary">
            <IconSettings size={18} />
            <h3 className="text-sm font-semibold text-content-headline tracking-tight font-sans">
              Configuración de Merlin Code
            </h3>
          </div>
          <button
            type="button"
            className="p-1.5 text-content-dim hover:text-content-headline hover:bg-[#152227] rounded-md transition-colors cursor-pointer"
            onClick={onClose}
            title="Cerrar configuración"
            aria-label="Cerrar"
          >
            <IconClose size={16} />
          </button>
        </div>

        {/* Modal Main Split: Left Sidebar & Right Content */}
        <div className="flex flex-1 overflow-hidden min-h-0">
          <SettingsNav
            categories={categories}
            activeCategory={activeCategory}
            onSelectCategory={setActiveCategory}
            projectsCount={projects.length}
          />

          {/* Right Content Panel */}
          <main className="flex-1 flex flex-col bg-[#0f1416] overflow-y-auto p-8 md:p-10 min-w-0">
            <div className="mb-8 pb-4 border-b border-border-subtle/40 shrink-0">
              <h2 className="text-xl font-bold text-content-headline mb-1.5 font-sans">
                {currentCategoryObj.label}
              </h2>
              <p className="text-sm text-content-dim leading-relaxed">
                {currentCategoryObj.description}
              </p>
            </div>

            {activeCategory === 'ai' && (
              <AISettingsTab
                settings={settings}
                onUpdate={updateSetting}
                telemetry={telemetry}
              />
            )}

            {activeCategory === 'general' && (
              <GeneralTab settings={settings} onUpdate={updateSetting} />
            )}

            {activeCategory === 'projects' && (
              <ProjectsTab
                projects={projects}
                activeProjectId={activeProjectId}
                onSelectProject={onSelectProject}
                onAddProject={onAddProject}
                onDeleteProject={onDeleteProject}
                onOpenFolder={onOpenFolder}
                onCloseModal={onClose}
              />
            )}

            {activeCategory === 'shortcuts' && <ShortcutsTab />}

            {activeCategory === 'privacy' && (
              <PrivacyTab settings={settings} onUpdate={updateSetting} />
            )}
          </main>
        </div>

        {/* Modal Bottom Footer */}
        <div className="flex items-center justify-between px-8 py-4 border-t border-border-subtle/50 bg-[#0c1012] shrink-0">
          <div className="flex items-center">
            {savedSuccess && (
              <span className="flex items-center gap-2 text-sm text-status-success font-medium animate-fadeIn">
                <IconCheck size={16} />
                <span>Cambios guardados con éxito</span>
              </span>
            )}
          </div>
          <div className="flex items-center gap-3">
            <button
              type="button"
              className="px-4 py-2 rounded-md border border-border-subtle hover:border-border-petrol text-sm text-content-muted hover:text-content-headline transition-colors cursor-pointer"
              onClick={onClose}
            >
              Cancelar
            </button>
            <button
              type="button"
              className="flex items-center gap-2 px-5 py-2 rounded-md bg-accent-primary hover:bg-accent-primary-hover text-[#0b0e10] font-semibold text-sm transition-colors cursor-pointer shadow-sm"
              onClick={handleSave}
            >
              <IconCheck size={16} />
              <span>Guardar Cambios</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
