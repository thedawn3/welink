/**
 * 类型定义
 */

export interface Contact {
  username: string;
  nickname: string;
  remark: string;
  alias: string;
  flag: number;
  description: string;
  big_head_url: string;
  small_head_url: string;
  delete_flag?: number;
  is_deleted?: boolean;
  is_biz?: boolean;
  likely_marketing?: boolean;
  contact_kind?: string;
  is_likely_alt?: boolean;
}

export interface ContactStats extends Contact {
  total_messages: number;
  their_messages?: number;
  my_messages?: number;
  first_message_time: string;
  last_message_time: string;
  first_msg?: string;
  type_pct?: Record<string, number>;
  type_cnt?: Record<string, number>;
  monthly_trend?: Record<string, number>;
  hourly_heatmap?: number[];
  type_mix?: Record<string, number>;
  shared_groups_count?: number;
}

export interface LateNightEntry {
  name: string;
  late_night_count: number;
  total_messages: number;
  ratio: number;
}

export interface GlobalStats {
  total_friends: number;
  zero_msg_friends: number;
  total_messages: number;
  monthly_trend: Record<string, number>;
  hourly_heatmap: number[];
  type_distribution: Record<string, number>;
  late_night_ranking: LateNightEntry[];
}

export interface ContactDetail {
  hourly_dist: number[];      // [24]
  weekly_dist: number[];      // [7]
  daily_heatmap: Record<string, number>;
  late_night_count: number;
  money_count: number;
  initiation_count: number;
  total_sessions: number;
}

export interface WordCount {
  word: string;
  count: number;
}

export interface DBInfo {
  name: string;
  path: string;
  size: number;
  type: 'contact' | 'message';
}

export interface TableInfo {
  name: string;
  row_count: number;
}

export interface ColumnInfo {
  cid: number;
  name: string;
  type: string;
  not_null: boolean;
  default_value: string;
  primary_key: boolean;
}

export interface TableData {
  columns: string[];
  rows: (string | number | null)[][];
  total: number;
}

export interface BackendStatus {
  is_indexing: boolean;
  is_initialized: boolean;
  total_cached: number;
  engine_type?: string;
  decrypt_state?: string;
  data_revision?: string | number;
  last_decrypt_at?: string;
  last_reindex_at?: string;
  pending_changes?: number;
  last_error?: string;
}

export type TabType = 'dashboard' | 'db' | 'groups' | 'privacy' | 'system';

export interface DecryptStartOptions {
  platform?: string;
  source_data_dir?: string;
  analysis_data_dir?: string;
  work_dir?: string;
  command?: string;
  auto_refresh?: boolean;
  wal_enabled?: boolean;
}

export interface RuntimeStatus {
  deployment_target?: string;
  engine_type?: string;
  decrypt_state?: string;
  data_revision?: string | number;
  last_decrypt_at?: string;
  last_reindex_at?: string;
  last_message_at?: string;
  last_sns_at?: string;
  pending_changes?: number;
  last_error?: string;
  last_change_reason?: string;
  updated_at?: string;
  is_indexing?: boolean;
  is_initialized?: boolean;
  total_cached?: number;
}

export interface RuntimeTask {
  id?: string;
  type?: string;
  status?: string;
  progress?: number;
  started_at?: string;
  finished_at?: string;
  message?: string;
  detail?: string;
  error?: string;
  updated_at?: string;
  work_dir?: string;
  command_summary?: string;
}

export interface RuntimeLogEntry {
  id?: number;
  time?: string;
  level?: string;
  source?: string;
  message: string;
  stream?: string;
  task_id?: string;
  fields?: Record<string, string>;
}

export interface RuntimeSyncStatus {
  running?: boolean;
  watch_wal?: boolean;
  last_revision_seq?: number;
}

export interface DirectoryValidation {
  path?: string;
  exists?: boolean;
  writable?: boolean;
  standard_layout?: boolean;
  has_contact?: boolean;
  has_message?: boolean;
  has_sns?: boolean;
  issues?: string[];
  warnings?: string[];
}

export interface RuntimeConfigCheckFeature {
  supported?: boolean;
  enabled?: boolean;
  provider?: string;
  issues?: string[];
  warnings?: string[];
}

export interface RuntimeConfigCheckSns {
  detected?: boolean;
  ready?: boolean;
  db_path?: string;
  issues?: string[];
  warnings?: string[];
}

export interface RuntimeConfigCheck {
  deployment_target?: string;
  mode?: string;
  can_start_sync?: boolean;
  primary_issue?: string;
  blocking_reasons?: string[];
  analysis_dir?: DirectoryValidation;
  source_dir?: DirectoryValidation;
  work_dir?: DirectoryValidation;
  decrypt?: RuntimeConfigCheckFeature;
  sync?: RuntimeConfigCheckFeature;
  sns?: RuntimeConfigCheckSns;
  issues?: string[];
  warnings?: string[];
  suggested_actions?: string[];
}

export interface RuntimeChanges {
  data_revision: number;
  pending_changes: number;
  last_reindex_at?: string;
  last_change_reason?: string;
  last_error?: string;
  items: RuntimeLogEntry[];
  sync?: RuntimeSyncStatus;
}

export interface RuntimeEvent {
  type: string;
  id?: string;
  at?: string;
  revision?: string | number;
  message?: string;
  payload?: Record<string, unknown>;
}

