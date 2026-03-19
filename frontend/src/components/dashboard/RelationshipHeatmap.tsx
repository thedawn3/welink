/**
 * 关系热度图组件
 */

import React from 'react';
import type { HealthStatus, ContactStats } from '../../types';

const MAX_AVATARS_MOBILE = 6;
const MAX_AVATARS_DESKTOP = 12;

interface RelationshipHeatmapProps {
  health: HealthStatus;
  totalContacts: number;
  hotContacts?: ContactStats[];
  onContactClick?: (contact: ContactStats) => void;
}

const Avatar: React.FC<{ contact: ContactStats; onClick?: () => void }> = ({ contact, onClick }) => {
  const name = contact.remark || contact.nickname || contact.username;
  const url = contact.small_head_url || contact.big_head_url;
  return (
    <button
      onClick={onClick}
      title={name}
      className="w-8 h-8 rounded-full ring-2 ring-white flex-shrink-0 overflow-hidden -ml-2 first:ml-0 hover:ring-[#07c160] hover:z-10 relative transition-all"
    >
      {url ? (
        <img src={url} alt={name} className="w-full h-full object-cover"
          onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }} />
      ) : (
        <div className="w-full h-full bg-gradient-to-br from-[#07c160] to-[#06ad56] flex items-center justify-center text-white text-xs font-black">
          {name.charAt(0)}
        </div>
      )}
    </button>
  );
};

export const RelationshipHeatmap: React.FC<RelationshipHeatmapProps> = ({
  health,
  totalContacts,
  hotContacts = [],
  onContactClick,
}) => {
  const getPercentage = (value: number) =>
    totalContacts > 0 ? ((value / totalContacts) * 100).toFixed(1) : '0.0';

  const maxAvatars = typeof window !== 'undefined' && window.innerWidth < 640 ? MAX_AVATARS_MOBILE : MAX_AVATARS_DESKTOP;
  const shown = hotContacts.slice(0, maxAvatars);
  const overflow = hotContacts.length - shown.length;

  const categories = [
    { label: '活跃', value: health.hot,     color: 'bg-[#07c160]', textColor: 'text-[#07c160]', description: '7 天内有消息',    showAvatars: true },
    { label: '温热', value: health.warm,    color: 'bg-[#9be94a]', textColor: 'text-[#7bc934]', description: '7–30 天未联系',   showAvatars: false },
    { label: '渐冷', value: health.cooling, color: 'bg-[#ff9500]', textColor: 'text-[#ff9500]', description: '30–180 天未联系', showAvatars: false },
    { label: '沉寂', value: health.silent,  color: 'bg-[#576b95]', textColor: 'text-[#576b95]', description: '超过 180 天',     showAvatars: false },
    { label: '零消息', value: health.cold,  color: 'bg-gray-300',  textColor: 'text-gray-400',  description: '从未聊天',        showAvatars: false },
  ];

  return (
    <div className="dk-card bg-white dk-border p-8 rounded-3xl border border-gray-100">
      <h3 className="dk-text text-xl font-black text-[#1d1d1f] mb-6">关系热度分布</h3>

      <div className="space-y-5">
        {categories.map((category) => (
          <div key={category.label}>
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-3">
                <div className={`w-3 h-3 rounded-full ${category.color}`} />
                <span className="font-bold text-sm text-gray-700">{category.label}</span>
                <span className="text-xs text-gray-400 font-medium">{category.description}</span>
              </div>
              <div className="flex items-center gap-3">
                {/* 活跃好友头像堆叠 */}
                {category.showAvatars && shown.length > 0 && (
                  <div className="flex items-center">
                    {shown.map((c) => (
                      <Avatar key={c.username} contact={c} onClick={() => onContactClick?.(c)} />
                    ))}
                    {overflow > 0 && (
                      <div className="w-8 h-8 rounded-full ring-2 ring-white -ml-2 bg-gray-100 flex items-center justify-center text-xs font-bold text-gray-500 flex-shrink-0">
                        +{overflow}
                      </div>
                    )}
                  </div>
                )}
                <div className="flex items-baseline gap-2">
                  <span className={`text-xl sm:text-2xl font-black ${category.textColor}`}>{category.value}</span>
                  <span className="text-xs font-semibold text-gray-400">{getPercentage(category.value)}%</span>
                </div>
              </div>
            </div>
            <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
              <div
                className={`h-full ${category.color} transition-all duration-500 ease-out`}
                style={{ width: `${getPercentage(category.value)}%` }}
              />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
