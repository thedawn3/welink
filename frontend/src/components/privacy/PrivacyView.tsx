/**
 * 隐私屏蔽设置页
 * 允许用户添加/删除联系人和群聊的屏蔽规则，屏蔽项不会出现在列表中
 */

import React, { useState } from 'react';
import { X, Plus, ShieldOff, User, Users } from 'lucide-react';
import type { ContactStats, GroupInfo } from '../../types';

interface PrivacyViewProps {
  blockedUsers: string[];
  blockedGroups: string[];
  onAddBlockedUser: (v: string) => void;
  onRemoveBlockedUser: (v: string) => void;
  onAddBlockedGroup: (v: string) => void;
  onRemoveBlockedGroup: (v: string) => void;
  allContacts?: ContactStats[];
  allGroups?: GroupInfo[];
}

const TagList: React.FC<{
  items: string[];
  onRemove: (v: string) => void;
  emptyText: string;
  labelFor?: (id: string) => string;
}> = ({ items, onRemove, emptyText, labelFor }) => (
  <div className="min-h-[56px] flex flex-wrap gap-2">
    {items.length === 0 ? (
      <span className="text-sm text-gray-400 self-center">{emptyText}</span>
    ) : (
      items.map((item) => {
        const label = labelFor ? labelFor(item) : item;
        const showId = label !== item;
        return (
          <span
            key={item}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium bg-gray-100 text-gray-700"
          >
            <span>{label}</span>
            {showId && <span className="text-xs text-gray-400">{item}</span>}
            <button
              onClick={() => onRemove(item)}
              className="text-gray-400 hover:text-red-500 transition-colors"
            >
              <X size={13} />
            </button>
          </span>
        );
      })
    )}
  </div>
);

const AddInput: React.FC<{
  placeholder: string;
  onAdd: (v: string) => void;
}> = ({ placeholder, onAdd }) => {
  const [value, setValue] = useState('');

  const submit = () => {
    if (value.trim()) {
      onAdd(value.trim());
      setValue('');
    }
  };

  return (
    <div className="flex flex-col sm:flex-row gap-2 mt-3">
      <input
        type="text"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && submit()}
        placeholder={placeholder}
        className="flex-1 px-4 py-2 text-sm border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#07c160]/20 focus:border-[#07c160] transition-all"
      />
      <button
        onClick={submit}
        className="flex items-center gap-1.5 px-4 py-2 bg-[#07c160] text-white text-sm font-semibold rounded-xl hover:bg-[#06ad56] transition-colors"
      >
        <Plus size={15} />
        添加
      </button>
    </div>
  );
};

export const PrivacyView: React.FC<PrivacyViewProps> = ({
  blockedUsers,
  blockedGroups,
  onAddBlockedUser,
  onRemoveBlockedUser,
  onAddBlockedGroup,
  onRemoveBlockedGroup,
  allContacts = [],
  allGroups = [],
}) => {
  const userLabelFor = (id: string): string => {
    const c = allContacts.find(
      (c) => c.username === id || c.nickname === id || c.remark === id
    );
    return c ? (c.remark || c.nickname || id) : id;
  };

  const groupLabelFor = (id: string): string => {
    const g = allGroups.find((g) => g.username === id || g.name === id);
    return g ? g.name : id;
  };

  return (
    <div className="max-w-none sm:max-w-2xl">
      <div className="mb-8">
        <h1 className="text-2xl font-black text-[#1d1d1f] mb-1">隐私屏蔽</h1>
        <p className="text-sm text-gray-400">被屏蔽的联系人和群聊将从所有列表中隐藏，数据仍保留在数据库中。</p>
      </div>

      <div className="bg-amber-50 border border-amber-200 rounded-2xl px-5 py-4 mb-6 flex gap-3">
        <ShieldOff size={18} className="text-amber-500 flex-shrink-0 mt-0.5" />
        <p className="text-sm text-amber-700">
          屏蔽规则仅存储在当前浏览器中，清除浏览器数据后将失效。支持按<strong>微信ID、昵称或备注名</strong>匹配，任一字段匹配即生效。
        </p>
      </div>

      {/* 屏蔽联系人 */}
      <div className="bg-white rounded-2xl border border-gray-100 p-6 mb-4">
        <div className="flex items-center gap-2 mb-4">
          <User size={18} className="text-[#07c160]" />
          <h2 className="font-bold text-[#1d1d1f]">屏蔽联系人</h2>
          {blockedUsers.length > 0 && (
            <span className="ml-auto text-xs font-bold px-2.5 py-1 rounded-full bg-gray-100 text-gray-500">
              {blockedUsers.length} 条
            </span>
          )}
        </div>
        <TagList
          items={blockedUsers}
          onRemove={onRemoveBlockedUser}
          emptyText="暂无屏蔽联系人"
          labelFor={userLabelFor}
        />
        <AddInput
          placeholder="输入微信ID、昵称或备注名，按回车添加"
          onAdd={onAddBlockedUser}
        />
      </div>

      {/* 屏蔽群聊 */}
      <div className="bg-white rounded-2xl border border-gray-100 p-6">
        <div className="flex items-center gap-2 mb-4">
          <Users size={18} className="text-[#07c160]" />
          <h2 className="font-bold text-[#1d1d1f]">屏蔽群聊</h2>
          {blockedGroups.length > 0 && (
            <span className="ml-auto text-xs font-bold px-2.5 py-1 rounded-full bg-gray-100 text-gray-500">
              {blockedGroups.length} 条
            </span>
          )}
        </div>
        <TagList
          items={blockedGroups}
          onRemove={onRemoveBlockedGroup}
          emptyText="暂无屏蔽群聊"
          labelFor={groupLabelFor}
        />
        <AddInput
          placeholder="输入群名称或群ID（以 @chatroom 结尾），按回车添加"
          onAdd={onAddBlockedGroup}
        />
      </div>
    </div>
  );
};
