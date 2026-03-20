import React from 'react';
import { RefreshCw, Search } from 'lucide-react';
import { Header } from '../layout/Header';
import { ContactTable } from '../dashboard/ContactTable';
import type { ContactStats, Gender } from '../../types';

export type ContactActivityFilter = 'all' | 'hot' | 'warm' | 'cold';
export type ContactCategoryFilter = 'all' | 'normal' | 'deleted';
export type ContactSortKey = 'messages_desc' | 'last_message_desc' | 'shared_groups_desc' | 'name_asc';
export type ContactGenderFilter = 'all' | Gender;

interface ContactsPageProps {
  contacts: ContactStats[];
  loading: boolean;
  total: number;
  search: string;
  activityFilter: ContactActivityFilter;
  categoryFilter: ContactCategoryFilter;
  genderFilter: ContactGenderFilter;
  sortKey: ContactSortKey;
  onSearchChange: (value: string) => void;
  onActivityFilterChange: (value: ContactActivityFilter) => void;
  onCategoryFilterChange: (value: ContactCategoryFilter) => void;
  onGenderFilterChange: (value: ContactGenderFilter) => void;
  onSortChange: (value: ContactSortKey) => void;
  onRefresh: () => void;
  onContactClick: (contact: ContactStats) => void;
}

export const ContactsPage: React.FC<ContactsPageProps> = ({
  contacts,
  loading,
  total,
  search,
  activityFilter,
  categoryFilter,
  genderFilter,
  sortKey,
  onSearchChange,
  onActivityFilterChange,
  onCategoryFilterChange,
  onGenderFilterChange,
  onSortChange,
  onRefresh,
  onContactClick,
}) => {
  return (
    <div>
      <Header title="联系人" subtitle="完整联系人列表与筛选面板" />
      <div className="mb-6 flex flex-col gap-4">
        <div className="flex items-center justify-between gap-4">
          <h2 className="dk-text text-2xl font-black text-[#1d1d1f]">
            联系人列表
            <span className="ml-3 text-lg font-semibold text-gray-400">{total} 位</span>
          </h2>
          <button
            type="button"
            onClick={onRefresh}
            className="inline-flex items-center gap-2 rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm font-semibold text-gray-700 transition hover:border-[#07c16060] hover:text-[#07c160]"
          >
            <RefreshCw size={14} />
            手动刷新
          </button>
        </div>
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={16} />
            <input
              type="text"
              placeholder="搜索联系人..."
              value={search}
              onChange={(event) => onSearchChange(event.target.value)}
              className="w-full rounded-xl border border-gray-200 bg-white py-2.5 pl-9 pr-4 text-sm font-medium placeholder:text-gray-400 focus:border-[#07c160] focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
            />
          </div>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:flex">
            <select
              value={activityFilter}
              onChange={(event) => onActivityFilterChange(event.target.value as ContactActivityFilter)}
              className="rounded-xl border border-gray-200 bg-white px-3 py-2.5 text-sm font-medium text-gray-600 focus:border-[#07c160] focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
            >
              <option value="all">全部状态</option>
              <option value="hot">活跃</option>
              <option value="warm">温热</option>
              <option value="cold">零消息</option>
            </select>
            <select
              value={categoryFilter}
              onChange={(event) => onCategoryFilterChange(event.target.value as ContactCategoryFilter)}
              className="rounded-xl border border-gray-200 bg-white px-3 py-2.5 text-sm font-medium text-gray-600 focus:border-[#07c160] focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
            >
              <option value="all">全部联系人</option>
              <option value="normal">普通联系人</option>
              <option value="deleted">已删好友</option>
            </select>
            <select
              value={genderFilter}
              onChange={(event) => onGenderFilterChange(event.target.value as ContactGenderFilter)}
              className="rounded-xl border border-gray-200 bg-white px-3 py-2.5 text-sm font-medium text-gray-600 focus:border-[#07c160] focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
            >
              <option value="all">全部性别</option>
              <option value="male">男</option>
              <option value="female">女</option>
              <option value="unknown">未知</option>
            </select>
            <select
              value={sortKey}
              onChange={(event) => onSortChange(event.target.value as ContactSortKey)}
              className="rounded-xl border border-gray-200 bg-white px-3 py-2.5 text-sm font-medium text-gray-600 focus:border-[#07c160] focus:outline-none focus:ring-2 focus:ring-[#07c160]/20"
            >
              <option value="messages_desc">按消息数</option>
              <option value="last_message_desc">按最近联系</option>
              <option value="shared_groups_desc">按共同群聊</option>
              <option value="name_asc">按名称</option>
            </select>
          </div>
        </div>
      </div>
      {loading && total === 0 ? (
        <div className="rounded-3xl border border-gray-100 bg-white p-20 text-center">
          <div className="animate-pulse text-lg font-bold text-gray-300">加载中...</div>
        </div>
      ) : (
        <ContactTable contacts={contacts} onContactClick={onContactClick} />
      )}
    </div>
  );
};
