// @vitest-environment jsdom
import React, { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { RightSidebar, type RightSidebarProps } from './RightSidebar';
import { LeftSidebar } from '../../sessions/components/LeftSidebar';

vi.mock('../../executions/ExecutionPanel', () => ({ ExecutionPanel: () => <div>Salida de ejecución</div> }));
let host: HTMLDivElement;
let root: Root;
let sidebarHeight: number;
const props: RightSidebarProps = { isOpen: true, onToggle: vi.fn(), width: 300, onWidthChange: vi.fn(), artifacts: [] };
const render = async (overrides: Partial<RightSidebarProps> = {}) => {
  await act(async () => root.render(<RightSidebar {...props} {...overrides} />));
};
const button = (label: string) => host.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`)!;
const separator = () => host.querySelector<HTMLElement>('[role="separator"]')!;
const executions = () => host.querySelector<HTMLElement>('section[aria-label="Ejecuciones"]')!;
beforeEach(() => {
  localStorage.clear();
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  sidebarHeight = 600;
  vi.spyOn(HTMLElement.prototype, 'clientHeight', 'get').mockImplementation(() => sidebarHeight);
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove(); localStorage.clear(); vi.restoreAllMocks(); vi.unstubAllGlobals();
});

it('shows artifacts and executions simultaneously with a dedicated execution destination', async () => {
  await render();
  expect(host.querySelector('section[aria-label="Artefactos"]')?.textContent).toContain('Sin artefactos aún');
  expect(executions().textContent).toContain('Salida de ejecución');
  expect(executions().hasAttribute('data-execution-destination')).toBe(true);
  expect(separator().getAttribute('aria-orientation')).toBe('horizontal');
  expect(host.querySelector('.execution-tabs')).toBeNull();
});

it('resizes vertically by dragging and restores the saved height after remounting', async () => {
  await render();
  await act(async () => separator().dispatchEvent(new MouseEvent('mousedown', { clientY: 300, button: 0, bubbles: true })));
  expect(document.body.style.cursor).toBe('row-resize');
  await act(async () => window.dispatchEvent(new MouseEvent('mousemove', { clientY: 200 })));
  expect(executions().style.height).toBe('460px');
  await act(async () => window.dispatchEvent(new MouseEvent('mouseup')));
  expect(document.body.style.cursor).toBe('');
  expect(localStorage.getItem('merlin_executions_panel_height')).toBe('460');
  await act(async () => root.unmount());
  root = createRoot(host);
  await render();
  expect(executions().style.height).toBe('460px');
});

it('supports keyboard resizing and leaves room for both sections when the window shrinks', async () => {
  await render();
  const key = async (value: string) => {
    await act(async () => separator().dispatchEvent(new KeyboardEvent('keydown', { key: value, bubbles: true, cancelable: true })));
  };
  await key('ArrowUp');
  expect(executions().style.height).toBe('384px');
  await key('Home');
  expect(executions().style.height).toBe('120px');
  await key('End');
  expect(executions().style.height).toBe('480px');
  sidebarHeight = 400;
  await act(async () => window.dispatchEvent(new Event('resize')));
  expect(executions().style.height).toBe('280px');
});

it('persists folding independently and expands executions when a launch is revealed', async () => {
  await render();
  await act(async () => button('Colapsar Ejecuciones').click());
  expect(localStorage.getItem('merlin_executions_panel_collapsed')).toBe('true');
  expect(host.querySelector('[role="separator"]')).toBeNull();
  const content = document.getElementById(button('Desplegar Ejecuciones').getAttribute('aria-controls')!)!;
  expect(content.hidden).toBe(true);
  expect(host.querySelector('section[aria-label="Artefactos"]')).not.toBeNull();
  await act(async () => root.unmount());
  root = createRoot(host);
  await render();
  expect(button('Desplegar Ejecuciones')).not.toBeNull();
  await render({ executionFocus: { id: 'a'.repeat(32), command: 'pnpm run dev', nonce: 1 } });
  expect(button('Colapsar Ejecuciones').getAttribute('aria-expanded')).toBe('true');
  expect(localStorage.getItem('merlin_executions_panel_collapsed')).toBe('false');
});

it('cleans up a drag if the sidebar closes before mouseup', async () => {
  await render();
  await act(async () => separator().dispatchEvent(new MouseEvent('mousedown', { clientY: 300, button: 0, bubbles: true })));
  await render({ isOpen: false });
  expect(document.body.style.cursor).toBe('');
  expect(document.body.style.userSelect).toBe('');
  await act(async () => window.dispatchEvent(new MouseEvent('mousemove', { clientY: 100 })));
  await render();
  expect(executions().style.height).toBe('360px');
});

it('names both inner panels when the right and left sidebars are collapsed', async () => {
  await act(async () => root.render(<>
    <RightSidebar {...props} isOpen={false} />
    <LeftSidebar isOpen={false} onToggle={vi.fn()} width={260} onWidthChange={vi.fn()} sessions={[]} activeSessionId=""
      onSelectSession={vi.fn()} onNewSession={vi.fn()} onDeleteSession={vi.fn()} />
  </>));
  expect(button('Expandir panel derecho').textContent).toContain('Artefactos | Ejecuciones');
  expect(button('Expandir panel de sesiones').textContent).toContain('Sesiones | Directorio');
});
