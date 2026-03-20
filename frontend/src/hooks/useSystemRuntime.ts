import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { systemApi } from '../services/api';
import type {
  RuntimeChanges,
  RuntimeConfigCheck,
  RuntimeEvent,
  RuntimeLogEntry,
  RuntimeMeta,
  RuntimeStatus,
  RuntimeTask,
} from '../types';

type MaybeArrayPayload<T> = T[] | { items?: T[]; tasks?: T[]; logs?: T[] };

const parseArrayPayload = <T>(value: MaybeArrayPayload<T>): T[] => {
  if (Array.isArray(value)) return value;
  if (Array.isArray(value.items)) return value.items;
  if (Array.isArray(value.tasks)) return value.tasks;
  if (Array.isArray(value.logs)) return value.logs;
  return [];
};

const parseEvent = (raw: string): RuntimeEvent => {
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    return {
      type: String(parsed.type ?? 'message'),
      id: typeof parsed.id === 'string' || typeof parsed.id === 'number' ? String(parsed.id) : undefined,
      at: typeof parsed.at === 'string' ? parsed.at : undefined,
      revision: typeof parsed.revision === 'string' || typeof parsed.revision === 'number' ? parsed.revision : undefined,
      message: typeof parsed.message === 'string' ? parsed.message : raw,
      payload: parsed,
    };
  } catch {
    return { type: 'message', message: raw };
  }
};

export const useSystemRuntime = (enabled = true, pollInterval = 5000) => {
  const [runtime, setRuntime] = useState<RuntimeStatus | null>(null);
  const [configCheck, setConfigCheck] = useState<RuntimeConfigCheck | null>(null);
  const [changes, setChanges] = useState<RuntimeChanges | null>(null);
  const [tasks, setTasks] = useState<RuntimeTask[]>([]);
  const [logs, setLogs] = useState<RuntimeLogEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [eventsConnected, setEventsConnected] = useState(false);
  const [latestEvent, setLatestEvent] = useState<RuntimeEvent | null>(null);
  const [lastEventAt, setLastEventAt] = useState<string | undefined>(undefined);
  const [lastRefreshAt, setLastRefreshAt] = useState<string | undefined>(undefined);
  const [lastRefreshReason, setLastRefreshReason] = useState<string | undefined>(undefined);
  const [pollingFallback, setPollingFallback] = useState(false);
  const sourceRef = useRef<EventSource | null>(null);

  const refresh = useCallback(async (reason = 'poll') => {
    if (!enabled) return;
    setLoading(true);
    try {
      const [runtimeRes, configCheckRes, taskRes, logRes, changesRes] = await Promise.all([
        systemApi.getRuntime(),
        systemApi.getConfigCheck(),
        systemApi.getTasks(),
        systemApi.getLogs(200),
        systemApi.getChanges(),
      ]);
      setConfigCheck(configCheckRes ?? null);
      setChanges(changesRes ?? null);
      setRuntime({
        ...(runtimeRes ?? {}),
        ...(changesRes ?? {}),
      });
      setTasks(parseArrayPayload(taskRes));
      const parsedLogs = parseArrayPayload(logRes);
      setLogs(parsedLogs.slice(0, 200));
      setError(null);
      setLastRefreshAt(new Date().toISOString());
      setLastRefreshReason(reason);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to fetch runtime status';
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [enabled]);

  useEffect(() => {
    if (!enabled) return;
    void refresh('bootstrap');
    const timer = window.setInterval(() => {
      void refresh(sourceRef.current ? 'poll-backfill' : 'poll');
    }, pollInterval);
    return () => window.clearInterval(timer);
  }, [enabled, pollInterval, refresh]);

  useEffect(() => {
    if (!enabled) return;
    const source = systemApi.createEventsSource();
    if (!source) {
      setPollingFallback(true);
      return;
    }
    sourceRef.current = source;
    setPollingFallback(false);

    source.onopen = () => {
      setEventsConnected(true);
      setPollingFallback(false);
    };
    source.onmessage = (event) => {
      const parsed = parseEvent(event.data);
      setLatestEvent(parsed);
      setLastEventAt(parsed.at ?? new Date().toISOString());
      setPollingFallback(false);
      void refresh('sse');
    };
    source.onerror = () => {
      setEventsConnected(false);
      setPollingFallback(true);
    };

    return () => {
      setEventsConnected(false);
      source.close();
      sourceRef.current = null;
    };
  }, [enabled, refresh]);

  const summary = useMemo(() => {
    if (!runtime) return '未获取到运行时状态';
    const engine = runtime.engine_type || 'unknown';
    const decrypt = runtime.decrypt_state || 'unknown';
    const revision = runtime.data_revision ?? '-';
    return `engine=${engine}, decrypt=${decrypt}, revision=${revision}`;
  }, [runtime]);

  return {
    runtime,
    configCheck,
    changes,
    tasks,
    logs,
    loading,
    error,
    eventsConnected,
    latestEvent,
    meta: { lastEventAt, lastRefreshAt, pollingFallback, lastRefreshReason } satisfies RuntimeMeta,
    summary,
    refresh,
  };
};
