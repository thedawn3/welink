import React, { useState } from 'react';
import { ArrowDownRight, ArrowUpRight, Hand, Loader2, Timer, TrendingUp, type LucideIcon } from 'lucide-react';

export type RelationOverviewListType = 'warming' | 'cooling' | 'initiative' | 'fast_reply';

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

export interface RelationOverviewData {
  warming: RelationOverviewItem[];
  cooling: RelationOverviewItem[];
  initiative: RelationOverviewItem[];
  fast_reply: RelationOverviewItem[];
}

interface RelationOverviewSectionProps {
  data: RelationOverviewData;
  loading?: boolean;
  emptyText?: string;
  className?: string;
  onItemClick?: (listType: RelationOverviewListType, item: RelationOverviewItem) => void;
}

interface ListMeta {
  key: RelationOverviewListType;
  title: string;
  subtitle: string;
  icon: LucideIcon;
  chipClass: string;
  scoreLabel: string;
}

const LIST_META: ListMeta[] = [
  {
    key: 'warming',
    title: '最近升温',
    subtitle: '近 7 天互动升幅最高',
    icon: ArrowUpRight,
    chipClass: 'bg-[#e7f8f0] text-[#07c160]',
    scoreLabel: '升温值',
  },
  {
    key: 'cooling',
    title: '最近降温',
    subtitle: '曾活跃但近期明显回落',
    icon: ArrowDownRight,
    chipClass: 'bg-orange-50 text-[#ff9500]',
    scoreLabel: '降温值',
  },
  {
    key: 'initiative',
    title: '我主动最多',
    subtitle: '新对话段由我发起占比高',
    icon: Hand,
    chipClass: 'bg-blue-50 text-[#10aeff]',
    scoreLabel: '主动值',
  },
  {
    key: 'fast_reply',
    title: '回复最快',
    subtitle: '中位响应更快的联系人',
    icon: Timer,
    chipClass: 'bg-purple-50 text-[#576b95]',
    scoreLabel: '响应值',
  },
];

const clampPercent = (value?: number): number | null => {
  if (typeof value !== 'number' || Number.isNaN(value)) return null;
  return Math.max(0, Math.min(100, Math.round(value)));
};

const getConfidenceMeta = (value: number | null): { label: string; chipClass: string } | null => {
  if (value == null) return null;
  if (value >= 75) {
    return { label: `高置信 ${value}`, chipClass: 'bg-emerald-50 text-emerald-700' };
  }
  if (value >= 50) {
    return { label: `中置信 ${value}`, chipClass: 'bg-amber-50 text-amber-700' };
  }
  return { label: `低置信 ${value}`, chipClass: 'bg-rose-50 text-rose-700' };
};

const buildInsightHint = (item: RelationOverviewItem, confidence: number | null): string => {
  const staleHint = item.stale_hint?.trim();
  if (staleHint) return staleHint;
  const reasonHint = item.confidence_reason?.trim();
  if (reasonHint) return reasonHint;
  if (confidence == null) return '';
  if (confidence < 45) return '样本偏少或联系较久远，结论更偏历史回看。';
  if (confidence < 65) return '样本有限，建议结合原始聊天记录一起判断。';
  return '';
};

const isHistoricalHint = (item: RelationOverviewItem, hint: string): boolean => {
  const source = `${item.stale_hint ?? ''} ${item.confidence_reason ?? ''} ${hint}`;
  return /久未联系|历史|回看|断联|不活跃/.test(source);
};

