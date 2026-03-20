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
  },
}));

const mockedSystemApi = vi.mocked(systemApi);

describe('useSystemRuntime', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('merges runtime and changes payloads on bootstrap, then supports manual refresh', async () => {
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
    expect(result.current.eventsConnected).toBe(false);
    expect(result.current.latestEvent).toBeNull();

    mockedSystemApi.getRuntime.mockResolvedValueOnce({
      engine_type: 'windows',
      decrypt_state: 'idle',
      updated_at: '2026-03-20T04:01:00Z',
    });
    mockedSystemApi.getConfigCheck.mockResolvedValueOnce({
      deployment_target: 'docker',
      mode: 'analysis-only',
    });
    mockedSystemApi.getTasks.mockResolvedValueOnce({ items: [] });
    mockedSystemApi.getLogs.mockResolvedValueOnce({ items: [] });
    mockedSystemApi.getChanges.mockResolvedValueOnce({
      data_revision: 4,
      pending_changes: 0,
      items: [],
    });

    await act(async () => {
      await result.current.refresh();
    });

    expect(result.current.runtime?.data_revision).toBe(4);
    expect(result.current.meta.lastRefreshReason).toBe('manual');
  });

  it('keeps polling fallback disabled without SSE or interval refresh', async () => {
    mockedSystemApi.getRuntime.mockResolvedValue({});
    mockedSystemApi.getConfigCheck.mockResolvedValue({});
    mockedSystemApi.getTasks.mockResolvedValue({ items: [] });
    mockedSystemApi.getLogs.mockResolvedValue({ items: [] });
    mockedSystemApi.getChanges.mockResolvedValue({
      data_revision: 0,
      pending_changes: 0,
      items: [],
    });

    const { result } = renderHook(() => useSystemRuntime(true, 60_000));

    await waitFor(() => {
      expect(result.current.meta.pollingFallback).toBe(false);
    });
    expect(result.current.eventsConnected).toBe(false);
  });
});
