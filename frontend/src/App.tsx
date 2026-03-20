/**
 * WeLink - 微信聊天数据分析平台
 * 重构版本 - 组件化 + 微信风格设计
 */

import { useState, useMemo, useEffect, useCallback } from 'react';
import { Users, MessageSquare, Flame, Snowflake } from 'lucide-react';

// Layout Components
import { Sidebar } from './components/layout/Sidebar';
import { Header } from './components/layout/Header';

// Dashboard Components
import { KPICard } from './components/dashboard/KPICard';
import { RelationshipHeatmap } from './components/dashboard/RelationshipHeatmap';
import { MonthlyTrendChart } from './components/dashboard/MonthlyTrendChart';
import { HourlyHeatmap } from './components/dashboard/HourlyHeatmap';
import { DatabaseView } from './components/dashboard/DatabaseView';
import { GlobalSearchPanel, type GlobalSearchFilterType } from './components/dashboard/GlobalSearchPanel';
import { LateNightRanking } from './components/dashboard/LateNightRanking';
import { RelationOverviewSection, type RelationOverviewListType } from './components/dashboard/RelationOverviewSection';
import { ControversyOverviewSection, type ControversyBoardKey } from './components/dashboard/ControversyOverviewSection';
import { GroupsView, GroupDetailModal } from './components/groups/GroupsView';
import { useDarkMode } from './hooks/useDarkMode';
import {
  ContactsPage,
  type ContactActivityFilter,
  type ContactCategoryFilter,
  type ContactGenderFilter,
  type ContactSortKey,
} from './components/contacts/ContactsPage';
import { SnsSearchPage } from './components/sns/SnsSearchPage';

// Contact Components
import { ContactModal } from './components/contact/ContactModal';

// Common Components
import { InitializingScreen } from './components/common/InitializingScreen';
import { WelcomePage } from './components/common/WelcomePage';

// Privacy Components
import { PrivacyView } from './components/privacy/PrivacyView';
import { SystemRuntimeView } from './components/system/SystemRuntimeView';

// Hooks
import { useContacts } from './hooks/useContacts';
import { useGlobalStats } from './hooks/useGlobalStats';
import { useBackendStatus } from './hooks/useBackendStatus';
import { usePrivacySettings } from './hooks/usePrivacySettings';
import { useSystemRuntime } from './hooks/useSystemRuntime';

// Types
import type {
  TabType,
  DecryptStartOptions,
  ContactStats,
  HealthStatus,
  TimeRange,
  GroupInfo,
  GlobalSearchHit,
  RelationOverviewGrouped,
  ControversyOverviewGrouped,
  ChatLabExportResponse,
} from './types';

// Utils
import { formatCompactNumber } from './utils/formatters';
import { contactsApi, exportApi, globalApi, groupsApi, relationsApi, systemApi } from './services/api';

const ALL_TIME: TimeRange = { from: null, to: null, label: '全部' };
type DashboardRelationMode = 'objective' | 'controversy';
type ContactModalView = 'timeline' | 'wordcloud' | 'detail' | 'sentiment' | 'search' | 'analysis';

const EMPTY_RELATION_GROUPED: RelationOverviewGrouped = {
  all: { warming: [], cooling: [], initiative: [], fast_reply: [] },
  male: { warming: [], cooling: [], initiative: [], fast_reply: [] },
  female: { warming: [], cooling: [], initiative: [], fast_reply: [] },
};

function getContactLastMessageTs(contact: ContactStats) {
  const ts = new Date(contact.last_message_time).getTime();
  return Number.isFinite(ts) ? ts : 0;
}

function getContactActivityStatus(contact: ContactStats): ContactActivityFilter {
  if (contact.total_messages === 0) return 'cold';
  const lastTs = getContactLastMessageTs(contact);
  if (!lastTs) return 'warm';
  return Date.now() - lastTs < 7 * 86400 * 1000 ? 'hot' : 'warm';
}

function getContactDisplayName(contact: ContactStats) {
  return contact.remark || contact.nickname || contact.username;
}

