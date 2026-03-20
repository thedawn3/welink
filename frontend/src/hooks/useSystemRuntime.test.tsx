import { act, renderHook, waitFor } from '@testing-library/react';
import { useSystemRuntime } from './useSystemRuntime';
import { systemApi } from '../services/api';

vi.mock('../services/api', () => ({
  systemApi: {
    getRuntime: vi.fn(),
    getConfigCheck: vi.fn(),
    getTasks: vi.fn(),
    getLogs: vi.fn(),
    getChanges: vi.fn(),
    createEventsSource: vi.fn(),
  },
}));

const mockedSystemApi = vi.mocked(systemApi);

class MockEventSource {
  onopen: ((this: EventSource, ev: Event) => unknown) | null = null;
  onmessage: ((this: EventSource, ev: MessageEvent<string>) => unknown) | null = null;
  onerror: ((this: EventSource, ev: Event) => unknown) | null = null;
  close = vi.fn();
}

describe('useSystemRuntime', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('merges runtime and changes payloads and refreshes on SSE events', async () => {
    const source = new MockEventSource();

    mockedSystemApi.getRuntime.mockResolvedValue({
      engine_type: 'windows',
      decrypt_state: 'ready',
      updated_at: '2026-03-20T04:00:00Z',
    });
    mockedSystemApi.getConfigCheck.mockResolvedValue({
      deployment_target: 'docker',
      mode: 'manual-stage',
      source_dir: { path: '/app/source-data', standard_layout: true },
    });
    mockedSystemApi.getTasks.mockResolvedValue({
      items: [{ id: 'task-1', type: 'decrypt', status: 'running', message: 'decrypting' }],
    });
    mockedSystemApi.getLogs.mockResolvedValue({
      items: [{ time: '2026-03-20T04:00:00Z', level: 'info', source: 'sync', message: 'detected change revision rev-1' }],
    });
    mockedSystemApi.getChanges.mockResolvedValue({
      data_revision: 3,
      pending_changes: 1,
      last_change_reason: 'message_0.db',
      items: [],
      sync: { running: true, watch_wal: true, last_revision_seq: 7 },
    });
    mockedSystemApi.createEventsSource.mockReturnValue(source as unknown as EventSource);

    const { result } = renderHook(() => useSystemRuntime(true, 60_000));

    await waitFor(() => {
      expect(result.current.runtime?.data_revision).toBe(3);
    });

    expect(result.current.runtime?.engine_type).toBe('windows');
    expect(result.current.configCheck?.mode).toBe('manual-stage');
    expect(result.current.runtime?.last_change_reason).toBe('message_0.db');
    expect(result.current.runtime?.updated_at).toBe('2026-03-20T04:00:00Z');
    expect(result.current.changes?.sync?.last_revision_seq).toBe(7);
    expect(result.current.tasks).toHaveLength(1);
    expect(result.current.logs).toHaveLength(1);
    expect(result.current.meta.lastRefreshAt).toBeTruthy();
    expect(result.current.meta.lastRefreshReason).toBe('bootstrap');

    act(() => {
      source.onopen?.call(source as unknown as EventSource, new Event('open'));
    });
    expect(result.current.eventsConnected).toBe(true);

    act(() => {
      source.onmessage?.call(source as unknown as EventSource, {
        data: JSON.stringify({ type: 'runtime.revision.detected', revision: 'rev-2', message: 'detected' }),
      } as MessageEvent<string>);
    });

    await waitFor(() => {
      expect(result.current.latestEvent?.type).toBe('runtime.revision.detected');
    });
    await waitFor(() => {
      expect(mockedSystemApi.getRuntime).toHaveBeenCalledTimes(2);
    });
    expect(result.current.meta.lastEventAt).toBeTruthy();
    expect(result.current.meta.lastRefreshReason).toBe('sse');
  });

  it('falls back to polling when SSE is unavailable', async () => {
    mockedSystemApi.getRuntime.mockResolvedValue({});
    mockedSystemApi.getConfigCheck.mockResolvedValue({});
    mockedSystemApi.getTasks.mockResolvedValue({ items: [] });
    mockedSystemApi.getLogs.mockResolvedValue({ items: [] });
    mockedSystemApi.getChanges.mockResolvedValue({
      data_revision: 0,
      pending_changes: 0,
      items: [],
    });
    mockedSystemApi.createEventsSource.mockReturnValue(null as unknown as EventSource);

    const { result } = renderHook(() => useSystemRuntime(true, 60_000));

    await waitFor(() => {
      expect(result.current.meta.pollingFallback).toBe(true);
    });
  });
});
