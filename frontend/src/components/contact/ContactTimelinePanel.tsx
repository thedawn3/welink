import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Clock3, Loader2 } from 'lucide-react';
import type { ChatMessage, ContactHistoryMessage, ContactHistoryPage, ContactHistoryRawResponse } from '../../types';
import { contactsApi } from '../../services/api';

interface ContactTimelinePanelProps {
  username: string;
  contactName: string;
  focusDate?: string;
  focusKey?: number;
  className?: string;
  pageSize?: number;
}

function asNumber(value: unknown): number | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) return value;
  if (typeof value === 'string') {
    const num = Number(value);
    if (Number.isFinite(num)) return num;
  }
  return undefined;
}

function toUnix(date: string, time: string, fallback = 0): number {
  const full = `${date} ${time || '00:00'}`;
  const ts = new Date(full).getTime();
  if (!Number.isFinite(ts)) return fallback;
  return Math.floor(ts / 1000);
}

function toHistoryMessage(raw: Record<string, unknown>, idx: number): ContactHistoryMessage | null {
  const contentRaw = raw.content;
  const content = typeof contentRaw === 'string' ? contentRaw : '';
  if (!content.trim()) return null;

  const dateRaw = raw.date;
  const date = typeof dateRaw === 'string' && dateRaw.trim() ? dateRaw.trim() : '';
  const timeRaw = raw.time;
  const time = typeof timeRaw === 'string' && timeRaw.trim() ? timeRaw.trim() : '00:00';
  const ts = asNumber(raw.timestamp) ?? asNumber(raw.ts) ?? (date ? toUnix(date, time, idx) : idx);

  return {
    id: typeof raw.id === 'string' && raw.id.trim() ? raw.id : undefined,
    date: date || new Date(ts * 1000).toISOString().slice(0, 10),
    time,
    ts,
    content,
    is_mine: Boolean(raw.is_mine),
    type: asNumber(raw.type) ?? 1,
  };
}

function normalizeHistoryPage(raw: ContactHistoryRawResponse): ContactHistoryPage {
  if (Array.isArray(raw)) {
    const messages = raw
      .map((item, idx) => toHistoryMessage(item as unknown as Record<string, unknown>, idx))
      .filter((item): item is ContactHistoryMessage => Boolean(item));
    return {
      messages,
      has_more: messages.length > 0,
    };
  }

  const candidates = (raw.messages ?? raw.items ?? raw.list ?? []) as ContactHistoryMessage[];
  const messages = candidates
    .map((item, idx) => toHistoryMessage(item as unknown as Record<string, unknown>, idx))
    .filter((item): item is ContactHistoryMessage => Boolean(item));

  return {
    messages,
    has_more: typeof raw.has_more === 'boolean'
      ? raw.has_more
      : (typeof raw.hasMore === 'boolean' ? raw.hasMore : messages.length > 0),
  };
}

function messageKey(msg: ContactHistoryMessage): string {
  return [
    msg.id ?? '',
    msg.ts ?? '',
    msg.date,
    msg.time,
    msg.is_mine ? '1' : '0',
    msg.type,
    msg.content,
  ].join('|');
}

function mergeMessages(base: ContactHistoryMessage[], incoming: ContactHistoryMessage[]): ContactHistoryMessage[] {
  const map = new Map<string, ContactHistoryMessage>();
  for (const item of [...base, ...incoming]) {
    map.set(messageKey(item), item);
  }
  return Array.from(map.values()).sort((a, b) => (a.ts ?? 0) - (b.ts ?? 0));
}

