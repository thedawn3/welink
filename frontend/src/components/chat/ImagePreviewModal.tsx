import React from "react";
import { ExternalLink, X } from "lucide-react";

interface ImagePreviewModalProps {
  src: string;
  alt: string;
  onClose: () => void;
}

export const ImagePreviewModal: React.FC<ImagePreviewModalProps> = ({
  src,
  alt,
  onClose,
}) => (
  <div
    className="fixed inset-0 z-[90] flex items-center justify-center bg-black/80 px-4 py-6 backdrop-blur-sm"
    onClick={onClose}
  >
    <div
      className="relative flex max-h-full w-full max-w-5xl flex-col overflow-hidden rounded-[28px] bg-[#0f1115] shadow-2xl"
      onClick={(event) => event.stopPropagation()}
    >
      <div className="flex items-center justify-between border-b border-white/10 px-4 py-3 text-white">
        <span className="truncate pr-3 text-sm font-semibold">
          {alt || "聊天图片"}
        </span>
        <div className="flex items-center gap-2">
          <a
            href={src}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1 rounded-full border border-white/15 px-3 py-1.5 text-xs font-semibold text-white/85 transition hover:border-white/30 hover:text-white"
          >
            <ExternalLink size={12} />
            原图
          </a>
          <button
            type="button"
            onClick={onClose}
            className="inline-flex h-8 w-8 items-center justify-center rounded-full border border-white/10 text-white/80 transition hover:border-white/25 hover:text-white"
          >
            <X size={16} />
          </button>
        </div>
      </div>
      <div className="flex min-h-[280px] items-center justify-center overflow-auto bg-[#06070a] p-4 sm:p-6">
        <img
          src={src}
          alt={alt}
          className="max-h-[78vh] w-auto max-w-full rounded-2xl object-contain"
        />
      </div>
    </div>
  </div>
);
