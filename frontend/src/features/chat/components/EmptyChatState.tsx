import React from 'react';
import logoImg from '../../../assets/images/logo.png';
import { IconFolder } from '../../../components/Icons';

interface EmptyChatStateProps {
  activeProjectName?: string;
  onOpenFolder?: () => void;
}

export const EmptyChatState: React.FC<EmptyChatStateProps> = ({
  activeProjectName,
  onOpenFolder,
}) => {
  return (
    <div className="flex-1 flex items-center justify-center my-auto">
      <div className="flex flex-col items-center justify-center p-8 text-center max-w-sm">
        <div className="w-16 h-16 rounded-full bg-surface-card border border-border-subtle p-3 flex items-center justify-center mb-4 shadow-sm">
          <img src={logoImg} alt="Merlin Code Logo" className="w-full h-full object-contain" />
        </div>
        <h2 className="text-base font-semibold text-content-headline mb-1.5 font-sans">
          Merlin Code
        </h2>
        {activeProjectName ? (
          <p className="text-xs text-content-dim leading-relaxed">
            Proyecto activo:{' '}
            <span className="text-accent-primary font-mono font-semibold">
              {activeProjectName}
            </span>
            . Escribe un mensaje o instrucción en el panel inferior para comenzar.
          </p>
        ) : (
          <>
            <p className="text-xs text-content-dim leading-relaxed mb-4">
              No hay ningún proyecto seleccionado. Abre una carpeta para que Merlin pueda
              trabajar y editar tus archivos con seguridad.
            </p>
            {onOpenFolder && (
              <button
                type="button"
                onClick={onOpenFolder}
                className="inline-flex items-center gap-2 px-3.5 py-2 bg-accent-primary hover:bg-accent-primary-hover text-[#0b0e10] font-semibold text-xs rounded-lg transition-colors cursor-pointer shadow-sm"
              >
                <IconFolder size={15} />
                <span>Seleccionar carpeta de proyecto</span>
              </button>
            )}
          </>
        )}
      </div>
    </div>
  );
};
