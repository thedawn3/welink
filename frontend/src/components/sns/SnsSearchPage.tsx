import React, { useMemo, useState } from 'react';
import { RefreshCw, Search } from 'lucide-react';
import { Header } from '../layout/Header';
import { snsApi } from '../../services/api';
import type { ContactStats, SnsSearchItem, SnsSearchKind, SnsSearchResponse } from '../../types';

interface SnsSearchPageProps {
  contacts: ContactStats[];
}

const formatTime = (value: string): string => {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(
    date.getHours(),
  ).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
};

const getDisplayName = (contact: ContactStats) => contact.remark || contact.nickname || contact.username;

const normalizeResponse = (payload: SnsSearchResponse | SnsSearchItem[]): SnsSearchResponse => {
  if (Array.isArray(payload)) {
    return { items: payload, total: payload.length, has_sns_db: payload.length > 0 };
  }
  if (Array.isArray(payload.items)) return payload;
  return { ...payload, items: [] };
};

export const SnsSearchPage: React.FC<SnsSearchPageProps> = ({ contacts }) => {
  const [query, setQuery] = useState('');
  const [selectedUsername, setSelectedUsername] = useState('');
  const [kind, setKind] = useState<SnsSearchKind>('all');
  const [fromDate, setFromDate] = useState('');
  const [toDate, setToDate] = useState('');
  const [limit, setLimit] = useState(100);
  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState<SnsSearchItem[]>([]);
  const [meta, setMeta] = useState<{ unavailableReason?: string; total?: number; hasSnsDb?: boolean }>({});
  const [searched, setSearched] = useState(false);

  const contactOptions = useMemo(() => {
    const sorted = [...contacts];
    sorted.sort((a, b) => getDisplayName(a).localeCompare(getDisplayName(b), 'zh-Hans-CN'));
    return sorted;
  }, [contacts]);

  const runSearch = async () => {
    setLoading(true);
    try {
      const payload = await snsApi.search({
        q: query.trim() || undefined,
        username: selectedUsername || undefined,
        kind,
        from: fromDate || undefined,
        to: toDate || undefined,
        limit,
      });
      const normalized = normalizeResponse(payload);
      setItems(normalized.items ?? []);
      setMeta({
        unavailableReason: normalized.unavailable_reason,
        total: normalized.total,
        hasSnsDb: normalized.has_sns_db,
      });
      setSearched(true);
    } catch (error) {
      console.error('SNS search failed', error);
      setItems([]);
      setMeta({
        unavailableReason: error instanceof Error ? error.message : '朋友圈查询失败',
      });
      setSearched(true);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <Header title="朋友圈" subtitle="按发帖、互动、索引记录进行手动查询" />
      <div className="mb-6 rounded-3xl border border-gray-100 bg-white p-4 sm:p-6">
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-12">
          <div className="relative lg:col-span-4">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={16} />
            <input
              type="text"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="关键词（内容、昵称）"
              className="w-full rounded-xl border border-gray-200 bg-white py-2.5 pl-9 pr-3 text-sm font-medium placeholder:text-gray-400 focus:border-[#07c160] focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
            />
          </div>
          <div className="lg:col-span-3">
            <select
              value={selectedUsername}
              onChange={(event) => setSelectedUsername(event.target.value)}
              className="w-full rounded-xl border border-gray-200 bg-white px-3 py-2.5 text-sm font-medium text-gray-600 focus:border-[#07c160] focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
            >
              <option value="">全部联系人</option>
              {contactOptions.map((contact) => (
                <option key={contact.username} value={contact.username}>
                  {getDisplayName(contact)}
                </option>
              ))}
            </select>
          </div>
          <div className="lg:col-span-2">
            <select
              value={kind}
              onChange={(event) => setKind(event.target.value as SnsSearchKind)}
              className="w-full rounded-xl border border-gray-200 bg-white px-3 py-2.5 text-sm font-medium text-gray-600 focus:border-[#07c160] focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
            >
              <option value="all">全部类型</option>
              <option value="post">发帖</option>
              <option value="interaction">互动</option>
              <option value="index">索引记录</option>
            </select>
          </div>
          <div className="lg:col-span-3">
            <div className="grid grid-cols-2 gap-2">
              <input
                type="date"
                value={fromDate}
                onChange={(event) => setFromDate(event.target.value)}
                className="rounded-xl border border-gray-200 bg-white px-3 py-2.5 text-sm font-medium text-gray-600 focus:border-[#07c160] focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
              />
              <input
                type="date"
                value={toDate}
                onChange={(event) => setToDate(event.target.value)}
                className="rounded-xl border border-gray-200 bg-white px-3 py-2.5 text-sm font-medium text-gray-600 focus:border-[#07c160] focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
              />
            </div>
          </div>
        </div>
        <div className="mt-3 flex flex-wrap items-center gap-3">
          <label className="inline-flex items-center gap-2 text-sm font-medium text-gray-600">
            限制
            <input
              type="number"
              min={1}
              max={500}
              value={limit}
              onChange={(event) => setLimit(Number(event.target.value) || 100)}
              className="w-24 rounded-lg border border-gray-200 px-2 py-1.5 text-sm focus:border-[#07c160] focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
            />
          </label>
          <button
            type="button"
            onClick={() => {
              void runSearch();
            }}
            className="inline-flex items-center gap-2 rounded-xl bg-[#07c160] px-4 py-2 text-sm font-bold text-white transition hover:bg-[#06ad56]"
          >
            <Search size={14} />
            手动查询
          </button>
          <button
            type="button"
            onClick={() => {
              void runSearch();
            }}
            className="inline-flex items-center gap-2 rounded-xl border border-gray-200 bg-white px-4 py-2 text-sm font-semibold text-gray-700 transition hover:border-[#07c16060] hover:text-[#07c160]"
          >
            <RefreshCw size={14} />
            刷新结果
          </button>
          <span className="text-xs font-medium text-gray-400">
            {typeof meta.total === 'number' ? `共 ${meta.total} 条` : searched ? `共 ${items.length} 条` : '尚未查询'}
          </span>
        </div>
      </div>

      {meta.unavailableReason ? (
        <div className="mb-6 rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm font-semibold text-amber-800">
          朋友圈不可用：{meta.unavailableReason}
        </div>
      ) : null}

      {!loading && searched && items.length === 0 ? (
        <div className="rounded-3xl border border-dashed border-gray-200 bg-white p-16 text-center text-sm font-semibold text-gray-400">
          未找到匹配记录
          {meta.hasSnsDb === false ? '（未检测到 sns.db）' : ''}
        </div>
      ) : null}

      {loading ? (
        <div className="rounded-3xl border border-gray-100 bg-white p-16 text-center">
          <p className="animate-pulse text-base font-semibold text-gray-400">查询中...</p>
        </div>
      ) : null}

      {!loading && items.length > 0 ? (
        <div className="space-y-3">
          {items.map((item, index) => (
            <article key={`${item.kind}-${item.feed_id}-${item.username}-${index}`} className="rounded-2xl border border-gray-100 bg-white p-4">
              <div className="mb-2 flex flex-wrap items-center gap-2">
                <span
                  className={`inline-flex rounded-full px-2 py-0.5 text-xs font-bold ${
                    item.kind === 'post'
                      ? 'bg-[#e7f8f0] text-[#07c160]'
                      : item.kind === 'interaction'
                        ? 'bg-blue-50 text-blue-600'
                        : 'bg-amber-50 text-amber-700'
                  }`}
                >
                  {item.kind === 'post' ? '发帖' : item.kind === 'interaction' ? '互动' : '索引记录'}
                </span>
                <span className="text-xs font-medium text-gray-500">feed_id: {item.feed_id || '-'}</span>
                <span className="text-xs font-medium text-gray-500">{formatTime(item.created_at)}</span>
              </div>
              <div className="text-sm font-bold text-[#1d1d1f]">
                {item.display_name || item.username}
                {item.kind === 'interaction' && item.counterparty_name ? (
                  <span className="ml-2 text-xs font-semibold text-gray-500">→ {item.counterparty_name}</span>
                ) : null}
                {item.kind === 'index' ? (
                  <span className="ml-2 text-xs font-semibold text-amber-700">索引记录（可能无正文）</span>
                ) : null}
              </div>
              <p className="mt-2 whitespace-pre-wrap break-words text-sm leading-6 text-gray-700">{item.content_text || '(空内容)'}</p>
              {item.raw_content ? (
                <details className="mt-2 text-xs text-gray-500">
                  <summary className="cursor-pointer font-semibold">查看原始内容</summary>
                  <pre className="mt-2 max-h-56 overflow-auto rounded-xl bg-[#f8f9fb] p-3 text-[11px] leading-5 text-gray-600">
                    {item.raw_content}
                  </pre>
                </details>
              ) : null}
            </article>
          ))}
        </div>
      ) : null}
    </div>
  );
};
