// @vitest-environment jsdom
import { act, createElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { ChatStreamEvent, Project } from '../../../types';
import { useSessions } from './useSessions';
import { useChat } from '../../chat/hooks/useChat';

const bridge = vi.hoisted(() => ({
  ListSessions: vi.fn(), CreateSession: vi.fn(), DeleteSession: vi.fn(), UpdateSessionTitle: vi.fn(),
  GetSessionMessages: vi.fn(), StartChatStream: vi.fn(), CancelChatStream: vi.fn(),
  UpdateMessageFeedback: vi.fn(), RespondToToolApproval: vi.fn(),
  listeners: new Set<(event: ChatStreamEvent) => void>(),
}));
vi.mock('../../../../wailsjs/go/app/App', () => bridge);
vi.mock('../../../../wailsjs/runtime/runtime', () => ({
  EventsOn: (_name: string, listener: (event: ChatStreamEvent) => void) => {
    bridge.listeners.add(listener);
    return () => bridge.listeners.delete(listener);
  },
}));
const projectA: Project = { id: 'a', name: 'Alpha', path: '/projects/alpha' };
const projectB: Project = { id: 'b', name: 'Beta', path: '/projects/beta' };
const savedSessions = [
  { id: 'a-chat', title: 'Alpha chat', projectId: 'a', projectPath: projectA.path, messagesCount: 1 },
  { id: 'b-chat', title: 'Beta chat', projectId: 'b', projectPath: projectB.path, messagesCount: 1 },
  { id: 'legacy-a', title: 'Legacy Alpha', projectId: '', projectPath: projectA.path, messagesCount: 1 },
  { id: 'unassigned', title: 'Unassigned', projectId: '', projectPath: '', messagesCount: 1 },
  { id: 'conflicting', title: 'Conflicting owner', projectId: 'b', projectPath: projectA.path, messagesCount: 1 },
];
function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: Error) => void;
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
function useConversation(project: Project | null) {
  const sessions = useSessions({ activeProject: project });
  const chat = useChat({ activeProject: project, activeSessionId: sessions.activeSessionId,
    onEnsureSession: sessions.ensureSession, onRefreshSessions: sessions.refreshSessions });
  return { ...sessions, chat };
}
let current: ReturnType<typeof useConversation>;
let root: Root;
let container: HTMLDivElement;
function Harness({ project }: { project: Project | null }) {
  current = useConversation(project);
  return null;
}
async function render(project: Project | null) {
  await act(async () => { root.render(createElement(Harness, { project })); });
}
function emit(event: ChatStreamEvent) { bridge.listeners.forEach((listener) => listener(event)); }

