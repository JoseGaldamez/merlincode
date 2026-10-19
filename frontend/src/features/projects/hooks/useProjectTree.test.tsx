// @vitest-environment jsdom
import { act, createElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { domain } from '../../../../wailsjs/go/models';
import { useProjectTree } from './useProjectTree';

const bridge = vi.hoisted(() => ({
  SetActiveProject: vi.fn(), GetProjectTree: vi.fn(), listeners: new Set<() => void>(),
}));
vi.mock('../../../../wailsjs/go/app/App', () => bridge);
vi.mock('../../../../wailsjs/runtime/runtime', () => ({
  EventsOn: (name: string, listener: () => void) => {
    expect(name).toBe('workspace:changed');
    bridge.listeners.add(listener);
    return () => bridge.listeners.delete(listener);
  },
}));

const file = (name: string) => domain.FileNode.createFrom({ name, relPath: name, fullPath: name, isDir: false, size: 0 });
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((yes) => { resolve = yes; });
  return { promise, resolve };
}
let current: ReturnType<typeof useProjectTree>;
let root: Root;
let container: HTMLDivElement;
function Harness({ path }: { path?: string }) { current = useProjectTree(path); return null; }
async function render(path?: string) {
  await act(async () => { root.render(createElement(Harness, { path })); });
}
async function changed() {
  await act(async () => { bridge.listeners.forEach((listener) => listener()); });
}
beforeEach(() => {
  vi.resetAllMocks();
  bridge.listeners.clear();
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  bridge.SetActiveProject.mockResolvedValue({});
  bridge.GetProjectTree.mockResolvedValue([]);
  container = document.createElement('div');
  document.body.append(container);
  root = createRoot(container);
});
afterEach(async () => {
  await act(async () => root.unmount());
  expect(bridge.listeners.size).toBe(0);
  container.remove();
  vi.unstubAllGlobals();
});

it('refreshes after a saved file without reactivating the project or losing expanded folders', async () => {
  await render('/alpha');
  await act(async () => { current.toggleDirectory('src'); });
  bridge.GetProjectTree.mockResolvedValue([file('nuevo.py')]);
  await changed();
  expect(current.nodes.map((node) => node.name)).toEqual(['nuevo.py']);
  expect(current.isExpanded('src')).toBe(false);
  expect(bridge.SetActiveProject).toHaveBeenCalledTimes(1);
  await act(async () => { await current.refreshTree(); });
  expect(bridge.SetActiveProject).toHaveBeenCalledTimes(1);
});

it('ignores an outdated response when another write refreshes the tree', async () => {
  await render('/alpha');
  const older = deferred<domain.FileNode[]>();
  bridge.GetProjectTree.mockReturnValueOnce(older.promise);
  await changed();
  bridge.GetProjectTree.mockResolvedValue([file('ultimo.py')]);
  await changed();
  await act(async () => { older.resolve([file('anterior.py')]); });
  expect(current.nodes.map((node) => node.name)).toEqual(['ultimo.py']);
  expect(current.loading).toBe(false);
});

it('ignores an old project response and removes its event subscription when switching projects', async () => {
  const older = deferred<domain.FileNode[]>();
  bridge.GetProjectTree.mockReturnValueOnce(older.promise);
  await render('/alpha');
  bridge.GetProjectTree.mockResolvedValue([file('beta.py')]);
  await render('/beta');
  await act(async () => { older.resolve([file('alpha.py')]); });
  expect(current.nodes.map((node) => node.name)).toEqual(['beta.py']);
  expect(bridge.listeners.size).toBe(1);
  expect(bridge.SetActiveProject.mock.calls).toEqual([['/alpha'], ['/beta']]);
});

it('does not request a tree without a selected project', async () => {
  await render();
  await changed();
  expect(current.nodes).toEqual([]);
  expect(current.loading).toBe(false);
  expect(bridge.GetProjectTree).not.toHaveBeenCalled();
  expect(bridge.SetActiveProject).not.toHaveBeenCalled();
});