const ItemRow: React.FC<{
  item: RelationOverviewItem;
  index: number;
  listType: RelationOverviewListType;
  scoreLabel: string;
  chipClass: string;
  onItemClick?: (listType: RelationOverviewListType, item: RelationOverviewItem) => void;
}> = ({ item, index, listType, scoreLabel, chipClass, onItemClick }) => {
  const canClick = Boolean(onItemClick);
  const firstEvidence = item.evidence_preview?.find((value) => value.trim().length > 0) ?? item.why ?? '';
  const confidence = clampPercent(item.confidence);
  const confidenceMeta = getConfidenceMeta(confidence);
  const insightHint = buildInsightHint(item, confidence);
  const historical = isHistoricalHint(item, insightHint);

  return (
    <button
      type="button"
      onClick={() => onItemClick?.(listType, item)}
      disabled={!canClick}
      className={`w-full text-left rounded-2xl border border-gray-100 bg-[#f8f9fb] p-3.5 transition-all ${
        canClick ? 'hover:border-[#07c16033] hover:shadow-sm active:scale-[0.99]' : 'cursor-default'
      }`}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-white px-1.5 text-[11px] font-black text-gray-500">
              {index + 1}
            </span>
            <p className="truncate text-sm font-bold text-[#1d1d1f]">{item.name || item.username}</p>
          </div>
          <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
            {confidenceMeta && (
              <span className={`inline-flex rounded-full px-2 py-0.5 text-[10px] font-bold ${confidenceMeta.chipClass}`}>
                {confidenceMeta.label}
              </span>
            )}
            {historical && (
              <span className="inline-flex rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-bold text-slate-600">
                历史回看
              </span>
            )}
          </div>
          {firstEvidence ? (
            <p className="mt-1.5 line-clamp-2 text-xs leading-relaxed text-gray-500">{firstEvidence}</p>
          ) : (
            <p className="mt-1.5 text-xs text-gray-300">暂无说明</p>
          )}
          {insightHint && (
            <p className="mt-1.5 text-[11px] font-medium text-gray-400 line-clamp-2">{insightHint}</p>
          )}
        </div>
        <div className={`inline-flex flex-col items-end gap-1 rounded-xl px-2.5 py-1.5 ${chipClass}`}>
          <span className="text-[10px] font-bold opacity-80">{scoreLabel}</span>
          <span className="text-sm font-black leading-none">{Math.round(item.score)}</span>
        </div>
      </div>
    </button>
  );
};

export const RelationOverviewSection: React.FC<RelationOverviewSectionProps> = ({
  data,
  loading = false,
  emptyText = '暂无数据',
  className,
  onItemClick,
}) => {
  const [activeKey, setActiveKey] = useState<RelationOverviewListType>('warming');
  const activeMeta = LIST_META.find((meta) => meta.key === activeKey) ?? LIST_META[0];
  const activeItems = data[activeKey] ?? [];

  return (
    <section className={`bg-white rounded-3xl border border-gray-100 p-4 sm:p-6 ${className ?? ''}`}>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-lg font-black tracking-tight text-[#1d1d1f]">关系分析 · 客观模式</h2>
          <p className="mt-1 text-xs font-medium text-gray-400">基于消息频率、主动度与回复速度；低样本或久未联系会下调置信度</p>
        </div>
        <span className="inline-flex items-center gap-1 rounded-full bg-[#e7f8f0] px-2.5 py-1 text-[11px] font-bold text-[#07c160]">
          <TrendingUp size={12} />
          客观模式
        </span>
      </div>

      <div className="mb-4 flex gap-2 flex-wrap">
        {LIST_META.map((meta) => (
          <button
            key={meta.key}
            type="button"
            onClick={() => setActiveKey(meta.key)}
            className={`px-3 py-1.5 rounded-full text-xs font-semibold transition ${
              activeKey === meta.key
                ? 'bg-[#07c160] text-white shadow-lg shadow-green-100/50'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
            }`}
          >
            {meta.title}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="flex h-56 items-center justify-center">
          <Loader2 size={30} className="animate-spin text-[#07c160]" />
        </div>
      ) : (
        <div className="rounded-2xl border border-gray-100 p-4 sm:p-6">
          <div className="flex items-start justify-between gap-2 mb-3">
            <div>
              <h3 className="text-sm font-black text-[#1d1d1f]">{activeMeta.title}</h3>
              <p className="mt-0.5 text-[11px] text-gray-400">{activeMeta.subtitle}</p>
            </div>
            <span className={`inline-flex h-8 w-8 items-center justify-center rounded-xl ${activeMeta.chipClass}`}>
              <activeMeta.icon size={14} />
            </span>
          </div>

          {activeItems.length === 0 ? (
            <div className="flex h-28 items-center justify-center rounded-xl border border-dashed border-gray-200 bg-[#f8f9fb] px-3 text-center text-xs font-semibold text-gray-300">
              {emptyText}
            </div>
          ) : (
            <div className="space-y-2">
              {activeItems.map((item, index) => (
                <ItemRow
                  key={`${activeMeta.key}-${item.username}-${index}`}
                  item={item}
                  index={index}
                  listType={activeMeta.key}
                  scoreLabel={activeMeta.scoreLabel}
                  chipClass={activeMeta.chipClass}
                  onItemClick={onItemClick}
                />
              ))}
            </div>
          )}
        </div>
      )}
    </section>
  );
};
