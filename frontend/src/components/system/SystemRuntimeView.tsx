import { useEffect, useMemo, useState } from 'react';
import { Activity, Download, PlayCircle, RefreshCw, Square, Wifi, WifiOff } from 'lucide-react';
import type {
  BackendStatus,
  DirectoryValidation,
  DecryptStartOptions,
  RuntimeChanges,
  RuntimeConfigCheck,
  RuntimeEvent,
  RuntimeLogEntry,
  RuntimeMeta,
  RuntimeStatus,
  RuntimeTask,
} from '../../types';

interface SystemRuntimeViewProps {
  backendStatus: BackendStatus | null;
  runtime: RuntimeStatus | null;
  configCheck?: RuntimeConfigCheck | null;
  changes: RuntimeChanges | null;
  tasks: RuntimeTask[];
  logs: RuntimeLogEntry[];
  latestEvent: RuntimeEvent | null;
  eventsConnected: boolean;
  loading: boolean;
  error: string | null;
  actionNotice: string | null;
  meta?: RuntimeMeta | null;
  defaultContactUsername?: string;
  defaultGroupUsername?: string;
  defaultSearchQuery?: string;
  defaultSearchIncludeMine?: boolean;
  onRefresh: () => void;
  onStartDecrypt: (options?: DecryptStartOptions) => void;
  onStopDecrypt: () => void;
  onReindex: () => void;
  onExportContact: (username: string, limit?: number) => void;
  onExportGroup: (username: string, date?: string) => void;
  onExportSearch: (query: string, includeMine?: boolean, limit?: number) => void;
}

const valueOrDash = (value: unknown) => {
  if (value === null || value === undefined || value === '') return '-';
  return String(value);
};

const boolLabel = (value: boolean | undefined, trueLabel = '是', falseLabel = '否') => {
  if (value === undefined) return '-';
  return value ? trueLabel : falseLabel;
};

const flattenMessages = (...groups: (string[] | undefined)[]) => {
  const merged: string[] = [];
  for (const group of groups) {
    if (!group) continue;
    for (const item of group) {
      const trimmed = item.trim();
      if (trimmed) merged.push(trimmed);
    }
  }
  return Array.from(new Set(merged));
};

const isSourceStandard = (directory?: DirectoryValidation) => {
  if (!directory) return undefined;
  if (directory.standard_layout !== undefined) return directory.standard_layout;
  if (directory.has_contact === undefined || directory.has_message === undefined) return undefined;
  return directory.has_contact && directory.has_message;
};

