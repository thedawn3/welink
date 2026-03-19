import React from 'react';
import { Loader2, MessageCircle, Search, Users } from 'lucide-react';

export type GlobalSearchFilterType = 'all' | 'direct' | 'group';

export interface GlobalSearchResultItem {
  username: string;
  name: string;
  is_group: boolean;
  time: string;
  date: string;
  content: string;
  is_mine: boolean;
  type: number;
}

interface GlobalSearchPanelProps {
  query: string;
  results: GlobalSearchResultItem[];
  loading: boolean;
  includeMine: boolean;
  filterType: GlobalSearchFilterType;
  onQueryChange: (value: string) => void;
  onSearch: () => void;
  onIncludeMineChange: (value: boolean) => void;
  onFilterTypeChange: (value: GlobalSearchFilterType) => void;
  onOpenContact?: (username: string) => void;
  onOpenGroup?: (username: string) => void;
  emptyText?: string;
}

const FILTER_OPTIONS: Array<{ value: GlobalSearchFilterType; label: string }> = [
  { value: 'all', label: '全部' },
  { value: 'direct', label: '私聊' },
  { value: 'group', label: '群聊' },
];

export const GlobalSearchPanel: React.FC<GlobalSearchPanelProps> = ({
  query,
  results,
  loading,
  includeMine,
  filterType,
  onQueryChange,
  onSearch,
  onIncludeMineChange,
  onFilterTypeChange,
  onOpenContact,
  onOpenGroup,
  emptyText = '暂无匹配消息',
}) => {
  const normalizedQuery = query.trim();
  const canSearch = normalizedQuery.length > 0 && !loading;

  const visibleResults = results.filter((item) => {
    if (filterType === 'direct') return !item.is_group;
    if (filterType === 'group') return item.is_group;
    return true;
  });

  const totalCount = results.length;
  const visibleCount = visibleResults.length;

  const renderResultItem = (item: GlobalSearchResultItem, idx: number) => {
    const openAction = item.is_group ? onOpenGroup : onOpenContact;
    const clickable = Boolean(openAction);

    return (
      <div key={`${item.username}-${item.date}-${item.time}-${idx}`} className="p-3 rounded-2xl bg-[#f8f9fb] border border-gray-100">
        <div className="flex items-center justify-between gap-2 mb-2">
          <button
            type="button"
            onClick={() => clickable && openAction?.(item.username)}
            disabled={!clickable}
            className={`text-left text-xs font-bold px-2.5 py-1 rounded-full ${
              item.is_group
                ? 'bg-blue-50 text-blue-600'
                : 'bg-[#07c16015] text-[#07c160]'
            } ${clickable ? 'hover:opacity-80 transition-opacity' : 'cursor-default'}`}
          >
            {item.is_group ? `群聊 · ${item.name}` : `私聊 · ${item.name}`}
          </button>
          <span className="text-[11px] text-gray-400">
            {item.date} {item.time}
          </span>
        </div>

        <div className="flex items-start gap-2">
          <span
            className={`mt-0.5 inline-flex items-center justify-center w-5 h-5 rounded-full text-[10px] font-black ${
              item.is_mine ? 'bg-[#07c160] text-white' : 'bg-[#576b95] text-white'
            }`}
          >
            {item.is_mine ? '我' : 'TA'}
          </span>
          <p className="text-sm leading-relaxed text-[#1d1d1f] break-words whitespace-pre-wrap">
            {item.content}
          </p>
        </div>
      </div>
    );
  };

  return (
    <section className="bg-white rounded-3xl border border-gray-100 p-4 sm:p-6">
      <div className="flex flex-col gap-3 mb-4">
        <form
          className="flex gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            if (canSearch) onSearch();
          }}
        >
          <div className="relative flex-1">
            <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-300" strokeWidth={2.5} />
            <input
              type="text"
              value={query}
              onChange={(e) => onQueryChange(e.target.value)}
              placeholder="搜索全部聊天记录..."
              className="w-full pl-9 pr-4 py-2.5 rounded-2xl border border-gray-200 text-sm focus:outline-none focus:border-[#07c160] transition-colors bg-gray-50"
            />
          </div>
          <button
            type="submit"
            disabled={!canSearch}
            className="px-5 py-2.5 bg-[#07c160] text-white rounded-2xl text-sm font-bold disabled:opacity-40 hover:bg-[#06ad56] transition-colors"
          >
            搜索
          </button>
        </form>

        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="inline-flex bg-[#f5f6f8] rounded-xl p-1">
            {FILTER_OPTIONS.map((option) => (
              <button
                key={option.value}
                type="button"
                onClick={() => onFilterTypeChange(option.value)}
                className={`px-3 py-1.5 rounded-lg text-xs font-bold transition-colors ${
                  filterType === option.value
                    ? 'bg-white text-[#07c160] shadow-sm'
                    : 'text-gray-500 hover:text-gray-700'
                }`}
              >
                {option.label}
              </button>
            ))}
          </div>

          <button
            type="button"
            onClick={() => onIncludeMineChange(!includeMine)}
            className={`flex items-center gap-1.5 text-xs font-bold px-3 py-1.5 rounded-full border transition-all ${
              includeMine
                ? 'bg-[#07c160] text-white border-[#07c160]'
                : 'bg-white text-gray-400 border-gray-200 hover:border-[#07c160] hover:text-[#07c160]'
            }`}
          >
            <span className={`w-2 h-2 rounded-full ${includeMine ? 'bg-white' : 'bg-gray-300'}`} />
            {includeMine ? '双方消息' : '仅对方消息'}
          </button>
        </div>
      </div>

      {loading ? (
        <div className="h-44 flex items-center justify-center">
          <Loader2 size={28} className="text-[#07c160] animate-spin" />
        </div>
      ) : visibleResults.length === 0 ? (
        <div className="h-32 flex items-center justify-center text-sm text-gray-300 font-semibold">
          {emptyText}
        </div>
      ) : (
        <div>
          <div className="flex flex-wrap items-center gap-2 text-xs text-gray-400 mb-3">
            <span className="inline-flex items-center gap-1">
              <Search size={12} />
              总计 {totalCount} 条
            </span>
            <span className="inline-flex items-center gap-1">
              <MessageCircle size={12} />
              私聊 {results.filter((item) => !item.is_group).length} 条
            </span>
            <span className="inline-flex items-center gap-1">
              <Users size={12} />
              群聊 {results.filter((item) => item.is_group).length} 条
            </span>
            {filterType !== 'all' && <span>当前筛选后 {visibleCount} 条</span>}
          </div>
          <div className="space-y-2 max-h-[52vh] overflow-y-auto pr-1">
            {visibleResults.map(renderResultItem)}
          </div>
        </div>
      )}
    </section>
  );
};
