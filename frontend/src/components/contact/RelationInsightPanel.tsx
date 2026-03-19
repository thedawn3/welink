import React from 'react';

export interface RelationStageItem {
  id?: string;
  stage: string;
  startDate: string;
  endDate?: string;
  summary?: string;
  score?: number;
}

export interface RelationMetricItem {
  key: string;
  label: string;
  value: string | number;
  subValue?: string;
  trend?: 'up' | 'down' | 'flat';
  hint?: string;
}

export interface RelationEvidenceItem {
  id?: string;
  date: string;
  time: string;
  content: string;
  isMine: boolean;
  reason: string;
}

export interface RelationEvidenceGroup {
  id?: string;
  title: string;
  subtitle?: string;
  items: RelationEvidenceItem[];
}

export interface RelationInsightPanelProps {
  stageTimeline: RelationStageItem[];
  objectiveSummary: string;
  playfulSummary: string;
  metrics: RelationMetricItem[];
  evidenceGroups: RelationEvidenceGroup[];
  confidence?: number;
  staleHint?: string;
  confidenceReason?: string;
  onEvidenceClick?: (item: RelationEvidenceItem, group: RelationEvidenceGroup) => void;
  className?: string;
  loading?: boolean;
  emptyText?: string;
}

const trendClass: Record<NonNullable<RelationMetricItem['trend']>, string> = {
  up: 'text-emerald-600 bg-emerald-50 border-emerald-200',
  down: 'text-rose-600 bg-rose-50 border-rose-200',
  flat: 'text-gray-500 bg-gray-50 border-gray-200',
};

const trendText: Record<NonNullable<RelationMetricItem['trend']>, string> = {
  up: 'Rising',
  down: 'Falling',
  flat: 'Flat',
};

function renderMetricValue(value: string | number): string {
  if (typeof value === 'number') {
    if (Number.isInteger(value)) return value.toString();
    return value.toFixed(2);
  }
  return value;
}