beforeEach(() => {
  vi.resetAllMocks();
  bridge.listeners.clear();
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  bridge.ListSessions.mockResolvedValue(savedSessions);
  bridge.CreateSession.mockResolvedValue({});
  bridge.UpdateSessionTitle.mockResolvedValue(undefined);
  bridge.CancelChatStream.mockResolvedValue(true);
  bridge.StartChatStream.mockResolvedValue(undefined);
  bridge.GetSessionMessages.mockImplementation(async (id: string) => [{ id: id + '-saved', role: 'user', content: 'history-' + id, status: 'done' }]);
  let clock = 1000;
  vi.spyOn(Date, 'now').mockImplementation(() => ++clock);
  container = document.createElement('div');
  document.body.append(container);
  root = createRoot(container);
});
afterEach(async () => {
  await act(async () => root.unmount());
  container.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe('conversations belong to the selected project', () => {
  it('changes the list, active conversation, messages and search when switching projects', async () => {
    await render(projectA);
    expect(current.sessions.map((session) => session.id)).toEqual(['a-chat', 'legacy-a']);
    expect(current.activeSessionId).toBe('a-chat');
    expect(current.chat.messages[0].content).toBe('history-a-chat');
    await act(async () => current.setSearchQuery('Alpha'));
    await render(projectB);
    expect(current.sessions.map((session) => session.id)).toEqual(['b-chat']);
    expect(current.activeSessionId).toBe('b-chat');
    expect(current.chat.messages[0].content).toBe('history-b-chat');
    expect(current.searchQuery).toBe('');
    await act(async () => current.selectSession('a-chat'));
    expect(current.activeSessionId).toBe('b-chat');
    await render(projectA);
    expect(current.activeSessionId).toBe('a-chat');
    await render({ id: 'empty', name: 'Empty', path: '/projects/empty' });
    expect(current.sessions).toEqual([]);
    expect(current.activeSessionId).toBe('');
    expect(current.chat.messages).toEqual([]);
    await render(null);
    expect(current.sessions).toEqual([]);
    expect(current.ensureSession('no project')).toBe('');
  });

  it('ignores a late session-list response from the previous project', async () => {
    const oldList = deferred<typeof savedSessions>();
    bridge.ListSessions.mockImplementationOnce(() => oldList.promise);
    await render(projectA);
    await render(projectB);
    await act(async () => oldList.resolve(savedSessions));
    expect(current.sessions.map((session) => session.id)).toEqual(['b-chat']);
    expect(current.activeSessionId).toBe('b-chat');
  });

  it('does not select a session whose creation completes after leaving its project', async () => {
    const oldCreation = deferred<object>();
    bridge.CreateSession.mockImplementationOnce(() => oldCreation.promise);
    await render(projectA);
    let creation!: Promise<string>;
    await act(async () => { creation = current.createSession('Alpha draft'); });
    expect(bridge.CreateSession).toHaveBeenLastCalledWith(expect.any(String), 'Alpha draft', 'a', projectA.path);
    await render(projectB);
    await act(async () => { oldCreation.resolve({}); await creation; });
    expect(current.sessions.map((session) => session.id)).toEqual(['b-chat']);
    expect(current.activeSessionId).toBe('b-chat');
    await act(async () => { await current.createSession('Beta draft'); });
    expect(bridge.CreateSession).toHaveBeenLastCalledWith(expect.any(String), 'Beta draft', 'b', projectB.path);
    expect(current.sessions[0].projectId).toBe('b');
  });

  it('ignores a previous project message load that finishes after switching', async () => {
    const oldMessages = deferred<object[]>();
    bridge.GetSessionMessages.mockImplementationOnce(() => oldMessages.promise);
    await render(projectA);
    await render(projectB);
    await act(async () => oldMessages.resolve([{ id: 'late', role: 'user', content: 'Alpha must stay hidden' }]));
    expect(current.chat.messages.map((message) => message.content)).toEqual(['history-b-chat']);
  });

  it('cancels the old stream and ignores its late events and errors while the new project streams', async () => {
    const oldStart = deferred<void>();
    bridge.StartChatStream.mockImplementationOnce(() => oldStart.promise);
    await render(projectA);
    await act(async () => { await current.chat.sendMessage('Alpha request'); });
    const alphaRequest = bridge.StartChatStream.mock.calls[0][0];
    expect(alphaRequest.sessionId).toBe('a-chat');
    expect(current.chat.isLoading).toBe(true);
    await render(projectB);
    expect(bridge.CancelChatStream).toHaveBeenCalledWith(alphaRequest.messageId);
    expect(current.chat.isLoading).toBe(false);
    expect(current.chat.telemetry.tokensPrompt).toBe(0);
    await act(async () => { await current.chat.sendMessage('Beta request'); });
    const betaRequest = bridge.StartChatStream.mock.calls[1][0];
    expect(betaRequest.sessionId).toBe('b-chat');
    expect(betaRequest.history).toEqual([{ role: 'user', content: 'history-b-chat' }]);
    await act(async () => {
      emit({ messageId: alphaRequest.messageId, type: 'status', statusText: 'Old status' });
      emit({ messageId: alphaRequest.messageId, type: 'done', tokensPrompt: 999 });
      oldStart.reject(new Error('Old request failed'));
    });
    expect(current.chat.isLoading).toBe(true);
    expect(current.chat.telemetry.tokensPrompt).not.toBe(999);
    expect(current.chat.streamingStatusText).not.toBe('Old status');
    await act(async () => emit({ messageId: betaRequest.messageId, type: 'content', content: 'Beta response' }));
    expect(current.chat.messages.at(-1)?.content).toBe('Beta response');
    expect(current.chat.messages.some((message) => message.content.includes('Alpha'))).toBe(false);
  });
});
