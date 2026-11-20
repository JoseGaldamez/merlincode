// @vitest-environment jsdom
import React, { act } from 'react';
import { createRoot } from 'react-dom/client';
import { expect, it, vi } from 'vitest';
import { ToolActivityPanel } from '../components/ToolActivityPanel';
import { useChat } from './useChat';

const getSessionMessages = vi.hoisted(() => vi.fn());
vi.mock('../../../../wailsjs/go/app/App', () => ({ GetSessionMessages: getSessionMessages }));
vi.mock('../../../../wailsjs/runtime/runtime', () => ({ EventsOn: () => () => {} }));

function ActivityForSession({ id }: { id: string }) {
  const { messages } = useChat({ activeSessionId: id });
  return messages.map((message) => message.toolActivity?.length
    ? <ToolActivityPanel key={message.id} activity={message.toolActivity} />
    : null);
}

it('restaura el desplegable y su actividad al volver a una sesión guardada', async () => {
  getSessionMessages.mockImplementation(async (id: string) => id === 'with-activity' ? [{
    id: 'assistant-1',
    role: 'assistant',
    toolTrace: [
      { toolName: 'read_file', arguments: '{"path":"package.json"}', result: 'Contenido del archivo' },
      { toolName: 'read_file', arguments: 'incompleto', result: 'Archivo no encontrado', isError: true },
      { toolName: 'edit_file', arguments: '{"path":"src/index.ts","new_string":"contenido de la edición"}', isError: true, approved: false },
    ],
  }] : [{ id: 'assistant-2', role: 'assistant' }]);

  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  const host = document.createElement('div');
  document.body.append(host);
  const root = createRoot(host);
  const openSession = async (id: string) => act(async () => root.render(<ActivityForSession id={id} />));

  try {
    await openSession('with-activity');
    await openSession('empty');
    expect(host.querySelector('button')).toBeNull();

    await openSession('with-activity');
    const toggle = host.querySelector('button');
    expect(toggle?.getAttribute('aria-expanded')).toBe('false');
    expect(toggle?.textContent).toContain('(3)');
    expect(host.textContent).not.toContain('package.json');

    await act(async () => toggle?.click());
    expect(host.textContent).toContain('package.json');
    expect(host.textContent).toContain('Archivo no encontrado');
    expect(host.textContent).toContain('Rechazado');
    expect(host.textContent).not.toContain('contenido de la edición');
  } finally {
    await act(async () => root.unmount());
    host.remove();
    vi.unstubAllGlobals();
  }
});