export const SystemRuntimeView: React.FC<SystemRuntimeViewProps> = ({
  backendStatus,
  runtime,
  configCheck,
  changes,
  tasks,
  logs,
  latestEvent,
  eventsConnected,
  loading,
  error,
  actionNotice,
  onRefresh,
  onStartDecrypt,
  onStopDecrypt,
  onReindex,
  onExportContact,
  onExportGroup,
  onExportSearch,
  meta,
  defaultContactUsername,
  defaultGroupUsername,
  defaultSearchQuery,
  defaultSearchIncludeMine = true,
}) => {
  const [contactTarget, setContactTarget] = useState('');
  const [groupTarget, setGroupTarget] = useState('');
  const [searchTarget, setSearchTarget] = useState('');
  const [platform, setPlatform] = useState('');
  const [sourceDir, setSourceDir] = useState('');
  const [analysisDir, setAnalysisDir] = useState('');
  const [workDir, setWorkDir] = useState('');
  const [command, setCommand] = useState('');
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [walEnabled, setWalEnabled] = useState(true);
  const [logSourceFilter, setLogSourceFilter] = useState('');
  const [logLevelFilter, setLogLevelFilter] = useState('');
  const [logKeywordFilter, setLogKeywordFilter] = useState('');
  const [contactLimit, setContactLimit] = useState(200);
  const [groupDate, setGroupDate] = useState('');
  const [searchIncludeMine, setSearchIncludeMine] = useState(true);
  const [searchLimit, setSearchLimit] = useState(200);

  const mergedStatus = {
    ...(backendStatus ?? {}),
    ...(runtime ?? {}),
    ...(changes ?? {}),
  };

  useEffect(() => {
    if (!contactTarget && defaultContactUsername) {
      setContactTarget(defaultContactUsername);
    }
  }, [contactTarget, defaultContactUsername]);

  useEffect(() => {
    if (!groupTarget && defaultGroupUsername) {
      setGroupTarget(defaultGroupUsername);
    }
  }, [defaultGroupUsername, groupTarget]);

  useEffect(() => {
    if (!searchTarget && defaultSearchQuery) {
      setSearchTarget(defaultSearchQuery);
    }
  }, [defaultSearchQuery, searchTarget]);

  useEffect(() => {
    setSearchIncludeMine(defaultSearchIncludeMine);
  }, [defaultSearchIncludeMine]);

  useEffect(() => {
    if (!sourceDir && configCheck?.source_dir?.path) {
      setSourceDir(configCheck.source_dir.path);
    }
  }, [configCheck?.source_dir?.path, sourceDir]);

  useEffect(() => {
    if (!analysisDir && configCheck?.analysis_dir?.path) {
      setAnalysisDir(configCheck.analysis_dir.path);
    }
  }, [analysisDir, configCheck?.analysis_dir?.path]);

  useEffect(() => {
    if (!workDir && configCheck?.work_dir?.path) {
      setWorkDir(configCheck.work_dir.path);
    }
  }, [configCheck?.work_dir?.path, workDir]);

  useEffect(() => {
    if (changes?.sync?.watch_wal !== undefined) {
      setWalEnabled(Boolean(changes.sync.watch_wal));
    }
  }, [changes?.sync?.watch_wal]);

  const logSources = useMemo(() => {
    return Array.from(new Set(logs.map((entry) => entry.source).filter((value): value is string => Boolean(value)))).sort();
  }, [logs]);

  const filteredLogs = useMemo(() => {
    return logs.filter((entry) => {
      const sourceOk = logSourceFilter ? entry.source === logSourceFilter : true;
      const levelOk = logLevelFilter ? (entry.level || '').toLowerCase().includes(logLevelFilter.toLowerCase()) : true;
      const keyword = logKeywordFilter.trim().toLowerCase();
      const keywordOk = keyword
        ? ((entry.message || '').toLowerCase().includes(keyword) ||
            (entry.fields ? Object.entries(entry.fields).some(([key, value]) =>
              key.toLowerCase().includes(keyword) || value.toLowerCase().includes(keyword)) : false))
        : true;
      return sourceOk && levelOk && keywordOk;
    });
  }, [logs, logKeywordFilter, logLevelFilter, logSourceFilter]);

  const isDockerManualMode = useMemo(() => {
    if (configCheck?.deployment_target !== 'docker') return false;
    return configCheck.mode === 'manual-stage' || configCheck.mode === 'analysis-only';
  }, [configCheck?.deployment_target, configCheck?.mode]);

  const startBlockedReasons = useMemo(() => {
    if (!configCheck) return [];

    const reasons = flattenMessages(
      configCheck.issues,
      configCheck.decrypt?.issues,
      configCheck.source_dir?.issues,
      configCheck.analysis_dir?.issues,
      configCheck.work_dir?.issues,
    );
    const sourcePath = (configCheck.source_dir?.path || '').trim();
    const analysisPath = (configCheck.analysis_dir?.path || '').trim();
    const sourceStandard = isSourceStandard(configCheck.source_dir);

    if (!sourcePath) {
      reasons.push('未配置 source_dir，请先映射标准目录');
    }
    if (configCheck.source_dir?.exists === false) {
      reasons.push('source_dir 不存在，无法同步');
    }
    if (sourcePath && sourceStandard === false) {
      reasons.push('source_dir 不是标准目录（必须包含 contact/message）');
    }
    if (sourcePath && analysisPath && sourcePath === analysisPath) {
      reasons.push('source_dir 与 analysis_dir 不能是同一目录');
    }
    if (configCheck.work_dir?.writable === false) {
      reasons.push('work_dir 不可写，请修复挂载权限');
    }
    if (configCheck.decrypt?.supported === false) {
      reasons.push('当前模式不支持解密/同步');
    }

    return Array.from(new Set(reasons));
  }, [configCheck]);

  const startButtonLabel = isDockerManualMode ? '校验并同步标准目录' : '启动解密';
  const canStartDecrypt = startBlockedReasons.length === 0;

  const configWarnings = useMemo(() => {
    if (!configCheck) return [];
    return flattenMessages(
      configCheck.warnings,
      configCheck.decrypt?.warnings,
      configCheck.sync?.warnings,
      configCheck.source_dir?.warnings,
      configCheck.analysis_dir?.warnings,
      configCheck.work_dir?.warnings,
      configCheck.sns?.warnings,
    );
  }, [configCheck]);

  const decryptOptions = (): DecryptStartOptions => ({
    platform: platform || undefined,
    source_data_dir: sourceDir || undefined,
    analysis_data_dir: analysisDir || undefined,
    work_dir: workDir || undefined,
    command: command || undefined,
    auto_refresh: isDockerManualMode ? false : autoRefresh,
    wal_enabled: isDockerManualMode ? false : walEnabled,
  });

  return (
    <div className="max-w-6xl">
      <div className="mb-8">
        <h1 className="text-2xl font-black text-[#1d1d1f] mb-1">系统与同步</h1>
        <p className="text-sm text-gray-400">运行时状态、自动解密、实时事件和导出入口统一管理。</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4 mb-6">
        <div className="bg-white rounded-2xl border border-gray-100 p-5">
          <p className="text-xs text-gray-400 mb-1">Engine</p>
          <p className="text-lg font-bold text-[#1d1d1f]">{valueOrDash(mergedStatus.engine_type)}</p>
        </div>
        <div className="bg-white rounded-2xl border border-gray-100 p-5">
          <p className="text-xs text-gray-400 mb-1">Decrypt</p>
          <p className="text-lg font-bold text-[#1d1d1f]">{valueOrDash(mergedStatus.decrypt_state)}</p>
        </div>
        <div className="bg-white rounded-2xl border border-gray-100 p-5">
          <p className="text-xs text-gray-400 mb-1">Revision</p>
          <p className="text-lg font-bold text-[#1d1d1f]">{valueOrDash(mergedStatus.data_revision)}</p>
        </div>
        <div className="bg-white rounded-2xl border border-gray-100 p-5">
          <p className="text-xs text-gray-400 mb-1">Pending Changes</p>
          <p className="text-lg font-bold text-[#1d1d1f]">{valueOrDash(mergedStatus.pending_changes)}</p>
        </div>
      </div>

      <div className="bg-white rounded-2xl border border-gray-100 p-5 mb-6">
        <div className="flex flex-wrap items-center gap-2 mb-3">
          <h2 className="text-sm font-black text-[#1d1d1f]">目录与配置校验</h2>
          <span className="text-[11px] text-gray-600 bg-gray-100 px-2 py-1 rounded-full">
            部署：{valueOrDash(configCheck?.deployment_target || runtime?.deployment_target)}
          </span>
          <span className="text-[11px] text-gray-600 bg-gray-100 px-2 py-1 rounded-full">
            模式：{valueOrDash(configCheck?.mode)}
          </span>
          <span className="text-[11px] text-gray-600 bg-gray-100 px-2 py-1 rounded-full">
            SNS：{boolLabel(configCheck?.sns?.detected, '已检测', '未检测')}
          </span>
        </div>
        <div className="grid gap-3 md:grid-cols-3">
          <div className="rounded-xl border border-gray-100 p-3">
            <p className="text-xs font-semibold text-gray-500">Source 目录</p>
            <p className="text-[11px] text-gray-500 mt-1 break-all">{valueOrDash(configCheck?.source_dir?.path)}</p>
            <p className="text-[11px] text-gray-600 mt-2">存在：{boolLabel(configCheck?.source_dir?.exists)}</p>
            <p className="text-[11px] text-gray-600">标准目录：{boolLabel(isSourceStandard(configCheck?.source_dir))}</p>
            <p className="text-[11px] text-gray-600">包含 contact：{boolLabel(configCheck?.source_dir?.has_contact)}</p>
            <p className="text-[11px] text-gray-600">包含 message：{boolLabel(configCheck?.source_dir?.has_message)}</p>
            <p className="text-[11px] text-gray-600">包含 sns：{boolLabel(configCheck?.source_dir?.has_sns)}</p>
          </div>
          <div className="rounded-xl border border-gray-100 p-3">
            <p className="text-xs font-semibold text-gray-500">Analysis 目录</p>
            <p className="text-[11px] text-gray-500 mt-1 break-all">{valueOrDash(configCheck?.analysis_dir?.path)}</p>
            <p className="text-[11px] text-gray-600 mt-2">存在：{boolLabel(configCheck?.analysis_dir?.exists)}</p>
            <p className="text-[11px] text-gray-600">包含 contact：{boolLabel(configCheck?.analysis_dir?.has_contact)}</p>
            <p className="text-[11px] text-gray-600">包含 message：{boolLabel(configCheck?.analysis_dir?.has_message)}</p>
          </div>
          <div className="rounded-xl border border-gray-100 p-3">
            <p className="text-xs font-semibold text-gray-500">Work / SNS 状态</p>
            <p className="text-[11px] text-gray-500 mt-1 break-all">{valueOrDash(configCheck?.work_dir?.path)}</p>
            <p className="text-[11px] text-gray-600 mt-2">work_dir 可写：{boolLabel(configCheck?.work_dir?.writable)}</p>
            <p className="text-[11px] text-gray-600">decrypt 支持：{boolLabel(configCheck?.decrypt?.supported)}</p>
            <p className="text-[11px] text-gray-600">sync 支持：{boolLabel(configCheck?.sync?.supported)}</p>
            <p className="text-[11px] text-gray-600">sns 就绪：{boolLabel(configCheck?.sns?.ready)}</p>
            <p className="text-[11px] text-gray-500 break-all">sns.db：{valueOrDash(configCheck?.sns?.db_path)}</p>
          </div>
        </div>
        {startBlockedReasons.length > 0 && (
          <div className="mt-3 rounded-xl border border-red-200 bg-red-50 px-3 py-2">
            <p className="text-xs font-semibold text-red-700">当前不可启动同步</p>
            <p className="text-xs text-red-600 mt-1">{startBlockedReasons.join('；')}</p>
          </div>
        )}
        {configWarnings.length > 0 && (
          <div className="mt-3 rounded-xl border border-amber-200 bg-amber-50 px-3 py-2">
            <p className="text-xs font-semibold text-amber-700">配置警告</p>
            <p className="text-xs text-amber-700 mt-1">{configWarnings.join('；')}</p>
          </div>
        )}
        {configCheck?.suggested_actions && configCheck.suggested_actions.length > 0 && (
          <div className="mt-3">
            <p className="text-xs font-semibold text-gray-600 mb-1">建议动作</p>
            <ul className="list-disc list-inside text-xs text-gray-600 space-y-1">
              {configCheck.suggested_actions.map((action, idx) => (
                <li key={`${action}-${idx}`}>{action}</li>
              ))}
            </ul>
          </div>
        )}
      </div>

      <div className="bg-white rounded-2xl border border-gray-100 p-5 mb-6">
        <div className="flex flex-wrap gap-3 mb-4">
          <button
            onClick={onRefresh}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-xl border border-gray-200 text-sm font-semibold hover:border-[#07c160] hover:text-[#07c160] transition"
          >
            <RefreshCw size={16} /> 刷新状态
          </button>
          <button
            onClick={() => onStartDecrypt(decryptOptions())}
            disabled={!canStartDecrypt}
            title={canStartDecrypt ? undefined : startBlockedReasons.join('；')}
            className={`inline-flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold transition ${
              canStartDecrypt
                ? 'bg-[#07c160] text-white hover:bg-[#06ad56]'
                : 'bg-gray-200 text-gray-500 cursor-not-allowed'
            }`}
          >
            <PlayCircle size={16} /> {startButtonLabel}
          </button>
          <button
            onClick={onStopDecrypt}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-[#1d1d1f] text-white text-sm font-semibold hover:bg-[#333] transition"
          >
            <Square size={16} /> 停止解密
          </button>
          <button
            onClick={onReindex}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-xl border border-gray-200 text-sm font-semibold hover:border-[#07c160] hover:text-[#07c160] transition"
          >
            <Activity size={16} /> 强制重建索引
          </button>
        </div>

        <div className="flex items-center gap-2 text-sm mb-2">
          {eventsConnected ? (
            <>
              <Wifi size={14} className="text-[#07c160]" />
              <span className="font-semibold text-[#07c160]">SSE 已连接</span>
            </>
          ) : (
            <>
              <WifiOff size={14} className="text-amber-500" />
              <span className="font-semibold text-amber-600">SSE 未连接（已回退轮询）</span>
            </>
          )}
        </div>

        {latestEvent && (
          <p className="text-xs text-gray-500">
            最近事件：<span className="font-semibold">{latestEvent.type}</span>
            {latestEvent.message ? ` - ${latestEvent.message}` : ''}
          </p>
        )}
        {(mergedStatus.last_change_reason || runtime?.updated_at || runtime?.last_message_at || runtime?.last_sns_at) && (
          <p className="text-xs text-gray-500 mt-2 flex flex-wrap gap-4">
            {mergedStatus.last_change_reason && (
              <span>最新变更原因：{valueOrDash(mergedStatus.last_change_reason)}</span>
            )}
            {runtime?.updated_at && (
              <span>
                最近更新时间：
                {runtime.updated_at ? new Date(runtime.updated_at).toLocaleString() : ''}
              </span>
            )}
            {runtime?.last_message_at && (
              <span>最近消息时间：{new Date(runtime.last_message_at).toLocaleString()}</span>
            )}
            {runtime?.last_sns_at && (
              <span>最近朋友圈时间：{new Date(runtime.last_sns_at).toLocaleString()}</span>
            )}
          </p>
        )}
        {actionNotice && <p className="text-xs text-[#07c160] mt-2">{actionNotice}</p>}
        {error && <p className="text-xs text-red-500 mt-2">{error}</p>}
        {loading && <p className="text-xs text-gray-400 mt-2">正在同步运行时状态...</p>}
        {meta?.pollingFallback && <p className="text-xs text-amber-600 mt-1">SSE 不可用，已回退为轮询</p>}
        <div className="flex flex-wrap gap-2 mt-3">
          {changes?.sync && (
            <>
              <span className="text-[11px] text-gray-600 bg-gray-100 px-2 py-1 rounded-full">
                watcher：{changes.sync.running ? '运行中' : '未运行'}
              </span>
              <span className="text-[11px] text-gray-600 bg-gray-100 px-2 py-1 rounded-full">
                WAL：{changes.sync.watch_wal ? '已监听' : '未监听'}
              </span>
              <span className="text-[11px] text-gray-600 bg-gray-100 px-2 py-1 rounded-full">
                revision seq：{valueOrDash(changes.sync.last_revision_seq)}
              </span>
            </>
          )}
          {meta?.lastEventAt && (
            <span className="text-[11px] text-gray-600 bg-gray-100 px-2 py-1 rounded-full">
              最近事件时间：{new Date(meta.lastEventAt).toLocaleString()}
            </span>
          )}
          {meta?.lastRefreshAt && (
            <span className="text-[11px] text-gray-600 bg-gray-100 px-2 py-1 rounded-full">
              最近刷新：{new Date(meta.lastRefreshAt).toLocaleString()}
              {meta.lastRefreshReason ? ` · ${meta.lastRefreshReason}` : ''}
            </span>
          )}
        </div>

        <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3 mt-4">
          <div>
            <p className="text-xs text-gray-500 mb-1">平台</p>
            <select
              value={platform}
              onChange={(e) => setPlatform(e.target.value)}
              className="w-full px-3 py-2 rounded-lg border border-gray-200 text-xs"
            >
              <option value="">默认</option>
              <option value="windows">Windows</option>
              <option value="macos">macOS</option>
            </select>
          </div>
          <div>
            <p className="text-xs text-gray-500 mb-1">源数据目录</p>
            <input
              value={sourceDir}
              onChange={(e) => setSourceDir(e.target.value)}
              placeholder="如 C:/wechat/source"
              className="w-full px-3 py-2 rounded-lg border border-gray-200 text-xs"
            />
          </div>
          <div>
            <p className="text-xs text-gray-500 mb-1">分析输出目录</p>
            <input
              value={analysisDir}
              onChange={(e) => setAnalysisDir(e.target.value)}
              placeholder="如 C:/wechat/analysis"
              className="w-full px-3 py-2 rounded-lg border border-gray-200 text-xs"
            />
          </div>
          <div>
            <p className="text-xs text-gray-500 mb-1">工作目录</p>
            <input
              value={workDir}
              onChange={(e) => setWorkDir(e.target.value)}
              placeholder="可选，解密临时目录"
              className="w-full px-3 py-2 rounded-lg border border-gray-200 text-xs"
            />
          </div>
          <div className="md:col-span-2">
            <p className="text-xs text-gray-500 mb-1">自定义命令</p>
            <input
              value={command}
              onChange={(e) => setCommand(e.target.value)}
              placeholder="可选，覆盖默认解密命令"
              className="w-full px-3 py-2 rounded-lg border border-gray-200 text-xs"
            />
          </div>
          {isDockerManualMode ? (
            <div className="md:col-span-2 lg:col-span-3 rounded-xl border border-gray-100 bg-gray-50 px-3 py-2">
              <p className="text-xs text-gray-600">
                当前为 Docker 手动同步模式：`auto_refresh` 与 `wal` 已按系统策略固定，不建议在容器内启用 watcher。
              </p>
            </div>
          ) : (
            <>
              <div className="flex items-center gap-3">
                <label className="flex items-center gap-2 text-xs text-gray-600">
                  <input
                    type="checkbox"
                    checked={autoRefresh}
                    onChange={(e) => setAutoRefresh(e.target.checked)}
                  />
                  自动刷新（启动 watcher）
                </label>
              </div>
              <div className="flex items-center gap-3">
                <label className="flex items-center gap-2 text-xs text-gray-600">
                  <input
                    type="checkbox"
                    checked={walEnabled}
                    onChange={(e) => setWalEnabled(e.target.checked)}
                  />
                  监听 WAL
                </label>
              </div>
            </>
          )}
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-2 mb-6">
        <div className="bg-white rounded-2xl border border-gray-100 p-5">
          <h2 className="text-sm font-black text-[#1d1d1f] mb-3">任务队列</h2>
          <div className="space-y-2 max-h-56 overflow-auto">
            {tasks.length === 0 ? (
              <p className="text-xs text-gray-400">暂无任务</p>
            ) : (
              tasks.map((task, idx) => (
                <div key={`${task.id ?? idx}`} className="rounded-xl border border-gray-100 px-3 py-2">
                  <p className="text-xs font-semibold text-[#1d1d1f]">
                    {valueOrDash(task.type)} · {valueOrDash(task.status)}
                  </p>
                  <p className="text-xs text-gray-400">{valueOrDash(task.message || task.detail)}</p>
                  {task.error && (
                    <p className="text-[11px] text-red-500 mt-1 break-words">{task.error}</p>
                  )}
                  {(task.work_dir || task.command_summary) && (
                    <p className="text-[11px] text-gray-500 mt-1">
                      {task.work_dir ? `work_dir=${task.work_dir} ` : ''}
                      {task.command_summary ? `cmd=${task.command_summary}` : ''}
                    </p>
                  )}
                  {task.updated_at && (
                    <p className="text-[11px] text-gray-500 mt-1">
                      更新时间：{new Date(task.updated_at).toLocaleString()}
                    </p>
                  )}
                </div>
              ))
            )}
          </div>
        </div>

        <div className="bg-white rounded-2xl border border-gray-100 p-5">
          <h2 className="text-sm font-black text-[#1d1d1f] mb-3">最新日志</h2>
          <div className="grid gap-2 md:grid-cols-3 mb-3">
            <select
              value={logSourceFilter}
              onChange={(e) => setLogSourceFilter(e.target.value)}
              className="w-full px-3 py-2 rounded-lg border border-gray-200 text-xs"
            >
              <option value="">全部 source</option>
              {logSources.map((source) => (
                <option key={source} value={source}>{source}</option>
              ))}
            </select>
            <input
              value={logLevelFilter}
              onChange={(e) => setLogLevelFilter(e.target.value)}
              placeholder="按 level 过滤"
              className="w-full px-3 py-2 rounded-lg border border-gray-200 text-xs"
            />
            <input
              value={logKeywordFilter}
              onChange={(e) => setLogKeywordFilter(e.target.value)}
              placeholder="按关键字/字段过滤"
              className="w-full px-3 py-2 rounded-lg border border-gray-200 text-xs"
            />
          </div>
          <div className="space-y-2 max-h-56 overflow-auto">
            {filteredLogs.length === 0 ? (
              <p className="text-xs text-gray-400">暂无日志</p>
            ) : (
              filteredLogs.map((entry, idx) => (
                <div key={`${entry.time ?? idx}-${idx}`} className="rounded-xl border border-gray-100 px-3 py-2">
                  <p className="text-[11px] text-gray-500">
                    #{valueOrDash(entry.id)} · {valueOrDash(entry.time)} · {valueOrDash(entry.level)} · {valueOrDash(entry.source)}
                  </p>
                  <p className="text-xs text-[#1d1d1f] break-words">{entry.message}</p>
                  {entry.fields && (
                    <div className="flex flex-wrap gap-1 mt-1">
                      {Object.entries(entry.fields).map(([k, v]) => (
                        <span
                          key={k}
                          className="text-[10px] text-gray-600 bg-gray-100 px-2 py-1 rounded-full"
                        >
                          {k}: {v}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      <div className="bg-white rounded-2xl border border-gray-100 p-5">
        <h2 className="text-sm font-black text-[#1d1d1f] mb-3">ChatLab 导出</h2>
        <div className="grid gap-4 md:grid-cols-3">
          <div className="rounded-xl border border-gray-100 p-3">
            <p className="text-xs font-semibold text-gray-500 mb-2">联系人导出</p>
            <input
              value={contactTarget}
              onChange={(e) => setContactTarget(e.target.value)}
              placeholder={defaultContactUsername ? `联系人 username（默认 ${defaultContactUsername}）` : '联系人 username'}
              className="w-full px-3 py-2 rounded-lg border border-gray-200 text-xs focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
            />
            <input
              type="number"
              min={1}
              value={contactLimit}
              onChange={(e) => setContactLimit(Number(e.target.value) || 0)}
              placeholder="Limit (默认 200)"
              className="w-full mt-2 px-3 py-2 rounded-lg border border-gray-200 text-xs focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
            />
            <button
              onClick={() => onExportContact(contactTarget, contactLimit || undefined)}
              className="mt-2 inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-[#07c160] text-white text-xs font-semibold"
            >
              <Download size={12} /> 导出
            </button>
          </div>
          <div className="rounded-xl border border-gray-100 p-3">
            <p className="text-xs font-semibold text-gray-500 mb-2">群聊导出</p>
            <input
              value={groupTarget}
              onChange={(e) => setGroupTarget(e.target.value)}
              placeholder={defaultGroupUsername ? `群聊 username（默认 ${defaultGroupUsername}）` : '群聊 username'}
              className="w-full px-3 py-2 rounded-lg border border-gray-200 text-xs focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
            />
            <input
              type="text"
              value={groupDate}
              onChange={(e) => setGroupDate(e.target.value)}
              placeholder="日期（可选，如 2024-01-01）"
              className="w-full mt-2 px-3 py-2 rounded-lg border border-gray-200 text-xs focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
            />
            <button
              onClick={() => onExportGroup(groupTarget, groupDate || undefined)}
              className="mt-2 inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-[#07c160] text-white text-xs font-semibold"
            >
              <Download size={12} /> 导出
            </button>
          </div>
          <div className="rounded-xl border border-gray-100 p-3">
            <p className="text-xs font-semibold text-gray-500 mb-2">搜索结果导出</p>
            <input
              value={searchTarget}
              onChange={(e) => setSearchTarget(e.target.value)}
              placeholder={defaultSearchQuery ? `关键词（默认 ${defaultSearchQuery}）` : '关键词'}
              className="w-full px-3 py-2 rounded-lg border border-gray-200 text-xs focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
            />
            <div className="flex items-center gap-2 mt-2">
              <input
                type="checkbox"
                checked={searchIncludeMine}
                onChange={(e) => setSearchIncludeMine(e.target.checked)}
              />
              <span className="text-xs text-gray-600">包含自己的消息</span>
            </div>
            <input
              type="number"
              min={1}
              value={searchLimit}
              onChange={(e) => setSearchLimit(Number(e.target.value) || 0)}
              placeholder="Limit (默认 200)"
              className="w-full mt-2 px-3 py-2 rounded-lg border border-gray-200 text-xs focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
            />
            <button
              onClick={() => onExportSearch(searchTarget, searchIncludeMine, searchLimit || undefined)}
              className="mt-2 inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-[#07c160] text-white text-xs font-semibold"
            >
              <Download size={12} /> 导出
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