export interface RuntimeMeta {
  lastEventAt?: string;
  lastRefreshAt?: string;
  pollingFallback?: boolean;
  lastRefreshReason?: string;
}

export type ChatLabExportScope = 'contact' | 'group' | 'search';

export interface ChatLabExportSummary {
  scope?: ChatLabExportScope;
  username?: string;
  query?: string;
  date?: string;
  include_mine?: boolean;
  limit?: number;
  conversation_name?: string;
  message_count?: number;
  member_count?: number;
}

export interface ChatLabExportResponse {
  file_name?: string;
  mime_type?: string;
  data?: unknown;
  summary?: ChatLabExportSummary;
  [key: string]: unknown;
}

export interface GroupInfo {
  username: string;
  name: string;
  small_head_url: string;
  total_messages: number;
  first_message_time?: string;
  last_message_time: string;
}

export interface MemberStat {
  speaker: string;
  count: number;
}

export interface GroupDetail {
  hourly_dist: number[];
  weekly_dist: number[];
  daily_heatmap: Record<string, number>;
  member_rank: MemberStat[];
  top_words: { word: string; count: number }[];
}

export interface HealthStatus {
  hot: number;   // 最近 7 天有消息
  warm: number;  // 有消息但超过 7 天
  cold: number;  // 零消息
}

export interface FilteredStats {
  contacts: ContactStats[];
  global_stats: GlobalStats;
}

export interface SentimentPoint {
  month: string;   // "2024-03"
  score: number;   // 0~1
  count: number;
}

export interface SentimentResult {
  monthly: SentimentPoint[];
  overall: number;   // 0~1
  positive: number;
  negative: number;
  neutral: number;
}

export interface ChatMessage {
  time: string;     // "14:23"
  content: string;
  is_mine: boolean;
  type: number;
  date?: string;    // "2024-03-15"，搜索结果中使用
  timestamp?: number;
  ts?: number;      // 兼容旧前端字段
}

export interface ContactHistoryQuery {
  limit?: number;
  before?: number;
}

export interface ContactHistoryMessage extends ChatMessage {
  date: string;
  id?: string;
}

export interface ContactHistoryPage {
  messages: ContactHistoryMessage[];
  has_more: boolean;
}

export type ContactHistoryRawResponse =
  | ContactHistoryMessage[]
  | {
      messages?: ContactHistoryMessage[];
      items?: ContactHistoryMessage[];
      list?: ContactHistoryMessage[];
      has_more?: boolean;
      hasMore?: boolean;
      total?: number;
    };

export interface GlobalSearchHit {
  username: string;
  name: string;
  is_group: boolean;
  time: string;
  date: string;
  content: string;
  is_mine: boolean;
  type: number;
}

export interface GroupChatMessage {
  time: string;
  speaker: string;
  content: string;
  is_mine: boolean;
  type: number;
}

export interface RelationOverviewItem {
  username: string;
  name: string;
  score: number;
  confidence?: number;
  why?: string;
  evidence_preview?: string[];
  stale_hint?: string;
  confidence_reason?: string;
}

export interface RelationOverview {
  warming: RelationOverviewItem[];
  cooling: RelationOverviewItem[];
  initiative: RelationOverviewItem[];
  fast_reply: RelationOverviewItem[];
}

export interface RelationEvidence {
  date: string;
  time: string;
  content: string;
  is_mine: boolean;
  reason: string;
}

export interface RelationEvidenceGroup {
  id?: string;
  title: string;
  subtitle?: string;
  items: RelationEvidence[];
}

export interface RelationMetricItem {
  key: string;
  label: string;
  value: string;
  sub_value?: string;
  trend?: 'up' | 'down' | 'flat' | string;
  hint?: string;
  raw_value?: number;
}

export interface RelationStageItem {
  id?: string;
  stage: string;
  start_date: string;
  end_date?: string;
  summary?: string;
  score?: number;
}

export interface ControversyMetric {
  key: string;
  label: string;
  value: number;
  display_value?: string;
}

export interface ControversialLabel {
  label: string;
  score: number;
  confidence: number;
  stale_hint?: string;
  confidence_reason?: string;
  why: string;
  metrics?: ControversyMetric[];
  evidence_groups?: RelationEvidence[];
}

export interface RelationProfileDetail {
  username: string;
  name: string;
  confidence?: number;
  stale_hint?: string;
  confidence_reason?: string;
  stage_timeline: RelationStageItem[];
  objective_summary: string;
  playful_summary: string;
  metrics: RelationMetricItem[];
  controversial_labels: ControversialLabel[];
  evidence_groups: RelationEvidenceGroup[];
}

export interface ControversyItem {
  username: string;
  name: string;
  label: string;
  score: number;
  confidence: number;
  why: string;
  evidence_preview: RelationEvidence[];
  stale_hint?: string;
  confidence_reason?: string;
}

export interface ControversyOverview {
  simp: ControversyItem[];
  ambiguity: ControversyItem[];
  faded: ControversyItem[];
  tool_person: ControversyItem[];
  cold_violence: ControversyItem[];
}

export interface ControversyDetail {
  username: string;
  name: string;
  confidence?: number;
  stale_hint?: string;
  confidence_reason?: string;
  controversial_labels: ControversialLabel[];
}

// null means "all time"
export interface TimeRange {
  from: number | null;  // Unix seconds
  to: number | null;    // Unix seconds
  label: string;        // e.g. "近1年" or "2024-03"
}
