import React from 'react';
import { IconMinus, IconSquare, IconRestore, IconClose } from '../../../components/Icons';
import { useWindowControls } from '../hooks/useWindowControls';

export const WindowControls: React.FC = () => {
  const { isMaximized, minimize, toggleMaximize, close } = useWindowControls();

  return (
    <div className="flex items-center h-full">
      <button
        type="button"
        className="h-full px-3.5 flex items-center justify-center text-content-muted hover:text-content-headline hover:bg-[#1a2429] transition-colors cursor-pointer"
        onClick={minimize}
        title="Minimizar"
        aria-label="Minimizar ventana"
      >
        <IconMinus size={11} />
      </button>

      <button
        type="button"
        className="h-full px-3.5 flex items-center justify-center text-content-muted hover:text-content-headline hover:bg-[#1a2429] transition-colors cursor-pointer"
        onClick={toggleMaximize}
        title={isMaximized ? 'Restaurar' : 'Maximizar'}
        aria-label={isMaximized ? 'Restaurar ventana' : 'Maximizar ventana'}
      >
        {isMaximized ? <IconRestore size={11} /> : <IconSquare size={11} />}
      </button>

      <button
        type="button"
        className="h-full px-3.5 flex items-center justify-center text-content-muted hover:text-white hover:bg-[#c42b1c] transition-colors cursor-pointer"
        onClick={close}
        title="Cerrar"
        aria-label="Cerrar ventana"
      >
        <IconClose size={13} />
      </button>
    </div>
  );
};
