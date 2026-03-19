import React from 'react';
import { AlertTriangle, Flame, ShieldAlert } from 'lucide-react';

export interface ControversyMetric {
  key: string;
  label: string;
  value: number | string;
  displayValue?: string;
}

export interface ControversyEvidence {
  date: string;
  time: string;
  content: string;
  is_mine: boolean;
  reason: string;
}

export interface ControversialLabel {
  label: string;
  score: number;
  confidence: number;
  stale_hint?: string;
  confidence_reason?: string;
  why: string;
  metrics?: ControversyMetric[];
  evidence_groups?: ControversyEvidence[];
}

export interface ControversyPanelProps {
  mode: 'controversy' | 'objective' | 'playful';
  labels: ControversialLabel[];
  selectedLabel?: string;
  analysisConfidence?: number;
  staleHint?: string;
  confidenceReason?: string;
  onSelectLabel?: (item: ControversialLabel) => void;
  onEvidenceClick?: (evidence: ControversyEvidence, label: ControversialLabel, index: number) => void;
  className?: string;
  emptyText?: string;
}

const FIXED_RISK_NOTICE =
  '风险提示：本页为争议模式，仅供娱乐解读，不构成真实关系结论。若长期未联系，标签更偏历史回看且置信度会下调。请结合客观指标与原始聊天记录二次确认。';

const labelAlias: Record<string, string> = {
  simp: '舔狗',
  ambiguity: '暧昧升温',
  faded: '已凉',
  tool_person: '工具人倾向',
  cold_violence: '冷暴力',
};

function asPercent(v: number): string {
  const n = Number.isFinite(v) ? Math.max(0, Math.min(100, Math.round(v))) : 0;
  return `${n}%`;
}

