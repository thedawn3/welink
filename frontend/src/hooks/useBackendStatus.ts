/**
 * 后端状态监控 Hook
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { globalApi } from '../services/api';
import type { BackendStatus } from '../types';

export const useBackendStatus = (_pollInterval = 1000) => {
  const [status, setStatus] = useState<BackendStatus | null>(null);
  const [backendReady, setBackendReady] = useState(false); // 后端可连通
  const [error, setError] = useState<Error | null>(null);
  const mountedRef = useRef(false);

  const fetchStatus = useCallback(async () => {
    try {
      const data = await globalApi.getStatus();
      setStatus(data);
      setBackendReady(true);
      setError(null);
    } catch (err) {
      setError(err as Error);
      console.error('Failed to fetch backend status:', err);
    }
  }, []);

  const startPolling = useCallback(() => {
    void fetchStatus();
  }, [fetchStatus]);

  useEffect(() => {
    if (mountedRef.current) return;
    mountedRef.current = true;
    void fetchStatus();
  }, [fetchStatus]);

  return {
    status,
    error,
    backendReady,
    startPolling,
    isInitialized: status?.is_initialized ?? false,
    isIndexing: status?.is_indexing ?? false,
    totalCached: status?.total_cached ?? 0,
  };
};