function App() {
  const { dark, toggle: toggleDark } = useDarkMode();

  // State — 从 localStorage 恢复，刷新不回到欢迎页
  const [activeTab, setActiveTab] = useState<TabType>('dashboard');
  const [contactSearch, setContactSearch] = useState('');
  const [contactActivityFilter, setContactActivityFilter] = useState<ContactActivityFilter>('all');
  const [contactCategoryFilter, setContactCategoryFilter] = useState<ContactCategoryFilter>('all');
  const [contactGenderFilter, setContactGenderFilter] = useState<ContactGenderFilter>('all');
  const [contactSort, setContactSort] = useState<ContactSortKey>('messages_desc');
  const [dashboardRelationMode, setDashboardRelationMode] = useState<DashboardRelationMode>('objective');
  const [globalQuery, setGlobalQuery] = useState('');
  const [globalResults, setGlobalResults] = useState<GlobalSearchHit[]>([]);
  const [globalSearchLoading, setGlobalSearchLoading] = useState(false);
  const [globalSearchTouched, setGlobalSearchTouched] = useState(false);
  const [globalIncludeMine, setGlobalIncludeMine] = useState(true);
  const [globalFilterType, setGlobalFilterType] = useState<GlobalSearchFilterType>('all');
  const [selectedContact, setSelectedContact] = useState<ContactStats | null>(null);
  const [selectedContactView, setSelectedContactView] = useState<ContactModalView>('wordcloud');
  const [selectedControversyLabel, setSelectedControversyLabel] = useState<string | undefined>(undefined);
  const [selectedGroup, setSelectedGroup] = useState<GroupInfo | null>(null);
  const [timeRange, setTimeRange] = useState<TimeRange>(() => {
    try {
      const saved = localStorage.getItem('welink_timeRange');
      return saved ? JSON.parse(saved) : ALL_TIME;
    } catch { return ALL_TIME; }
  });
  const [initLoading, setInitLoading] = useState(false);
  const [systemActionNotice, setSystemActionNotice] = useState<string | null>(null);
  const [hasStarted, setHasStarted] = useState(() => {
    return localStorage.getItem('welink_hasStarted') === 'true';
  });

  // Backend Status Hook
  const { status: backendStatus, isInitialized, isIndexing, backendReady, startPolling } = useBackendStatus(1000);
  const {
    runtime,
    configCheck,
    changes: runtimeChanges,
    tasks: runtimeTasks,
    logs: runtimeLogs,
    eventsConnected,
    loading: runtimeLoading,
    error: runtimeError,
    meta: runtimeMeta,
    refresh: refreshRuntime,
  } = useSystemRuntime(backendReady, 5000);

  // Privacy settings
  const {
    blockedUsers,
    blockedGroups,
    addBlockedUser,
    removeBlockedUser,
    addBlockedGroup,
    removeBlockedGroup,
  } = usePrivacySettings();

  // Data Hooks (只在初始化完成后启动)
  const {
    contacts: allContacts,
    loading: contactsLoading,
    refresh: refreshContacts,
  } = useContacts(isInitialized, false, 15000);
  const {
    stats: rawGlobalStats,
    refresh: refreshGlobalStats,
  } = useGlobalStats(isInitialized, false, 15000);
  const [allGroups, setAllGroups] = useState<GroupInfo[]>([]);
  const [relationOverview, setRelationOverview] = useState<RelationOverviewGrouped | null>(null);
  const [relationOverviewLoading, setRelationOverviewLoading] = useState(false);
  const [controversyOverview, setControversyOverview] = useState<ControversyOverviewGrouped | null>(null);
  const [controversyOverviewLoading, setControversyOverviewLoading] = useState(false);
  const loadGroups = useCallback(async () => {
    if (!isInitialized) return;
    try {
      const list = await groupsApi.getList();
      setAllGroups(list || []);
    } catch (error) {
      console.error('Failed to fetch group list', error);
    }
  }, [isInitialized]);

  const loadRelationData = useCallback(async () => {
    if (!isInitialized) return;
    setRelationOverviewLoading(true);
    setControversyOverviewLoading(true);
    try {
      const [overview, controversy] = await Promise.all([
        relationsApi.getOverview().catch((error) => {
          console.error('Failed to fetch relation overview', error);
          return null;
        }),
        relationsApi.getControversyOverview().catch((error) => {
          console.error('Failed to fetch controversy overview', error);
          return null;
        }),
      ]);
      setRelationOverview(overview);
      setControversyOverview(controversy);
    } finally {
      setRelationOverviewLoading(false);
      setControversyOverviewLoading(false);
    }
  }, [isInitialized]);

  useEffect(() => {
    void loadGroups();
  }, [loadGroups]);

  useEffect(() => {
    void loadRelationData();
  }, [loadRelationData]);

  const refreshDerivedData = useCallback(() => {
    refreshContacts();
    refreshGlobalStats();
    void loadGroups();
    void loadRelationData();
  }, [refreshContacts, refreshGlobalStats, loadGroups, loadRelationData]);

  const statsLoading = contactsLoading;

  useEffect(() => {
    if (!selectedContact) return;
    const next = allContacts.find((item) => item.username === selectedContact.username);
    if (!next) return;
    if (next !== selectedContact) {
      setSelectedContact(next);
    }
  }, [allContacts, selectedContact]);

  useEffect(() => {
    if (!selectedGroup) return;
    const next = allGroups.find((item) => item.username === selectedGroup.username);
    if (!next) return;
    if (next !== selectedGroup) {
      setSelectedGroup(next);
    }
  }, [allGroups, selectedGroup]);

  // 屏蔽过滤后的联系人列表
  const contacts = useMemo(() => {
    if (blockedUsers.length === 0) return allContacts;
    return allContacts.filter(
      (c) => !blockedUsers.some(
        (b) => b === c.username || b === c.nickname || b === c.remark
      )
    );
  }, [allContacts, blockedUsers]);

  // 被屏蔽联系人的显示名集合（用于过滤深夜排行，排行只有 name 无 username）
  const blockedDisplayNames = useMemo(() => {
    if (blockedUsers.length === 0) return new Set<string>();
    return new Set(
      allContacts
        .filter((c) => blockedUsers.some((b) => b === c.username || b === c.nickname || b === c.remark))
        .map((c) => c.remark || c.nickname || c.username)
    );
  }, [allContacts, blockedUsers]);

  const blockedUsernames = useMemo(() => {
    if (blockedUsers.length === 0) return new Set<string>();
    const usernames = new Set<string>();
    for (const contact of allContacts) {
      if (blockedUsers.some((value) => value === contact.username || value === contact.nickname || value === contact.remark)) {
        usernames.add(contact.username);
      }
    }
    for (const value of blockedUsers) {
      usernames.add(value);
    }
    return usernames;
  }, [allContacts, blockedUsers]);

  // 屏蔽过滤后的全局统计（深夜排行中过滤被屏蔽联系人）
  const globalStats = useMemo(() => {
    if (!rawGlobalStats || blockedDisplayNames.size === 0) return rawGlobalStats;
    return {
      ...rawGlobalStats,
      late_night_ranking: rawGlobalStats.late_night_ranking.filter(
        (e) => !blockedDisplayNames.has(e.name)
      ),
    };
  }, [rawGlobalStats, blockedDisplayNames]);

  const visibleRelationOverview = useMemo(() => {
    if (!relationOverview || blockedUsernames.size === 0) return relationOverview;
    const filterItems = (items: RelationOverviewGrouped['all']['warming']) =>
      items.filter((item) => !blockedUsernames.has(item.username));
    const filterGroup = (group: RelationOverviewGrouped['all']) => ({
      warming: filterItems(group.warming),
      cooling: filterItems(group.cooling),
      initiative: filterItems(group.initiative),
      fast_reply: filterItems(group.fast_reply),
    });
    return {
      all: filterGroup(relationOverview.all),
      male: filterGroup(relationOverview.male),
      female: filterGroup(relationOverview.female),
    };
  }, [relationOverview, blockedUsernames]);

  const visibleControversyOverview = useMemo(() => {
    if (!controversyOverview || blockedUsernames.size === 0) return controversyOverview;
    const filterItems = (items: ControversyOverviewGrouped['all']['simp']) =>
      items.filter((item) => !blockedUsernames.has(item.username));
    const filterGroup = (group: ControversyOverviewGrouped['all']) => ({
      simp: filterItems(group.simp),
      ambiguity: filterItems(group.ambiguity),
      faded: filterItems(group.faded),
      tool_person: filterItems(group.tool_person),
      cold_violence: filterItems(group.cold_violence),
    });
    return {
      all: filterGroup(controversyOverview.all),
      male: filterGroup(controversyOverview.male),
      female: filterGroup(controversyOverview.female),
    };
  }, [controversyOverview, blockedUsernames]);

  // Computed Values
  const filteredContacts = useMemo(() => {
    const searchLower = contactSearch.trim().toLowerCase();
    const next = contacts.filter((contact) => {
      if (contactActivityFilter !== 'all' && getContactActivityStatus(contact) !== contactActivityFilter) {
        return false;
      }
      if (contactCategoryFilter !== 'all') {
        if (contactCategoryFilter === 'deleted' && !contact.is_deleted) return false;
        if (
          contactCategoryFilter === 'normal' &&
          contact.is_deleted
        ) {
          return false;
        }
      }
      if (contactGenderFilter !== 'all' && (contact.gender ?? 'unknown') !== contactGenderFilter) {
        return false;
      }
      if (!searchLower) {
        return true;
      }
      return `${contact.remark}${contact.nickname}${contact.username}`.toLowerCase().includes(searchLower);
    });

    next.sort((left, right) => {
      if (contactSort === 'last_message_desc') {
        return getContactLastMessageTs(right) - getContactLastMessageTs(left);
      }
      if (contactSort === 'shared_groups_desc') {
        const groupDiff = (right.shared_groups_count ?? 0) - (left.shared_groups_count ?? 0);
        if (groupDiff !== 0) return groupDiff;
        return right.total_messages - left.total_messages;
      }
      if (contactSort === 'name_asc') {
        return getContactDisplayName(left).localeCompare(getContactDisplayName(right), 'zh-Hans-CN');
      }
      if (right.total_messages !== left.total_messages) {
        return right.total_messages - left.total_messages;
      }
      return getContactLastMessageTs(right) - getContactLastMessageTs(left);
    });
    return next;
  }, [contactActivityFilter, contactCategoryFilter, contactGenderFilter, contactSearch, contactSort, contacts]);

  const visibleGlobalResults = useMemo(() => {
    return globalResults.filter((item) => {
      if (!item.is_group) {
        return !blockedUsers.some((blocked) => blocked === item.username || blocked === item.name);
      }
      return !blockedGroups.some((blocked) => blocked === item.username || blocked === item.name);
    });
  }, [blockedGroups, blockedUsers, globalResults]);

  const healthStatus: HealthStatus = useMemo(() => {
    if (!contacts.length) return { hot: 0, warm: 0, cold: 0 };

    const now = Date.now() / 1000;
    let hot = 0,
      warm = 0,
      cold = 0;

    contacts.forEach((c) => {
      const ts = new Date(c.last_message_time).getTime() / 1000;
      if (c.total_messages === 0) {
        cold++;
      } else if (now - ts < 7 * 86400) {
        hot++;
      } else {
        warm++;
      }
    });

    return { hot, warm, cold };
  }, [contacts]);

  const mergedRuntimeStatus = useMemo(() => {
    return {
      ...(backendStatus ?? {}),
      ...(runtime ?? {}),
    };
  }, [backendStatus, runtime]);

  // Handlers
  const handleContactClick = (contact: ContactStats) => {
    setSelectedGroup(null);
    setSelectedContactView('wordcloud');
    setSelectedControversyLabel(undefined);
    setSelectedContact(contact);
  };

  const handleCloseModal = () => {
    setSelectedContact(null);
    setSelectedControversyLabel(undefined);
  };

  const runGlobalSearch = useCallback(async (query: string, includeMine: boolean) => {
    const trimmed = query.trim();
    setGlobalSearchTouched(true);
    if (!trimmed) {
      setGlobalResults([]);
      return;
    }

    setGlobalSearchLoading(true);
    try {
      const results = await contactsApi.searchAllMessages(trimmed, includeMine, 200);
      setGlobalResults(results ?? []);
    } catch (error) {
      console.error('Global search failed', error);
      setGlobalResults([]);
    } finally {
      setGlobalSearchLoading(false);
    }
  }, []);

  const handleOpenSearchContact = useCallback((username: string) => {
    const contact = allContacts.find((item) => item.username === username);
    if (contact) {
      setSelectedGroup(null);
      setSelectedContactView('search');
      setSelectedControversyLabel(undefined);
      setSelectedContact(contact);
    }
  }, [allContacts]);

  const handleOpenRelationContact = useCallback((username: string, view: ContactModalView, label?: string) => {
    const contact = allContacts.find((item) => item.username === username);
    if (!contact) return;
    setSelectedGroup(null);
    setSelectedContactView(view);
    setSelectedControversyLabel(label);
    setSelectedContact(contact);
  }, [allContacts]);

  const handleOpenSearchGroup = useCallback((username: string) => {
    const group = allGroups.find((item) => item.username === username);
    if (group) {
      setSelectedContact(null);
      setSelectedGroup(group);
      return;
    }

    const hit = visibleGlobalResults.find((item) => item.username === username && item.is_group);
    if (!hit) return;
    setSelectedContact(null);
    setSelectedGroup({
      username,
      name: hit.name,
      small_head_url: '',
      total_messages: 0,
      last_message_time: `${hit.date} ${hit.time}`,
    });
  }, [allGroups, visibleGlobalResults]);

  const handleStart = async (from: number | null, to: number | null, label: string) => {
    setInitLoading(true);
    try {
      await globalApi.init(from, to);
      const range = { from, to, label };
      setTimeRange(range);
      setHasStarted(true);
      localStorage.setItem('welink_hasStarted', 'true');
      localStorage.setItem('welink_timeRange', JSON.stringify(range));
      startPolling(); // 重新开始轮询，等待 is_initialized 变为 true
    } catch (e) {
      console.error('Init failed', e);
    } finally {
      setInitLoading(false);
    }
  };

  const handleReselect = () => {
    setHasStarted(false);
    setTimeRange(ALL_TIME);
    localStorage.removeItem('welink_hasStarted');
    localStorage.removeItem('welink_timeRange');
  };

  const runRuntimeAction = useCallback(async (fn: () => Promise<unknown>, successText: string) => {
    try {
      await fn();
      setSystemActionNotice(successText);
      await refreshRuntime();
    } catch (error) {
      const message = error instanceof Error ? error.message : '操作失败';
      setSystemActionNotice(message);
    }
  }, [refreshRuntime]);

  const handleStartDecrypt = useCallback((options?: DecryptStartOptions) => {
    const summary = [
      options?.platform ? `平台 ${options.platform}` : null,
      options?.auto_refresh === false ? '自动刷新关闭' : '自动刷新开启',
      options?.wal_enabled === false ? 'WAL 关闭' : 'WAL 开启',
    ].filter(Boolean).join(' · ');
    return runRuntimeAction(
      () => systemApi.startDecrypt(options ?? {}),
      summary ? `已触发解密启动（${summary}）` : '已触发解密启动',
    );
  }, [runRuntimeAction]);

  const handleStopDecrypt = useCallback(() => {
    return runRuntimeAction(() => systemApi.stopDecrypt(), '已触发解密停止');
  }, [runRuntimeAction]);

  const handleReindex = useCallback(() => {
    return runRuntimeAction(async () => {
      await systemApi.reindex(timeRange.from, timeRange.to);
      startPolling();
    }, `已触发索引重建（${timeRange.label}）`);
  }, [runRuntimeAction, startPolling, timeRange.from, timeRange.label, timeRange.to]);

  const downloadChatLabPayload = useCallback((payload: unknown, fallbackName: string) => {
    const record = (payload && typeof payload === 'object') ? payload as Record<string, unknown> : null;
    const fileName = typeof record?.file_name === 'string' ? record.file_name : fallbackName;
    const mimeType = typeof record?.mime_type === 'string' ? record.mime_type : 'application/json';
    const content = Object.prototype.hasOwnProperty.call(record ?? {}, 'data') ? record?.data : payload;
    const blob = new Blob([JSON.stringify(content, null, 2)], { type: mimeType });
    const url = window.URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = fileName;
    anchor.click();
    window.URL.revokeObjectURL(url);
  }, []);

  const formatExportNotice = useCallback((kind: string, target: string, payload: ChatLabExportResponse) => {
    const summary = payload.summary;
    const messageCount = summary?.message_count;
    const memberCount = summary?.member_count;
    const conversationName = summary?.conversation_name;
    const countText = [
      typeof messageCount === 'number' ? `${messageCount} 条消息` : null,
      typeof memberCount === 'number' ? `${memberCount} 位成员` : null,
    ].filter(Boolean).join(' / ');
    const displayTarget = conversationName || target;
    return `${kind} ChatLab 导出完成：${displayTarget}${countText ? `（${countText}）` : ''}`;
  }, []);

  const handleExportContact = useCallback(async (username: string, limit = 200) => {
    const target = username.trim();
    if (!target) {
      setSystemActionNotice('请输入联系人 username');
      return;
    }
    try {
      const data = await exportApi.exportChatLabContact(target, limit);
      downloadChatLabPayload(data, `chatlab-contact-${target}.json`);
      setSystemActionNotice(formatExportNotice('联系人', target, data));
    } catch (error) {
      const message = error instanceof Error ? error.message : '导出失败';
      setSystemActionNotice(message);
    }
  }, [downloadChatLabPayload, formatExportNotice]);

  const handleExportGroup = useCallback(async (username: string, date?: string) => {
    const target = username.trim();
    if (!target) {
      setSystemActionNotice('请输入群聊 username');
      return;
    }
    try {
      const data = await exportApi.exportChatLabGroup(target, date);
      downloadChatLabPayload(data, `chatlab-group-${target}.json`);
      setSystemActionNotice(formatExportNotice('群聊', target, data));
    } catch (error) {
      const message = error instanceof Error ? error.message : '导出失败';
      setSystemActionNotice(message);
    }
  }, [downloadChatLabPayload, formatExportNotice]);

  const handleExportSearch = useCallback(async (query: string, includeMine = true, limit = 200) => {
    const target = query.trim();
    if (!target) {
      setSystemActionNotice('请输入搜索关键词');
      return;
    }
    try {
      const data = await exportApi.exportChatLabSearch(target, includeMine, limit);
      downloadChatLabPayload(data, `chatlab-search-${Date.now()}.json`);
      setSystemActionNotice(formatExportNotice('搜索结果', target, data));
    } catch (error) {
      const message = error instanceof Error ? error.message : '导出失败';
      setSystemActionNotice(message);
    }
  }, [downloadChatLabPayload, formatExportNotice]);

  useEffect(() => {
    if (!systemActionNotice) return;
    const timer = window.setTimeout(() => setSystemActionNotice(null), 5000);
    return () => window.clearTimeout(timer);
  }, [systemActionNotice]);

  // 后端未连通时等待
  if (!backendReady) {
    return <InitializingScreen message="正在连接后端服务..." onRefresh={() => { void startPolling(); }} />;
  }

  // 用户还没选时间范围，或主动点了「重新选择」
  if (!hasStarted) {
    return <WelcomePage onStart={handleStart} loading={initLoading} />;
  }

  // 已选择时间范围，等待索引完成
  if (!isInitialized || isIndexing) {
    return (
      <InitializingScreen
        message={`正在建立索引（${timeRange.label}）...`}
        onRefresh={() => { void startPolling(); }}
        onReset={handleReselect}
      />
    );
  }

  return (
    <div className="flex h-screen dk-page bg-[#f8f9fb] dk-text text-[#1d1d1f] font-sans overflow-hidden">
      {/* Sidebar */}
      <Sidebar activeTab={activeTab} onTabChange={setActiveTab} dark={dark} onToggleDark={toggleDark} />

      {/* Main Content */}
      <main className="flex-1 overflow-y-auto p-4 sm:p-10 pb-20 sm:pb-10 dk-page">
        {activeTab === 'dashboard' ? (
          <div>
            {/* Header */}
            <Header
              title="首页"
              subtitle="微信聊天数据分析总览"
            />

            {/* 当前时间范围标签 */}
            <div className="mb-6 flex items-center gap-2">
              <span className="text-xs font-bold text-[#07c160] bg-[#07c16015] px-3 py-1.5 rounded-full">
                当前分析范围：{timeRange.label}
              </span>
              <span className="text-xs font-bold text-[#1d1d1f] bg-white px-3 py-1.5 rounded-full border border-gray-200">
                引擎：{mergedRuntimeStatus.engine_type || 'unknown'}
              </span>
              <span className="text-xs font-bold text-[#1d1d1f] bg-white px-3 py-1.5 rounded-full border border-gray-200">
                解密：{mergedRuntimeStatus.decrypt_state || 'unknown'}
              </span>
              <span className="text-xs font-bold text-[#1d1d1f] bg-white px-3 py-1.5 rounded-full border border-gray-200">
                Revision：{String(mergedRuntimeStatus.data_revision ?? '-')}
              </span>
              <button
                onClick={handleReselect}
                className="text-xs text-gray-400 hover:text-gray-600 underline"
              >
                重新选择
              </button>
            </div>

            {/* KPI Cards */}
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-6 mb-6 sm:mb-8">
              <KPICard
                title="总好友数"
                value={globalStats?.total_friends || 0}
                subtitle="Total Friends"
                icon={Users}
                color="green"
              />
              <KPICard
                title="总消息量"
                value={formatCompactNumber(globalStats?.total_messages || 0)}
                subtitle="Total Messages"
                icon={MessageSquare}
                color="blue"
              />
              <KPICard
                title="活跃好友"
                value={healthStatus.hot}
                subtitle="7 天内有消息"
                icon={Flame}
                color="orange"
              />
              <KPICard
                title="零消息"
                value={healthStatus.cold}
                subtitle="从未聊天"
                icon={Snowflake}
                color="purple"
              />
            </div>

            {/* Relationship Heatmap */}
            <div className="mb-6 sm:mb-8">
              <RelationshipHeatmap
                health={healthStatus}
                totalContacts={contacts.length}
              />
            </div>

            {/* Charts */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 sm:gap-8 mb-6 sm:mb-8">
              <MonthlyTrendChart data={globalStats} />
              <HourlyHeatmap data={globalStats} />
            </div>

            {/* Late Night Ranking */}
            <div className="mb-6 sm:mb-8">
              <LateNightRanking data={globalStats} />
            </div>

            <div className="mb-6 sm:mb-8">
              <div className="mb-3 flex items-center gap-2">
                <button
                  onClick={() => setDashboardRelationMode('objective')}
                  className={`px-4 py-2 rounded-full text-sm font-bold transition ${
                    dashboardRelationMode === 'objective'
                      ? 'bg-[#07c160] text-white shadow-lg shadow-green-100/50'
                      : 'bg-white text-gray-500 border border-gray-200 hover:border-[#07c16030]'
                  }`}
                >
                  客观模式
                </button>
                <button
                  onClick={() => setDashboardRelationMode('controversy')}
                  className={`px-4 py-2 rounded-full text-sm font-bold transition ${
                    dashboardRelationMode === 'controversy'
                      ? 'bg-[#1d1d1f] text-white shadow-lg shadow-black/10'
                      : 'bg-white text-gray-500 border border-gray-200 hover:border-[#1d1d1f30]'
                  }`}
                >
                  争议模式
                </button>
              </div>
              {dashboardRelationMode === 'objective' ? (
                <RelationOverviewSection
                  data={visibleRelationOverview ?? EMPTY_RELATION_GROUPED}
                  loading={relationOverviewLoading}
                  onItemClick={(_listType: RelationOverviewListType, item) => {
                    handleOpenRelationContact(item.username, 'analysis');
                  }}
                />
              ) : (
                <ControversyOverviewSection
                  data={visibleControversyOverview}
                  loading={controversyOverviewLoading}
                  onItemClick={(item, _board: ControversyBoardKey) => {
                    handleOpenRelationContact(item.username, 'analysis', item.label);
                  }}
                />
              )}
            </div>

            <div className="mb-6 sm:mb-8">
              <GlobalSearchPanel
                query={globalQuery}
                results={visibleGlobalResults}
                loading={globalSearchLoading}
                includeMine={globalIncludeMine}
                filterType={globalFilterType}
                onQueryChange={setGlobalQuery}
                onSearch={() => { void runGlobalSearch(globalQuery, globalIncludeMine); }}
                onIncludeMineChange={(value) => {
                  setGlobalIncludeMine(value);
                  if (globalSearchTouched && globalQuery.trim()) {
                    void runGlobalSearch(globalQuery, value);
                  }
                }}
                onFilterTypeChange={setGlobalFilterType}
                onOpenContact={handleOpenSearchContact}
                onOpenGroup={handleOpenSearchGroup}
                emptyText={globalSearchTouched ? '未找到相关消息' : '输入关键词后搜索全部聊天记录'}
              />
            </div>
            <div className="flex justify-end">
              <button
                type="button"
                onClick={() => {
                  refreshDerivedData();
                  if (globalSearchTouched && globalQuery.trim()) {
                    void runGlobalSearch(globalQuery, globalIncludeMine);
                  }
                }}
                className="inline-flex items-center gap-2 rounded-xl border border-gray-200 bg-white px-4 py-2 text-sm font-semibold text-gray-700 transition hover:border-[#07c16060] hover:text-[#07c160]"
              >
                手动刷新首页数据
              </button>
            </div>
          </div>
        ) : activeTab === 'contacts' ? (
          <ContactsPage
            contacts={filteredContacts}
            loading={statsLoading}
            total={filteredContacts.length}
            search={contactSearch}
            activityFilter={contactActivityFilter}
            categoryFilter={contactCategoryFilter}
            genderFilter={contactGenderFilter}
            sortKey={contactSort}
            onSearchChange={setContactSearch}
            onActivityFilterChange={setContactActivityFilter}
            onCategoryFilterChange={setContactCategoryFilter}
            onGenderFilterChange={setContactGenderFilter}
            onSortChange={setContactSort}
            onRefresh={() => {
              refreshContacts();
            }}
            onContactClick={handleContactClick}
          />
        ) : activeTab === 'sns' ? (
          <SnsSearchPage contacts={contacts} />
        ) : activeTab === 'groups' ? (
          <GroupsView allContacts={allContacts} onContactClick={handleContactClick} blockedGroups={blockedGroups} onBlockGroup={addBlockedGroup} />
        ) : activeTab === 'privacy' ? (
          <PrivacyView
            blockedUsers={blockedUsers}
            blockedGroups={blockedGroups}
            onAddBlockedUser={addBlockedUser}
            onRemoveBlockedUser={removeBlockedUser}
            onAddBlockedGroup={addBlockedGroup}
            onRemoveBlockedGroup={removeBlockedGroup}
            allContacts={allContacts}
            allGroups={allGroups}
          />
        ) : activeTab === 'system' ? (
          <SystemRuntimeView
            backendStatus={backendStatus}
            runtime={runtime}
            configCheck={configCheck}
            changes={runtimeChanges}
            tasks={runtimeTasks}
            logs={runtimeLogs}
            latestEvent={null}
            eventsConnected={eventsConnected}
            loading={runtimeLoading}
            error={runtimeError}
            actionNotice={systemActionNotice}
            meta={runtimeMeta}
            defaultContactUsername={selectedContact?.username}
            defaultGroupUsername={selectedGroup?.username}
            defaultSearchQuery={globalQuery}
            defaultSearchIncludeMine={globalIncludeMine}
            onRefresh={() => { void refreshRuntime(); }}
            onStartDecrypt={(options) => { void handleStartDecrypt(options); }}
            onStopDecrypt={() => { void handleStopDecrypt(); }}
            onReindex={() => { void handleReindex(); }}
            onExportContact={(username, limit) => { void handleExportContact(username, limit); }}
            onExportGroup={(username, date) => { void handleExportGroup(username, date); }}
            onExportSearch={(query, includeMine, limit) => { void handleExportSearch(query, includeMine, limit); }}
          />
        ) : (
          <div>
            <Header title="Database" subtitle="数据库管理" />
            <DatabaseView />
          </div>
        )}
      </main>

      {/* Contact Detail Modal */}
      <ContactModal
        contact={selectedContact}
        onClose={handleCloseModal}
        initialTab={selectedContactView}
        initialControversyLabel={selectedControversyLabel}
        onGroupClick={(g) => { setSelectedContact(null); setSelectedGroup(g); }}
        onBlock={(username) => { addBlockedUser(username); }}
      />

      {/* Group Detail Modal (triggered from contact modal) */}
      {selectedGroup && (
        <GroupDetailModal
          group={selectedGroup}
          onClose={() => setSelectedGroup(null)}
          allContacts={allContacts}
          onContactClick={(c) => { setSelectedGroup(null); setSelectedContact(c); }}
          onBlock={(username) => { addBlockedGroup(username); }}
        />
      )}
    </div>
  );
}

export default App;