export const RelationInsightPanel: React.FC<RelationInsightPanelProps> = ({
  stageTimeline,
  objectiveSummary,
  playfulSummary,
  metrics,
  evidenceGroups,
  confidence,
  staleHint,
  confidenceReason,
  onEvidenceClick,
  className,
  loading = false,
  emptyText = 'No relation insight yet.',
}) => {
  const hasContent =
    stageTimeline.length > 0 ||
    metrics.length > 0 ||
    evidenceGroups.some((group) => group.items.length > 0) ||
    Boolean(objectiveSummary.trim()) ||
    Boolean(playfulSummary.trim());

  if (loading) {
    return (
      <div className={`space-y-4 sm:space-y-5 ${className ?? ''}`}>
        <div className="rounded-2xl border border-gray-100 bg-gray-50 p-5 sm:p-6 animate-pulse">
          <div className="h-4 w-32 bg-gray-200 rounded mb-4" />
          <div className="h-3 w-full bg-gray-200 rounded mb-2" />
          <div className="h-3 w-4/5 bg-gray-200 rounded" />
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 sm:gap-4">
          {[1, 2, 3, 4].map((idx) => (
            <div key={idx} className="rounded-2xl border border-gray-100 p-4 sm:p-5 animate-pulse">
              <div className="h-3 w-20 bg-gray-200 rounded mb-3" />
              <div className="h-6 w-28 bg-gray-200 rounded mb-2" />
              <div className="h-3 w-16 bg-gray-200 rounded" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (!hasContent) {
    return (
      <div className={`rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-4 py-8 text-center text-sm text-gray-500 ${className ?? ''}`}>
        {emptyText}
      </div>
    );
  }

  const normalizedConfidence =
    typeof confidence === 'number' && Number.isFinite(confidence)
      ? Math.max(0, Math.min(100, Math.round(confidence)))
      : null;

  return (
    <section className={`space-y-5 sm:space-y-6 ${className ?? ''}`}>
      {(normalizedConfidence !== null || staleHint || confidenceReason) && (
        <div className="rounded-3xl border border-emerald-100 bg-gradient-to-br from-[#f7fff9] via-white to-[#f4fbff] p-4 sm:p-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-xs font-bold uppercase tracking-wider text-emerald-600">可信度提示</p>
              <p className="mt-1 text-sm text-gray-600">
                客观模式按样本量和近期活跃度综合评估可信度。
              </p>
            </div>
            {normalizedConfidence !== null && (
              <span className="inline-flex items-center rounded-full border border-emerald-200 bg-white px-3 py-1 text-sm font-black text-emerald-600">
                confidence {normalizedConfidence}%
              </span>
            )}
          </div>
          {staleHint && (
            <div className="mt-3 rounded-2xl border border-amber-200 bg-amber-50 px-3 py-2 text-xs font-semibold text-amber-800">
              {staleHint}
            </div>
          )}
          {confidenceReason && (
            <p className="mt-2 text-xs font-medium leading-relaxed text-gray-500">
              {confidenceReason}
            </p>
          )}
        </div>
      )}

      <div className="rounded-3xl border border-gray-100 bg-white p-4 sm:p-6">
        <div className="flex items-center justify-between gap-3 mb-4">
          <h4 className="text-sm sm:text-base font-black text-[#1d1d1f] tracking-tight">Stage Timeline</h4>
          <span className="text-[11px] font-bold uppercase tracking-wider text-gray-400">
            {stageTimeline.length} stages
          </span>
        </div>
        {stageTimeline.length === 0 ? (
          <p className="text-sm text-gray-400">No timeline data.</p>
        ) : (
          <ol className="space-y-3 sm:space-y-4">
            {stageTimeline.map((item, index) => (
              <li key={item.id ?? `${item.stage}-${index}`} className="relative pl-7">
                <span className="absolute left-0 top-1.5 h-3 w-3 rounded-full bg-[#07c160]" />
                {index < stageTimeline.length - 1 && (
                  <span className="absolute left-[5px] top-4 bottom-[-14px] w-[2px] bg-gray-100" />
                )}
                <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-1">
                  <div>
                    <p className="text-sm sm:text-[15px] font-bold text-[#1d1d1f]">{item.stage}</p>
                    <p className="text-xs text-gray-400 mt-0.5">
                      {item.startDate}
                      {item.endDate ? ` - ${item.endDate}` : ' - now'}
                    </p>
                  </div>
                  {typeof item.score === 'number' && (
                    <span className="inline-flex items-center rounded-full border border-[#07c16030] bg-[#07c16010] px-2.5 py-0.5 text-xs font-semibold text-[#07c160]">
                      {Math.round(item.score)}/100
                    </span>
                  )}
                </div>
                {item.summary && <p className="text-xs sm:text-sm text-gray-600 mt-1.5 leading-relaxed">{item.summary}</p>}
              </li>
            ))}
          </ol>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 sm:gap-5">
        <article className="rounded-3xl border border-gray-100 bg-white p-4 sm:p-6">
          <h4 className="text-sm sm:text-base font-black text-[#1d1d1f] tracking-tight mb-2">Objective Summary</h4>
          <p className="text-sm text-gray-600 leading-relaxed whitespace-pre-wrap">
            {objectiveSummary.trim() || 'No objective summary.'}
          </p>
        </article>
        <article className="rounded-3xl border border-gray-100 bg-gradient-to-br from-[#f6fffa] to-[#f8fbff] p-4 sm:p-6">
          <h4 className="text-sm sm:text-base font-black text-[#1d1d1f] tracking-tight mb-2">Playful Summary</h4>
          <p className="text-sm text-gray-600 leading-relaxed whitespace-pre-wrap">
            {playfulSummary.trim() || 'No playful summary.'}
          </p>
        </article>
      </div>

      <div className="rounded-3xl border border-gray-100 bg-white p-4 sm:p-6">
        <div className="flex items-center justify-between gap-3 mb-4">
          <h4 className="text-sm sm:text-base font-black text-[#1d1d1f] tracking-tight">Core Metrics</h4>
          <span className="text-[11px] font-bold uppercase tracking-wider text-gray-400">
            {metrics.length} items
          </span>
        </div>
        {metrics.length === 0 ? (
          <p className="text-sm text-gray-400">No metric data.</p>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3 sm:gap-4">
            {metrics.map((metric) => (
              <div key={metric.key} className="rounded-2xl border border-gray-100 bg-gray-50/70 px-4 py-3.5">
                <p className="text-xs font-semibold uppercase tracking-wider text-gray-400">{metric.label}</p>
                <p className="text-xl sm:text-2xl font-black text-[#1d1d1f] mt-1.5">{renderMetricValue(metric.value)}</p>
                <div className="mt-2 flex flex-wrap items-center gap-2">
                  {metric.subValue && <span className="text-xs font-medium text-gray-500">{metric.subValue}</span>}
                  {metric.trend && (
                    <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-bold ${trendClass[metric.trend]}`}>
                      {trendText[metric.trend]}
                    </span>
                  )}
                </div>
                {metric.hint && <p className="text-xs text-gray-400 mt-2 leading-relaxed">{metric.hint}</p>}
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="rounded-3xl border border-gray-100 bg-white p-4 sm:p-6">
        <div className="flex items-center justify-between gap-3 mb-4">
          <h4 className="text-sm sm:text-base font-black text-[#1d1d1f] tracking-tight">Evidence Groups</h4>
          <span className="text-[11px] font-bold uppercase tracking-wider text-gray-400">
            {evidenceGroups.length} groups
          </span>
        </div>
        {evidenceGroups.length === 0 ? (
          <p className="text-sm text-gray-400">No evidence data.</p>
        ) : (
          <div className="space-y-4">
            {evidenceGroups.map((group, groupIndex) => (
              <section key={group.id ?? `${group.title}-${groupIndex}`} className="rounded-2xl border border-gray-100 bg-gray-50/60 p-3.5 sm:p-4">
                <div className="mb-2.5">
                  <h5 className="text-sm font-bold text-[#1d1d1f]">{group.title}</h5>
                  {group.subtitle && <p className="text-xs text-gray-400 mt-0.5">{group.subtitle}</p>}
                </div>
                {group.items.length === 0 ? (
                  <p className="text-xs text-gray-400">No evidence in this group.</p>
                ) : (
                  <ul className="space-y-2.5">
                    {group.items.map((item, itemIndex) => {
                      const clickable = Boolean(onEvidenceClick);
                      return (
                        <li key={item.id ?? `${item.date}-${item.time}-${itemIndex}`}>
                          <button
                            type="button"
                            disabled={!clickable}
                            onClick={() => onEvidenceClick?.(item, group)}
                            className={`w-full text-left rounded-xl border border-gray-100 bg-white px-3 py-2.5 transition ${
                              clickable ? 'hover:border-[#07c16050] hover:bg-[#f8fffb] cursor-pointer' : 'cursor-default'
                            }`}
                          >
                            <div className="flex items-center justify-between gap-3 mb-1">
                              <p className="text-[11px] font-semibold uppercase tracking-wide text-gray-400">
                                {item.date} {item.time} · {item.isMine ? 'Me' : 'Them'}
                              </p>
                              {clickable && <span className="text-[11px] font-semibold text-[#07c160]">Jump</span>}
                            </div>
                            <p className="text-sm text-gray-700 leading-relaxed line-clamp-2 whitespace-pre-wrap">{item.content || '[Empty]'}</p>
                            <p className="mt-1.5 text-xs text-gray-500">{item.reason}</p>
                          </button>
                        </li>
                      );
                    })}
                  </ul>
                )}
              </section>
            ))}
          </div>
        )}
      </div>
    </section>
  );
};
