import React, { useState } from 'react';
import {
  AlertTriangle,
  Flame,
  HeartCrack,
  Loader2,
  Sparkles,
  Wrench,
} from 'lucide-react';

export type ControversyBoardKey = 'simp' | 'ambiguity' | 'faded' | 'tool_person' | 'cold_violence';
export type ControversyGroupKey = 'all' | 'male' | 'female';

export interface ControversyEvidencePreview {
  date: string;
  time: string;
  content: string;
  is_mine: boolean;
  reason: string;
}

export interface ControversyItem {
  username: string;
  name: string;
  label: string;
  score: number;
  confidence: number;
  why: string;
  evidence_preview: ControversyEvidencePreview[];
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

export interface ControversyOverviewGrouped {
  all: ControversyOverview;
  male: ControversyOverview;
  female: ControversyOverview;
}

interface ControversyOverviewSectionProps {
  data: ControversyOverviewGrouped | null;
  loading: boolean;
  onItemClick?: (item: ControversyItem, board: ControversyBoardKey) => void;
  maxItems?: number;
  evidencePreviewCount?: number;
  emptyText?: string;
  className?: string;
}

type BoardMeta = {
  key: ControversyBoardKey;
  title: string;
  hint: string;
  icon: React.ReactNode;
  badgeClass: string;
  accentClass: string;
  panelClass: string;
};

const BOARD_META: BoardMeta[] = [
  {
    key: 'simp',
    title: '舔狗榜',
    hint: '高主动 + 低回报',
    icon: <Flame size={16} />,
    badgeClass: 'bg-rose-100 text-rose-700',
    accentClass: 'text-rose-300',
    panelClass: 'from-[#2a1117] via-[#231217] to-[#1b1318]',
  },
  {
    key: 'ambiguity',
    title: '暧昧榜',
    hint: '升温趋势 + 深夜浓度',
    icon: <Sparkles size={16} />,
    badgeClass: 'bg-fuchsia-100 text-fuchsia-700',
    accentClass: 'text-fuchsia-300',
    panelClass: 'from-[#24112a] via-[#1f152b] to-[#17162a]',
  },
  {
    key: 'faded',
    title: '已凉榜',
    hint: '历史高频近期降温',
    icon: <HeartCrack size={16} />,
    badgeClass: 'bg-sky-100 text-sky-700',
    accentClass: 'text-sky-300',
    panelClass: 'from-[#101f2d] via-[#15202d] to-[#161c2a]',
  },
  {
    key: 'tool_person',
    title: '工具人榜',
    hint: '事务沟通占比偏高',
    icon: <Wrench size={16} />,
    badgeClass: 'bg-amber-100 text-amber-700',
    accentClass: 'text-amber-300',
    panelClass: 'from-[#291f12] via-[#2a1f15] to-[#241d16]',
  },
  {
    key: 'cold_violence',
    title: '冷暴力榜',
    hint: '持续低频 + 延迟拉长',
    icon: <AlertTriangle size={16} />,
    badgeClass: 'bg-violet-100 text-violet-700',
    accentClass: 'text-violet-300',
    panelClass: 'from-[#181327] via-[#1b1629] to-[#1a1928]',
  },
];

const clampPercent = (value: number): number => {
  if (Number.isNaN(value)) return 0;
  return Math.max(0, Math.min(100, Math.round(value)));
};

const confidenceBadgeClass = (value: number): string => {
  if (value >= 75) return 'bg-emerald-500/25 text-emerald-200';
  if (value >= 50) return 'bg-amber-500/25 text-amber-100';
  return 'bg-rose-500/25 text-rose-100';
};

const buildConfidenceHint = (item: ControversyItem, confidence: number): string => {
  const staleHint = item.stale_hint?.trim();
  if (staleHint) return staleHint;
  const reason = item.confidence_reason?.trim();
  if (reason) return reason;
  if (confidence < 45) return '样本偏少或联系久远，当前判断偏历史回看。';
  if (confidence < 65) return '样本有限，建议回到客观模式结合证据复核。';
  return '';
};

const isHistoricalView = (item: ControversyItem, hint: string): boolean => {
  const source = `${item.stale_hint ?? ''} ${item.confidence_reason ?? ''} ${hint}`;
  return /久未联系|历史|回看|断联|不活跃/.test(source);
};

const formatEvidenceTime = (evidence: ControversyEvidencePreview): string => {
  if (evidence.date && evidence.time) return `${evidence.date} ${evidence.time}`;
  if (evidence.date) return evidence.date;
  if (evidence.time) return evidence.time;
  return '-';
};

const renderLoadingCard = (key: string) => (
  <div
    key={key}
    className="rounded-3xl border border-white/10 bg-gradient-to-br from-[#1e222a] to-[#15181f] p-4 sm:p-5 animate-pulse"
  >
    <div className="h-4 w-28 bg-white/10 rounded mb-3" />
    <div className="h-3 w-40 bg-white/10 rounded mb-5" />
    <div className="space-y-2">
      <div className="h-16 rounded-2xl bg-white/5" />
      <div className="h-16 rounded-2xl bg-white/5" />
    </div>
  </div>
);

export const ControversyOverviewSection: React.FC<ControversyOverviewSectionProps> = ({
  data,
  loading,
  onItemClick,
  maxItems = 5,
  evidencePreviewCount = 2,
  emptyText = '暂无争议榜单数据，先建立索引后再试',
  className = '',
}) => {
  const [activeGroup, setActiveGroup] = useState<ControversyGroupKey>('all');
  const [activeBoard, setActiveBoard] = useState<ControversyBoardKey>('simp');
  const activeGroupData = data?.[activeGroup] ?? data?.all ?? null;
  const boardMeta = BOARD_META.find((board) => board.key === activeBoard) ?? BOARD_META[0];
  const boardItems = activeGroupData?.[activeBoard] ?? [];
  const hasAnyData = Boolean(
    activeGroupData &&
      (activeGroupData.simp.length ||
        activeGroupData.ambiguity.length ||
        activeGroupData.faded.length ||
        activeGroupData.tool_person.length ||
        activeGroupData.cold_violence.length)
  );

  return (
    <section
      className={`rounded-3xl bg-[#0f1217] border border-white/10 p-4 sm:p-6 shadow-[0_10px_40px_rgba(0,0,0,0.18)] ${className}`}
    >
      <div className="flex flex-wrap items-start justify-between gap-3 mb-4">
        <div>
          <h3 className="text-lg sm:text-xl font-black text-white">关系分析 · 争议模式</h3>
          <p className="text-xs sm:text-sm text-gray-400 mt-1">
            娱乐化锐评，与客观模式共用同一指标底座；低样本或久未联系会下调置信度
          </p>
        </div>
        <span className="inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-bold bg-[#ff3b3018] text-[#ff8f87] border border-[#ff3b3040]">
          <AlertTriangle size={12} />
          争议模式
        </span>
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <span className="text-xs font-semibold text-gray-300">榜单分组</span>
        {([
          ['all', '全部'],
          ['male', '男'],
          ['female', '女'],
        ] as [ControversyGroupKey, string][]).map(([key, label]) => (
          <button
            key={key}
            type="button"
            onClick={() => setActiveGroup(key)}
            className={`rounded-full px-3 py-1 text-xs font-semibold transition ${
              activeGroup === key ? 'bg-white text-[#1d1d1f]' : 'bg-white/10 text-white/70 hover:bg-white/20'
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      <div className="mb-4 flex gap-2 flex-wrap">
        {BOARD_META.map((board) => (
          <button
            key={board.key}
            type="button"
            onClick={() => setActiveBoard(board.key)}
            className={`px-3 py-1.5 rounded-full text-xs font-semibold transition ${
              activeBoard === board.key
                ? 'bg-gradient-to-r from-[#ff6b6b] to-[#ffe66d] text-white shadow-sm'
                : 'bg-white/10 text-white/70 hover:bg-white/20'
            }`}
          >
            {board.title}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="space-y-4">
          <div className="h-16 rounded-2xl border border-white/10 bg-[#151a22] flex items-center justify-center gap-2 text-sm text-gray-300">
            <Loader2 size={16} className="animate-spin" />
            正在计算争议标签与证据...
          </div>
          <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
            {BOARD_META.map((board) => renderLoadingCard(board.key))}
          </div>
        </div>
      ) : !hasAnyData ? (
        <div className="h-40 rounded-2xl border border-dashed border-white/15 bg-[#151920] flex items-center justify-center text-sm text-gray-400 font-semibold px-4 text-center">
          {emptyText}
        </div>
      ) : (
        <div
          className={`rounded-3xl border border-white/10 bg-gradient-to-br ${boardMeta.panelClass} p-4 sm:p-5`}
        >
          <div className="flex items-start justify-between gap-3 mb-4">
            <div>
              <div className="flex items-center gap-2">
                <span
                  className={`inline-flex items-center gap-1 px-2 py-1 rounded-lg text-[11px] font-black ${boardMeta.badgeClass}`}
                >
                  {boardMeta.icon}
                  {boardMeta.title}
                </span>
              </div>
              <p className={`text-xs mt-1.5 ${boardMeta.accentClass}`}>{boardMeta.hint}</p>
            </div>
            <span className="text-[11px] text-white/70">Top {Math.max(1, boardItems.length)}</span>
          </div>

          {boardItems.length === 0 ? (
            <div className="h-24 rounded-2xl border border-white/10 bg-white/5 flex items-center justify-center text-xs text-gray-300">
              当前无上榜联系人
            </div>
          ) : (
            <div className="space-y-3">
              {boardItems.slice(0, maxItems).map((item, idx) => {
                const score = clampPercent(item.score);
                const confidence = clampPercent(item.confidence);
                const confidenceHint = buildConfidenceHint(item, confidence);
                const historicalView = isHistoricalView(item, confidenceHint);
                const evidenceRows = (item.evidence_preview || []).slice(0, evidencePreviewCount);
                const clickable = Boolean(onItemClick);

                return (
                  <button
                    key={`${boardMeta.key}-${item.username}-${idx}`}
                    type="button"
                    onClick={() => clickable && onItemClick?.(item, boardMeta.key)}
                    disabled={!clickable}
                    className={`w-full text-left rounded-2xl border p-3 transition-all ${
                      clickable
                        ? 'border-white/15 bg-black/20 hover:bg-black/35 hover:border-white/30'
                        : 'border-white/10 bg-black/20'
                    }`}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <div className="text-[11px] font-black text-gray-300">#{idx + 1}</div>
                        <div className="text-sm font-bold text-white truncate">{item.name || item.username}</div>
                      </div>
                      <div className="text-right flex-shrink-0">
                        <div className="text-xs text-gray-300">分数</div>
                        <div className="text-sm font-black text-white">{score}</div>
                      </div>
                    </div>

                    <div className="mt-2">
                      <div className="h-1.5 bg-white/10 rounded-full overflow-hidden">
                        <div
                          className="h-full bg-gradient-to-r from-[#ff6b6b] via-[#ffb36b] to-[#ffe66d] rounded-full"
                          style={{ width: `${score}%` }}
                        />
                      </div>
                    </div>

                    <div className="mt-2 flex items-center gap-2 text-[11px]">
                      <span className={`px-2 py-0.5 rounded-full font-bold ${confidenceBadgeClass(confidence)}`}>
                        置信度 {confidence}
                      </span>
                      <span className="px-2 py-0.5 rounded-full bg-[#07c1601f] text-[#8ee3ba]">
                        {item.label}
                      </span>
                      {historicalView && (
                        <span className="px-2 py-0.5 rounded-full bg-slate-500/25 text-slate-200">
                          历史回看
                        </span>
                      )}
                    </div>

                    <p className="mt-2 text-xs text-gray-300 leading-relaxed line-clamp-2">{item.why}</p>
                    {confidenceHint && (
                      <p className="mt-1.5 text-[11px] text-gray-300/90 line-clamp-2">{confidenceHint}</p>
                    )}

                    {evidenceRows.length > 0 && (
                      <div className="mt-2.5 space-y-1.5">
                        {evidenceRows.map((evidence, evidenceIndex) => (
                          <div
                            key={`${item.username}-evidence-${evidenceIndex}`}
                            className="rounded-xl border border-white/10 bg-white/5 p-2"
                          >
                            <div className="flex items-center justify-between gap-2 mb-1">
                              <span
                                className={`text-[10px] font-bold ${
                                  evidence.is_mine ? 'text-[#7be0b0]' : 'text-[#9bb8ff]'
                                }`}
                              >
                                {evidence.is_mine ? '我' : 'TA'}
                              </span>
                              <span className="text-[10px] text-gray-400">{formatEvidenceTime(evidence)}</span>
                            </div>
                            <p className="text-[11px] text-gray-200 leading-relaxed line-clamp-2">
                              {evidence.content}
                            </p>
                            {evidence.reason && (
                              <p className="mt-1 text-[10px] text-gray-400">依据: {evidence.reason}</p>
                            )}
                          </div>
                        ))}
                      </div>
                    )}
                  </button>
                );
              })}
            </div>
          )}
        </div>
      )}

      <div className="mt-4 flex items-start gap-2 rounded-2xl border border-amber-200 bg-amber-50/90 px-3 py-2.5 text-xs text-amber-800">
        <AlertTriangle size={14} className="mt-0.5 flex-shrink-0" />
        <p className="font-semibold leading-5">
          风险提示：本页为争议模式，仅供娱乐解读，不构成真实关系结论。请务必结合客观指标与原始聊天记录二次确认。
        </p>
      </div>
    </section>
  );
};