function toDateLabel(date: string): string {
  const d = new Date(date);
  if (Number.isNaN(d.getTime())) return date;
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`;
}

export const ContactTimelinePanel: React.FC<ContactTimelinePanelProps> = ({
  username,
  contactName,
  focusDate,
  focusKey,
  className,
  pageSize = 80,
}) => {
  const [messages, setMessages] = useState<ContactHistoryMessage[]>([]);
  const [loadingInitial, setLoadingInitial] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadingFocus, setLoadingFocus] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(true);
  const [activeFocusDate, setActiveFocusDate] = useState<string | undefined>(undefined);
  const [jumpDate, setJumpDate] = useState('');

  const dayRefs = useRef<Record<string, HTMLElement | null>>({});
  const focusFetchedRef = useRef<Record<string, boolean>>({});

  const groupedMessages = useMemo(() => {
    const grouped = new Map<string, ContactHistoryMessage[]>();
    for (const msg of messages) {
      if (!grouped.has(msg.date)) grouped.set(msg.date, []);
      grouped.get(msg.date)!.push(msg);
    }
    return Array.from(grouped.entries()).map(([date, items]) => ({ date, items }));
  }, [messages]);

  const tryScrollToDate = (date: string): boolean => {
    const node = dayRefs.current[date];
    if (!node) return false;
    node.scrollIntoView({ behavior: 'smooth', block: 'start' });
    setActiveFocusDate(date);
    return true;
  };

  const loadLatest = async () => {
    setLoadingInitial(true);
    setError(null);
    try {
      const raw = await contactsApi.getMessageHistory(username, { limit: pageSize });
      const page = normalizeHistoryPage(raw);
      const sorted = page.messages.sort((a, b) => (a.ts ?? 0) - (b.ts ?? 0));
      setMessages(sorted);
      setHasMore(page.messages.length >= pageSize && Boolean(page.has_more));
    } catch (err) {
      console.error('Failed to fetch message history', err);
      setError('聊天记录暂时加载失败，请稍后再试');
      setMessages([]);
      setHasMore(false);
    } finally {
      setLoadingInitial(false);
    }
  };

  const loadOlder = async () => {
    if (loadingMore || !hasMore) return;
    setLoadingMore(true);
    setError(null);
    try {
      const oldest = messages[0];
      const beforeTs = oldest?.ts ?? oldest?.timestamp;
      const raw = await contactsApi.getMessageHistory(username, {
        limit: pageSize,
        before: beforeTs,
      });
      const page = normalizeHistoryPage(raw);
      const incoming = page.messages.sort((a, b) => (a.ts ?? 0) - (b.ts ?? 0));
      setMessages((current) => mergeMessages(current, incoming));
      if (!page.has_more || incoming.length === 0 || incoming.length < pageSize) {
        setHasMore(false);
      }
    } catch (err) {
      console.error('Failed to load older messages', err);
      setError('加载更早消息失败，请重试');
    } finally {
      setLoadingMore(false);
    }
  };

  useEffect(() => {
    setMessages([]);
    setHasMore(true);
    setActiveFocusDate(undefined);
    setJumpDate('');
    focusFetchedRef.current = {};
    void loadLatest();
  }, [username]);

  const focusOnDate = (targetDate: string) => {
    if (!targetDate) return;
    if (tryScrollToDate(targetDate)) {
      setJumpDate(targetDate);
      return;
    }
    if (focusFetchedRef.current[targetDate]) {
      setJumpDate(targetDate);
      return;
    }

    focusFetchedRef.current[targetDate] = true;
    setLoadingFocus(true);
    contactsApi.getDayMessages(username, targetDate)
      .then((dayMessages: ChatMessage[]) => {
        const mapped = (dayMessages ?? [])
          .map((item, idx) => toHistoryMessage({ ...(item as unknown as Record<string, unknown>), date: targetDate }, idx))
          .filter((item): item is ContactHistoryMessage => Boolean(item));
        if (mapped.length === 0) return;
        setMessages((current) => mergeMessages(current, mapped));
        setTimeout(() => {
          tryScrollToDate(targetDate);
        }, 80);
      })
      .catch((err) => {
        console.error('Failed to focus target day', err);
      })
      .finally(() => setLoadingFocus(false));
  };

  useEffect(() => {
    if (!focusDate) return;
    focusOnDate(focusDate);
  }, [focusDate, focusKey, username]);

  return (
    <section className={`space-y-4 ${className ?? ''}`}>
      <div className="rounded-2xl border border-gray-100 bg-[#f8f9fb] px-4 py-3 flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-sm font-black text-[#1d1d1f]">聊天记录时间线</p>
          <p className="text-xs text-gray-500 mt-0.5">按日期分组，双向消息顺序阅读；可继续加载更早记录</p>
        </div>
        <div className="flex flex-col items-end gap-2">
          {activeFocusDate ? (
            <span className="inline-flex items-center gap-1 rounded-full bg-[#07c16014] px-2.5 py-1 text-xs font-semibold text-[#07c160]">
              <Clock3 size={12} />
              已定位 {activeFocusDate}
            </span>
          ) : null}
          <form
            className="flex items-center gap-2"
            onSubmit={(event) => {
              event.preventDefault();
              focusOnDate(jumpDate);
            }}
          >
            <input
              type="date"
              value={jumpDate}
              onChange={(event) => setJumpDate(event.target.value)}
              className="rounded-xl border border-gray-200 bg-white px-3 py-1.5 text-xs text-gray-600 focus:border-[#07c160] focus:outline-none"
            />
            <button
              type="submit"
              disabled={!jumpDate}
              className="rounded-xl bg-[#1d1d1f] px-3 py-1.5 text-xs font-semibold text-white disabled:opacity-50"
            >
              跳到当天
            </button>
          </form>
        </div>
      </div>

      {hasMore && (
        <div className="flex justify-center">
          <button
            type="button"
            disabled={loadingMore || loadingInitial}
            onClick={() => void loadOlder()}
            className="px-4 py-2 rounded-full border border-gray-200 bg-white text-sm font-semibold text-gray-600 hover:border-[#07c16066] hover:text-[#07c160] disabled:opacity-60 disabled:cursor-not-allowed"
          >
            {loadingMore ? '加载中...' : '加载更早消息'}
          </button>
        </div>
      )}

      <div className="max-h-[58vh] overflow-y-auto space-y-4 pr-1">
        {loadingInitial ? (
          <div className="h-44 flex items-center justify-center">
            <Loader2 size={28} className="text-[#07c160] animate-spin" />
          </div>
        ) : error ? (
          <div className="rounded-2xl border border-red-100 bg-red-50 px-4 py-8 text-center text-sm text-red-500">
            {error}
          </div>
        ) : groupedMessages.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-4 py-10 text-center text-sm text-gray-400">
            暂无聊天记录
          </div>
        ) : (
          groupedMessages.map((group) => (
            <section
              key={group.date}
              ref={(node) => { dayRefs.current[group.date] = node; }}
              className={`rounded-2xl border p-3 sm:p-4 ${
                activeFocusDate === group.date
                  ? 'border-[#07c16066] bg-[#f3fff8]'
                  : 'border-gray-100 bg-white'
              }`}
            >
              <div className="sticky top-0 z-[1] mb-2.5 inline-flex rounded-full bg-white/95 px-2.5 py-1 text-xs font-bold text-gray-500 shadow-sm border border-gray-100">
                {toDateLabel(group.date)}
              </div>
              <div className="space-y-2">
                {group.items.map((msg, index) => (
                  <div key={`${messageKey(msg)}-${index}`} className={`flex items-end gap-2 ${msg.is_mine ? 'flex-row-reverse' : 'flex-row'}`}>
                    <div className={`w-6 h-6 rounded-full flex-shrink-0 flex items-center justify-center text-white text-[9px] font-black ${
                      msg.is_mine ? 'bg-[#07c160]' : 'bg-[#576b95]'
                    }`}>
                      {msg.is_mine ? '我' : contactName.charAt(0)}
                    </div>
                    <div className={`flex flex-col gap-0.5 max-w-[78%] ${msg.is_mine ? 'items-end' : 'items-start'}`}>
                      <div className={`px-3 py-2 rounded-2xl text-sm leading-relaxed break-words whitespace-pre-wrap ${
                        msg.is_mine
                          ? 'bg-[#07c160] text-white rounded-br-sm'
                          : 'bg-[#f0f0f0] text-[#1d1d1f] rounded-bl-sm'
                      } ${msg.type !== 1 ? 'italic text-xs' : ''}`}>
                        {msg.content}
                      </div>
                      <span className="text-[10px] text-gray-300 px-1">{msg.time}</span>
                    </div>
                  </div>
                ))}
              </div>
            </section>
          ))
        )}
        {loadingFocus && (
          <div className="text-center text-xs text-gray-400 py-2">正在定位指定日期聊天记录...</div>
        )}
      </div>
    </section>
  );
};
