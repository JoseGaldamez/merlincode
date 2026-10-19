// @vitest-environment jsdom
import React, { act, useState } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { RightSidebar } from '../artifacts/components/RightSidebar';
import { useProjects } from '../projects/hooks/useProjects';
import { ExecutionProvider } from './ExecutionProvider';
import { RunningProjectsIndicator } from './RunningProjectsIndicator';
import type { Execution, ExecutionFocus, RunProposal } from './types';

const bridge = vi.hoisted(() => ({
  ListAppRuns: vi.fn(), SetActiveProject: vi.fn(), SelectProjectFolder: vi.fn(),
  PrepareAppRun: vi.fn(), StartAppRun: vi.fn(), StopAppRun: vi.fn(),
  ClearAppRunLogs: vi.fn(), RemoveAppRun: vi.fn(), OpenURLInDefaultBrowser: vi.fn(),
  listeners: new Set<(run: Execution) => void>(),
}));
vi.mock('../../../wailsjs/go/app/App', () => bridge);
vi.mock('../../../wailsjs/runtime/runtime', () => ({ EventsOn: (_event: string, listener: (run: Execution) => void) => {
  bridge.listeners.add(listener); return () => { bridge.listeners.delete(listener); };
} }));

const alpha = { id: 'a', name: 'Alpha', path: '/alpha' };
const beta = { id: 'b', name: 'Beta', path: '/beta' };
const proposal: RunProposal = { id: 'a'.repeat(32), project: alpha.path, directory: '.', title: 'Alpha server', command: 'pnpm run dev', reason: 'package.json', request: { directory: '.' } };
const alphaRun: Execution = { proposal, status: 'running', logs: 'Alpha output', revision: 1 };
const betaRun: Execution = { ...alphaRun, proposal: { ...proposal, id: 'b'.repeat(32), project: beta.path, title: 'Beta server' }, logs: 'Beta output' };
const betaRun2: Execution = { ...betaRun, proposal: { ...betaRun.proposal, id: 'c'.repeat(32), title: 'Beta worker' } };
let host: HTMLDivElement;
let root: Root;
function Harness() {
  const { projects, activeProject, selectRunningProject } = useProjects();
  const [open, setOpen] = useState(false);
  const [focus, setFocus] = useState<ExecutionFocus>();
  const reveal = (next: ExecutionFocus) => { setFocus(next); setOpen(true); };
  return <ExecutionProvider project={activeProject?.path || ''} onReveal={reveal}>
    <span data-active-project>{activeProject?.path}</span>
    <RunningProjectsIndicator projects={projects} onSelect={async (path, next) => {
      await selectRunningProject(path); reveal(next);
    }} />
    <RightSidebar isOpen={open} onToggle={() => setOpen(!open)} width={300} onWidthChange={vi.fn()} artifacts={[]} executionFocus={focus} />
  </ExecutionProvider>;
}
const button = (label: string) => host.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`)!;
const render = async () => { await act(async () => root.render(<Harness />)); };
beforeEach(() => {
  vi.resetAllMocks();
  bridge.listeners.clear();
  localStorage.clear();
  localStorage.setItem('merlin_projects_list', JSON.stringify([alpha, beta]));
  localStorage.setItem('merlin_active_project', alpha.id);
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  Element.prototype.scrollIntoView = vi.fn();
  bridge.ListAppRuns.mockResolvedValue([alphaRun, betaRun, betaRun2]);
  bridge.SetActiveProject.mockImplementation(async (path: string) => path === alpha.path ? alpha : beta);
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove(); localStorage.clear(); vi.unstubAllGlobals();
});

it('groups counts across projects, activates the selected project and opens its executions', async () => {
  await render();
  await act(async () => button('Ejecuciones activas: 3').click());
  expect(button('Alpha: 1 en curso').getAttribute('aria-current')).toBe('true');
  expect(button('Beta: 2 en curso')).toBeTruthy();
  await act(async () => button('Beta: 2 en curso').click());
  expect(bridge.SetActiveProject).toHaveBeenLastCalledWith(beta.path);
  expect(host.querySelector('[data-active-project]')?.textContent).toBe(beta.path);
  expect(host.querySelectorAll('[data-run-id]')).toHaveLength(2);
  expect(host.querySelector('pre')?.textContent).toBe('Beta output');
  expect(host.querySelector('section[aria-label="Ejecuciones"]')).not.toBeNull();
  expect(button('Colapsar Ejecuciones').getAttribute('aria-expanded')).toBe('true');
  expect(bridge.StopAppRun).not.toHaveBeenCalled();
  expect(bridge.StartAppRun).not.toHaveBeenCalled();
  expect(button('Ejecuciones activas: 3').getAttribute('aria-expanded')).toBe('false');

  await act(async () => { bridge.listeners.forEach((listener) => listener({ ...betaRun2, status: 'exited', revision: 2 })); });
  await act(async () => button('Ejecuciones activas: 2').click());
  expect(button('Beta: 1 en curso')).toBeTruthy();
  await act(async () => button('Alpha: 1 en curso').click());
  expect(host.querySelector('pre')?.textContent).toBe('Alpha output');
  expect(host.querySelectorAll('[data-run-id]')).toHaveLength(1);
});

it('keeps a project with active servers reachable after it was removed from the saved list', async () => {
  localStorage.setItem('merlin_projects_list', JSON.stringify([alpha]));
  await render();
  await act(async () => button('Ejecuciones activas: 3').click());
  await act(async () => button('beta: 2 en curso').click());
  expect(host.querySelector('[data-active-project]')?.textContent).toBe(beta.path);
  expect(JSON.parse(localStorage.getItem('merlin_projects_list')!)).toContainEqual(beta);
});

it('reports an unavailable project without changing the selection or opening the wrong console', async () => {
  await render();
  bridge.SetActiveProject.mockRejectedValue(new Error('unavailable'));
  await act(async () => button('Ejecuciones activas: 3').click());
  await act(async () => button('Beta: 2 en curso').click());
  expect(host.querySelector('[role="alert"]')?.textContent).toContain('No se pudo abrir el proyecto');
  expect(host.querySelector('[data-active-project]')?.textContent).toBe(alpha.path);
  expect(host.querySelector('[data-run-id]')).toBeNull();
});

it('hides when no executions are active and reappears with its disclosure closed on the next launch', async () => {
  bridge.ListAppRuns.mockResolvedValue([{ ...alphaRun, status: 'exited' }]);
  await render();
  expect(host.querySelector('.running-projects')).toBeNull();
  await act(async () => { bridge.listeners.forEach((listener) => listener({ ...alphaRun, revision: 2 })); });
  await act(async () => button('Ejecuciones activas: 1').click());
  expect(host.querySelector('[aria-label="Ejecuciones por proyecto"]')).not.toBeNull();
  await act(async () => { bridge.listeners.forEach((listener) => listener({ ...alphaRun, status: 'stopped', revision: 3 })); });
  expect(host.querySelector('.running-projects')).toBeNull();
  expect(host.querySelector('[aria-label="Ejecuciones por proyecto"]')).toBeNull();
  await act(async () => { bridge.listeners.forEach((listener) => listener({ ...alphaRun, revision: 4 })); });
  expect(button('Ejecuciones activas: 1').getAttribute('aria-expanded')).toBe('false');
});

it('closes the disclosure with Escape', async () => {
  bridge.ListAppRuns.mockResolvedValue([alphaRun]);
  await render();
  const trigger = button('Ejecuciones activas: 1');
  await act(async () => trigger.click());
  await act(async () => trigger.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true })));
  expect(trigger.getAttribute('aria-expanded')).toBe('false');
  expect(document.activeElement).toBe(trigger);
});
