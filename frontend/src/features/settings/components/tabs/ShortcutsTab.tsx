import React from 'react';

export const ShortcutsTab: React.FC = () => {
  const shortcuts = [
    { label: 'Enviar mensaje o instrucción', key: 'Enter' },
    { label: 'Salto de línea en el campo de texto', key: 'Shift + Enter' },
    { label: 'Buscar en el historial de sesiones', key: 'Ctrl + F' },
    { label: 'Abrir Configuración', key: 'Ctrl + ,' },
    { label: 'Cerrar diálogo o modal', key: 'Esc' },
  ];

  return (
    <div className="flex flex-col gap-9 max-w-3xl">
      <div>
        <div className="mb-4">
          <h3 className="text-xs font-mono font-semibold tracking-wider text-accent-primary uppercase">
            Comandos de Teclado
          </h3>
          <p className="text-xs text-content-dim mt-1 leading-relaxed">
            Combinaciones rápidas para agilizar tu flujo de trabajo en Merlin Code.
          </p>
        </div>

        <div className="divide-y divide-border-subtle/40 border-t border-b border-border-subtle/40">
          {shortcuts.map((sc) => (
            <div
              key={sc.label}
              className="py-3.5 flex items-center justify-between gap-4"
            >
              <span className="text-sm font-medium text-content-body">{sc.label}</span>
              <kbd className="px-3 py-1.5 rounded-md bg-[#141d22] border border-[#21353f] text-accent-primary text-xs font-mono font-semibold shadow-sm">
                {sc.key}
              </kbd>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
