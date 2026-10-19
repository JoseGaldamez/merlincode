// @vitest-environment jsdom
import React, { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ExecutionProvider } from './ExecutionProvider';
import { ExecutionPanel } from './ExecutionPanel';
import { RunProposalCard } from './RunProposalCard';
import { plainLogs } from './ExecutionPanel';
import { useChat } from '../chat/hooks/useChat';
import type { Execution, RunProposal } from './types';

const bridge = vi.hoisted(() => ({
  ListAppRuns: vi.fn(), PrepareAppRun: vi.fn(), StartAppRun: vi.fn(), StopAppRun: vi.fn(), ClearAppRunLogs: vi.fn(), RemoveAppRun: vi.fn(), OpenURLInDefaultBrowser: vi.fn(),
  StartChatStream: vi.fn(), CancelChatStream: vi.fn(), GetSessionMessages: vi.fn(), UpdateMessageFeedback: vi.fn(), RespondToToolApproval: vi.fn(),
  listeners: new Set<(run: Execution) => void>(),
}));
vi.mock('../../../wailsjs/go/app/App', () => bridge);
vi.mock('../../../wailsjs/runtime/runtime', () => ({ EventsOn: (_event: string, listener: (run: Execution) => void) => {
  bridge.listeners.add(listener); return () => { bridge.listeners.delete(listener); };
} }));

const proposal: RunProposal = { id: 'b'.repeat(32), project: '/app', directory: '.', title: 'dev · pnpm', command: 'pnpm run dev', reason: 'package.json: vite', request: { directory: '.' } };
const run: Execution = { proposal, status: 'running', logs: 'ready\n<img src=x onerror=alert(1)>', revision: 1 };
let host: HTMLDivElement;
let root: Root;

beforeEach(() => {
  vi.clearAllMocks();
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  Element.prototype.scrollIntoView = vi.fn();
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  bridge.ListAppRuns.mockResolvedValue([]);
  bridge.PrepareAppRun.mockResolvedValue([proposal]);
  bridge.StartAppRun.mockResolvedValue(run);
  bridge.StopAppRun.mockResolvedValue(undefined);
  bridge.ClearAppRunLogs.mockResolvedValue({ ...run, logs: '', revision: 3 });
  bridge.RemoveAppRun.mockResolvedValue({ ...run, status: 'removed', logs: '', revision: 50 });
  bridge.StartChatStream.mockResolvedValue(undefined);
  bridge.GetSessionMessages.mockResolvedValue([]);
  bridge.CancelChatStream.mockResolvedValue(true);
});
afterEach(async () => { await act(async () => root.unmount()); host.remove(); bridge.listeners.clear(); });