export const ControversyPanel: React.FC<ControversyPanelProps> = ({
  mode,
  labels,
  selectedLabel,
  analysisConfidence,
  staleHint,
  confidenceReason,
  onSelectLabel,
  onEvidenceClick,
  className,
  emptyText = '当前没有可展示的争议标签',
}) => {
  if (mode !== 'controversy') return null;

  const normalized = labels ?? [];
  const active =
    normalized.find((item) => item.label === selectedLabel) ??
    normalized[0] ??
    null;
  const normalizedAnalysisConfidence =
    typeof analysisConfidence === 'number' && Number.isFinite(analysisConfidence)
      ? Math.max(0, Math.min(100, Math.round(analysisConfidence)))
      : null;

  return (
    <section
      className={`rounded-3xl border border-rose-100 bg-gradient-to-br from-white via-rose-50/50 to-amber-50/70 p-4 sm:p-6 ${className ?? ''}`}
    >
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2 text-[#1d1d1f]">
          <ShieldAlert size={18} className="text-rose-500" />
          <h4 className="text-base sm:text-lg font-black tracking-tight">争议锐评模式</h4>
        </div>
        <span className="inline-flex items-center gap-1 rounded-full border border-rose-200 bg-white px-2.5 py-1 text-[11px] font-bold uppercase tracking-wider text-rose-500">
          <Flame size={12} />
          娱乐锐评
        </span>
      </div>

      {normalized.length === 0 || !active ? (
        <div className="rounded-2xl border border-dashed border-gray-200 bg-white/80 px-4 py-8 text-center text-sm font-medium text-gray-400">
          {emptyText}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-[240px_minmax(0,1fr)]">
          <aside className="space-y-2">
            {normalized.map((item) => {
              const isActive = item.label === active.label;
              const scoreText = asPercent(item.score);
              return (
                <button
                  type="button"
                  key={item.label}
                  onClick={() => onSelectLabel?.(item)}
                  className={`w-full rounded-2xl border px-3 py-3 text-left transition-all ${
                    isActive
                      ? 'border-rose-300 bg-white shadow-sm shadow-rose-100'
                      : 'border-transparent bg-white/70 hover:border-rose-200 hover:bg-white'
                  }`}
                >
                  <div className="mb-1 flex items-center justify-between gap-2">
                    <span className="text-sm font-black text-[#1d1d1f]">
                      {labelAlias[item.label] ?? item.label}
                    </span>
                    <span className="text-xs font-black text-rose-500">{scoreText}</span>
                  </div>
                  <div className="h-1.5 w-full overflow-hidden rounded-full bg-rose-100">
                    <div
                      className="h-full rounded-full bg-gradient-to-r from-rose-500 to-amber-400"
                      style={{ width: scoreText }}
                    />
                  </div>
                </button>
              );
            })}
          </aside>

          <div className="rounded-2xl border border-white/80 bg-white/90 p-4 sm:p-5">
            <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
              <div>
                <p className="text-xs font-bold uppercase tracking-wider text-gray-400">强标签</p>
                <h5 className="text-xl font-black tracking-tight text-[#1d1d1f]">
                  {labelAlias[active.label] ?? active.label}
                </h5>
              </div>
              <div className="flex items-center gap-2">
                <span className="rounded-full bg-rose-100 px-3 py-1 text-xs font-black text-rose-600">
                  强度 {asPercent(active.score)}
                </span>
                <span className="rounded-full bg-amber-100 px-3 py-1 text-xs font-black text-amber-700">
                  可信度 {asPercent(active.confidence)}
                </span>
              </div>
            </div>

            <div className="mb-4 rounded-xl border border-rose-100 bg-rose-50/60 p-3">
              <p className="text-xs font-bold uppercase tracking-wider text-rose-500">判定原因</p>
              <p className="mt-1 text-sm font-semibold leading-6 text-[#1d1d1f]">{active.why}</p>
            </div>

            {(normalizedAnalysisConfidence !== null || staleHint || confidenceReason || active.stale_hint || active.confidence_reason) && (
              <div className="mb-4 rounded-xl border border-amber-200 bg-amber-50/80 p-3">
                <p className="text-xs font-bold uppercase tracking-wider text-amber-600">可信度提示</p>
                {normalizedAnalysisConfidence !== null && (
                  <p className="mt-1 text-sm font-black text-amber-700">
                    全局可信度 {normalizedAnalysisConfidence}%
                  </p>
                )}
                {(active.stale_hint || staleHint) && (
                  <p className="mt-1 text-xs font-semibold leading-5 text-amber-800">{active.stale_hint ?? staleHint}</p>
                )}
                {(active.confidence_reason || confidenceReason) && (
                  <p className="mt-1 text-xs leading-5 text-amber-700/90">{active.confidence_reason ?? confidenceReason}</p>
                )}
              </div>
            )}

            {!!active.metrics?.length && (
              <div className="mb-4 grid grid-cols-1 gap-2 sm:grid-cols-2">
                {active.metrics.map((metric) => (
                  <div
                    key={metric.key}
                    className="rounded-xl border border-gray-100 bg-gray-50 px-3 py-2"
                  >
                    <p className="text-xs font-bold uppercase tracking-wider text-gray-400">
                      {metric.label}
                    </p>
                    <p className="mt-0.5 text-sm font-black text-gray-700">
                      {metric.displayValue ?? String(metric.value)}
                    </p>
                  </div>
                ))}
              </div>
            )}

            <div>
              <p className="mb-2 text-xs font-bold uppercase tracking-wider text-gray-400">证据消息</p>
              <div className="space-y-2">
                {(active.evidence_groups ?? []).slice(0, 5).map((evidence, index) => (
                  <button
                    type="button"
                    key={`${evidence.date}-${evidence.time}-${index}`}
                    onClick={() => onEvidenceClick?.(evidence, active, index)}
                    className="w-full rounded-xl border border-gray-100 bg-white px-3 py-2 text-left transition-colors hover:border-rose-200 hover:bg-rose-50/50"
                  >
                    <div className="mb-1 flex items-center justify-between gap-2 text-xs font-semibold">
                      <span className={evidence.is_mine ? 'text-emerald-600' : 'text-sky-600'}>
                        {evidence.is_mine ? '我发送' : '对方发送'}
                      </span>
                      <span className="text-gray-400">
                        {evidence.date} {evidence.time}
                      </span>
                    </div>
                    <p className="mb-1 line-clamp-2 text-sm leading-5 text-[#1d1d1f]">{evidence.content}</p>
                    <p className="text-xs font-medium text-rose-500">依据: {evidence.reason}</p>
                  </button>
                ))}
                {!(active.evidence_groups ?? []).length && (
                  <div className="rounded-xl border border-dashed border-gray-200 bg-gray-50 px-3 py-5 text-center text-xs font-medium text-gray-400">
                    暂无证据消息
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      <div className="mt-4 flex items-start gap-2 rounded-2xl border border-amber-200 bg-amber-50/90 px-3 py-2.5 text-xs text-amber-800">
        <AlertTriangle size={14} className="mt-0.5 flex-shrink-0" />
        <p className="font-semibold leading-5">{FIXED_RISK_NOTICE}</p>
      </div>
    </section>
  );
};
