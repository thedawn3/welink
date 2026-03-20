import React, { useMemo, useState } from "react";
import type { ChatMessage, GroupChatMessage } from "../../types";
import { ImagePreviewModal } from "./ImagePreviewModal";

type BubbleMessage =
  | Pick<
      ChatMessage,
      | "content"
      | "type"
      | "media_kind"
      | "thumb_url"
      | "media_url"
      | "media_status"
    >
  | Pick<
      GroupChatMessage,
      | "content"
      | "type"
      | "media_kind"
      | "thumb_url"
      | "media_url"
      | "media_status"
    >;

interface ChatMessageBubbleProps {
  message: BubbleMessage;
  isMine?: boolean;
}

export const ChatMessageBubble: React.FC<ChatMessageBubbleProps> = ({
  message,
  isMine = false,
}) => {
  const [previewOpen, setPreviewOpen] = useState(false);

  const imageSrc = useMemo(
    () => message.thumb_url || message.media_url || "",
    [message.media_url, message.thumb_url],
  );
  const previewSrc = useMemo(
    () => message.media_url || message.thumb_url || "",
    [message.media_url, message.thumb_url],
  );
  const isImage = message.media_kind === "image" && Boolean(imageSrc);
  const fallbackText =
    message.content?.trim() || (message.type === 3 ? "[图片]" : "[消息]");

  if (!isImage) {
    return (
      <div
        className={`px-3 py-2 rounded-2xl text-sm leading-relaxed break-words whitespace-pre-wrap ${
          isMine
            ? "bg-[#07c160] text-white rounded-br-sm"
            : "bg-[#f0f0f0] text-[#1d1d1f] rounded-bl-sm"
        } ${message.type !== 1 ? "italic text-xs" : ""}`}
      >
        {fallbackText}
      </div>
    );
  }

  return (
    <>
      <button
        type="button"
        onClick={() => setPreviewOpen(true)}
        className={`group overflow-hidden rounded-[22px] border bg-white text-left shadow-sm transition hover:shadow-md focus:outline-none focus:ring-2 focus:ring-[#07c16055] ${
          isMine
            ? "border-[#07c16033] rounded-br-sm"
            : "border-gray-200 rounded-bl-sm"
        }`}
      >
        <img
          src={imageSrc}
          alt={fallbackText}
          loading="lazy"
          className="block max-h-[260px] w-auto max-w-[220px] bg-[#f3f4f6] object-cover sm:max-w-[280px]"
        />
        <div className="flex items-center justify-between gap-3 border-t border-black/5 bg-white/95 px-3 py-2">
          <span className="truncate text-[11px] font-semibold text-[#1d1d1f]">
            {fallbackText}
          </span>
          <span className="text-[10px] font-medium text-[#07c160] opacity-80 transition group-hover:opacity-100">
            点击查看
          </span>
        </div>
      </button>
      {message.media_status && message.media_status !== "ready" ? (
        <div className="px-1 text-[10px] text-amber-500">
          {message.media_status === "missing_aes_key"
            ? "图片密钥未配置，部分原图可能无法查看"
            : "图片资源不完整"}
        </div>
      ) : null}
      {previewOpen && previewSrc ? (
        <ImagePreviewModal
          src={previewSrc}
          alt={fallbackText}
          onClose={() => setPreviewOpen(false)}
        />
      ) : null}
    </>
  );
};