const button = (label: string) => host.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`)!;

describe('application execution controls', () => {
  it('sends an error correction request to the current chat only after clicking, with diagnostic context', async () => {
    const failed: Execution = { ...run, proposal: { ...proposal, command: 'go run server.go', request: { directory: '.', entry: 'server.go' } },
      status: 'error', exitCode: 1, error: 'La aplicación terminó con un error.', logs: '\x1b[31mserver.go:10:14: w.Header().Seft undefined\x1b[0m' };
    bridge.ListAppRuns.mockResolvedValue([failed]);
    function ChatHarness() {
      const chat = useChat({ activeProject: { id: 'app', name: 'App', path: '/app' }, activeSessionId: 'chat-1', selectedProviderId: 'openai' });
      return <ExecutionProvider project="/app" onReveal={vi.fn()} onRequestFix={chat.sendMessage} chatBusy={chat.isLoading}>
        <ExecutionPanel /><div data-chat-messages>{chat.messages.map((message) => <p key={message.id}>{message.content}</p>)}</div>
      </ExecutionProvider>;
    }
    await act(async () => root.render(<ChatHarness />));
    expect(bridge.StartChatStream).not.toHaveBeenCalled();
    await act(async () => button('Pedir solución a Merlin').click());
    expect(bridge.StartChatStream).toHaveBeenCalledOnce();
    const request = bridge.StartChatStream.mock.calls[0][0];
    expect(request.sessionId).toBe('chat-1');
    expect(request.providerId).toBe('openai');
    expect(request.prompt).toContain('Soluciona este error');
    expect(request.prompt).toContain('Comando: go run server.go');
    expect(request.prompt).toContain('Archivo de entrada: server.go');
    expect(request.prompt).toContain('Código de salida: 1');
    expect(request.prompt).toContain('server.go:10:14: w.Header().Seft undefined');
    expect(request.prompt).toContain('tarjeta de aprobación');
    expect(request.prompt).not.toContain('\x1b');
    expect(host.querySelector('[data-chat-messages]')?.textContent).toContain('Soluciona este error');
    expect(button('Pedir solución a Merlin').disabled).toBe(true);
    await act(async () => button('Pedir solución a Merlin').click());
    expect(bridge.StartChatStream).toHaveBeenCalledOnce();
    expect(bridge.StartAppRun).not.toHaveBeenCalled();
  });

  it('does not offer a correction for successful runs or errors in another project', async () => {
    bridge.ListAppRuns.mockResolvedValue([run, { ...run, proposal: { ...proposal, id: 'a'.repeat(32), project: '/other' }, status: 'error' }]);
    const send = vi.fn().mockResolvedValue(undefined);
    await act(async () => root.render(<ExecutionProvider project="/app" onReveal={vi.fn()} onRequestFix={send}><ExecutionPanel /></ExecutionProvider>));
    expect(button('Pedir solución a Merlin')).toBeNull();
    expect(send).not.toHaveBeenCalled();
  });

  it('can request diagnosis without logs and reports failures to send without removing the execution', async () => {
    bridge.ListAppRuns.mockResolvedValue([{ ...run, status: 'error', logs: '', error: 'No se pudo iniciar la aplicación.' }]);
    const send = vi.fn().mockRejectedValue(new Error('No se pudo enviar la consulta'));
    await act(async () => root.render(<ExecutionProvider project="/app" onReveal={vi.fn()} onRequestFix={send}><ExecutionPanel /></ExecutionProvider>));
    await act(async () => button('Pedir solución a Merlin').click());
    expect(send.mock.calls[0][0]).toContain('No hay logs disponibles');
    expect(send.mock.calls[0][0]).toContain('No se pudo iniciar la aplicación.');
    expect(host.querySelector('[role="alert"]')?.textContent).toContain('No se pudo enviar la consulta');
    expect(host.querySelector('[data-run-id]')).not.toBeNull();
    expect(button('Pedir solución a Merlin').disabled).toBe(false);
  });

  it('bounds diagnostic output and keeps log backticks inside their diagnostic block', async () => {
    bridge.ListAppRuns.mockResolvedValue([{ ...run, status: 'error', logs: 'old output\n' + 'x'.repeat(13000) + '\n```\nlatest error' }]);
    const send = vi.fn().mockResolvedValue(undefined);
    await act(async () => root.render(<ExecutionProvider project="/app" onReveal={vi.fn()} onRequestFix={send}><ExecutionPanel /></ExecutionProvider>));
    await act(async () => button('Pedir solución a Merlin').click());
    const prompt = send.mock.calls[0][0];
    expect(prompt).not.toContain('old output');
    expect(prompt).toContain('solo el tramo final');
    expect(prompt).toContain('````text\n');
    expect(prompt).toContain('latest error\n````');
    expect(prompt.length).toBeLessThan(14000);
  });

  it('opens HTTP URLs from console output in the default browser while preserving text', async () => {
    const logs = '\x1b[32mLocal:\x1b[0m http://localhost:5173/\nDocs: (https://example.com/api?q=1&x=2#ready).\nIPv6: http://[::1]:8000/\nWiki: https://example.com/wiki/Test_(example)\njavascript:alert(1) file:///tmp/test <img src=x onerror=alert(1)>';
    bridge.ListAppRuns.mockResolvedValue([{ ...run, logs }]);
    await act(async () => root.render(<ExecutionProvider project="/app" onReveal={vi.fn()}><ExecutionPanel /></ExecutionProvider>));
    const links = Array.from(host.querySelectorAll<HTMLAnchorElement>('pre a'));
    const urls = ['http://localhost:5173/', 'https://example.com/api?q=1&x=2#ready', 'http://[::1]:8000/', 'https://example.com/wiki/Test_(example)'];
    expect(links.map((link) => link.getAttribute('href'))).toEqual(urls);
    expect(host.querySelector('pre')?.textContent).toBe(plainLogs(logs));
    expect(host.querySelector('img')).toBeNull();
    expect(bridge.OpenURLInDefaultBrowser).not.toHaveBeenCalled();
    for (const [index, link] of links.entries()) {
      const event = new MouseEvent('click', { bubbles: true, cancelable: true });
      await act(async () => { link.dispatchEvent(event); });
      expect(event.defaultPrevented).toBe(true);
      expect(bridge.OpenURLInDefaultBrowser).toHaveBeenNthCalledWith(index + 1, urls[index]);
    }
  });

  it('detects links as logs arrive and removes them when logs are cleared', async () => {
    bridge.ListAppRuns.mockResolvedValue([{ ...run, logs: 'Listening at http://' }]);
    await act(async () => root.render(<ExecutionProvider project="/app" onReveal={vi.fn()}><ExecutionPanel /></ExecutionProvider>));
    expect(host.querySelector('pre a')).toBeNull();
    await act(async () => { bridge.listeners.forEach((listener) => listener({ ...run, revision: 2, logs: 'Listening at http://localhost:8000' })); });
    expect(host.querySelector('pre a')?.textContent).toBe('http://localhost:8000');
    await act(async () => button('Limpiar logs').click());
    expect(host.querySelector('pre a')).toBeNull();
  });

  it('starts only on Play, reveals the panel, streams text, clears and stops', async () => {
    const reveal = vi.fn();
    await act(async () => root.render(<ExecutionProvider project="/app" onReveal={reveal}><RunProposalCard proposal={proposal} /><ExecutionPanel /></ExecutionProvider>));
    expect(bridge.StartAppRun).not.toHaveBeenCalled();
    await act(async () => button('Play: pnpm run dev').click());
    expect(reveal).toHaveBeenCalledOnce();
    expect(bridge.StartAppRun).toHaveBeenCalledWith(proposal.id);
    expect(host.querySelector('pre')?.textContent).toContain('<img src=x');
    expect(host.querySelector('img')).toBeNull();
    await act(async () => button('Limpiar logs').click());
    await act(async () => { bridge.listeners.forEach((listener) => listener({ ...run, revision: 2, logs: 'delayed old logs' })); });
    expect(host.querySelector('pre')?.textContent).not.toContain('delayed old logs');
    await act(async () => button('Detener aplicación').click());
    expect(bridge.StopAppRun).toHaveBeenCalledWith(proposal.id);
    await act(async () => { bridge.listeners.forEach((listener) => listener({ ...run, revision: 4, status: 'stopped', exitCode: 1, logs: '' })); });
    expect(button('Volver a ejecutar')).toBeTruthy();
    await act(async () => button('Volver a ejecutar').click());
    expect(bridge.StartAppRun).toHaveBeenCalledTimes(2);
  });

  it('refuses a historical card when the project command has changed', async () => {
    bridge.PrepareAppRun.mockResolvedValue([{ ...proposal, id: 'c'.repeat(32), command: 'yarn run dev' }]);
    await act(async () => root.render(<ExecutionProvider project="/app" onReveal={vi.fn()}><RunProposalCard proposal={proposal} /></ExecutionProvider>));
    await act(async () => button('Play: pnpm run dev').click());
    expect(bridge.StartAppRun).not.toHaveBeenCalled();
    expect(host.textContent).toContain('Cambió el proyecto');
  });

  it('reruns edited source with fresh evidence and replaces the old entry', async () => {
    const current = { ...proposal, id: 'c'.repeat(32) };
    bridge.ListAppRuns.mockResolvedValue([{ ...run, status: 'exited' }]);
    bridge.PrepareAppRun.mockResolvedValue([current]);
    bridge.StartAppRun.mockResolvedValue({ ...run, proposal: current, logs: 'updated file', revision: 51 });
    await act(async () => root.render(<ExecutionProvider project="/app" onReveal={vi.fn()}><ExecutionPanel /></ExecutionProvider>));
    await act(async () => button('Volver a ejecutar').click());
    expect(bridge.RemoveAppRun).toHaveBeenCalledWith(proposal.id);
    expect(bridge.StartAppRun).toHaveBeenCalledWith(current.id);
    expect(bridge.RemoveAppRun.mock.invocationCallOrder[0]).toBeLessThan(bridge.StartAppRun.mock.invocationCallOrder[0]);
    await act(async () => { bridge.listeners.forEach((listener) => listener({ ...run, revision: 49 })); });
    expect(host.querySelector(`[data-run-id="${proposal.id}"]`)).toBeNull();
    expect(host.querySelectorAll('[data-run-id]')).toHaveLength(1);
    expect(host.querySelector('pre')?.textContent).toBe('updated file');
  });

  it('keeps clear logs separate from removing a running execution', async () => {
    bridge.ListAppRuns.mockResolvedValue([run]);
    await act(async () => root.render(<ExecutionProvider project="/app" onReveal={vi.fn()}><ExecutionPanel /></ExecutionProvider>));
    await act(async () => button('Limpiar logs').click());
    expect(host.querySelectorAll('[data-run-id]')).toHaveLength(1);
    expect(bridge.RemoveAppRun).not.toHaveBeenCalled();
    let removed!: (value: Execution) => void;
    bridge.RemoveAppRun.mockImplementation(() => new Promise<Execution>((resolve) => { removed = resolve; }));
    await act(async () => button('Eliminar ejecución').click());
    expect(host.querySelectorAll('[data-run-id]')).toHaveLength(1);
    await act(async () => removed({ ...run, status: 'removed', logs: '', revision: 50 }));
    expect(host.querySelectorAll('[data-run-id]')).toHaveLength(0);
    await act(async () => { bridge.listeners.forEach((listener) => listener({ ...run, revision: 49 })); });
    expect(host.querySelectorAll('[data-run-id]')).toHaveLength(0);
  });

  it('retains the entry if the backend cannot stop and remove it', async () => {
    bridge.ListAppRuns.mockResolvedValue([run]);
    bridge.RemoveAppRun.mockRejectedValue(new Error('No se pudo detener'));
    await act(async () => root.render(<ExecutionProvider project="/app" onReveal={vi.fn()}><ExecutionPanel /></ExecutionProvider>));
    await act(async () => button('Eliminar ejecución').click());
    expect(host.querySelectorAll('[data-run-id]')).toHaveLength(1);
    expect(host.querySelector('[role="alert"]')?.textContent).toContain('No se pudo detener');
  });

  it('strips terminal control sequences without interpreting HTML', () => {
    expect(plainLogs('\x1b[31merror\x1b[0m\x1b]0;title\x07')).toBe('error');
  });
});
