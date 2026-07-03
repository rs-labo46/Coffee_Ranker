import { useState } from "react";
import type { ContentType } from "../types";

type ContentVisualProps = {
  title: string;
  imageUrl?: string;
  contentType: ContentType;
  compact?: boolean;
};

function gradientClass(contentType: ContentType): string {
  if (contentType === "bean") {
    return "from-amber-950 via-stone-900 to-orange-700";
  }
  return "from-slate-950 via-stone-900 to-emerald-800";
}

export function ContentVisual({
  title,
  imageUrl,
  contentType,
  compact = false,
}: ContentVisualProps) {
  const [failed, setFailed] = useState<boolean>(false);
  const showImage = imageUrl !== undefined && imageUrl.trim() !== "" && !failed;

  return (
    <div
      className={`relative overflow-hidden ${
        compact ? "h-36 rounded-3xl" : "h-full rounded-[1.8rem]"
      } bg-gradient-to-br ${gradientClass(contentType)}`}
    >
      {showImage ? (
        <img
          className="absolute inset-0 h-full w-full object-cover"
          src={imageUrl}
          alt={title}
          loading="lazy"
          onError={() => setFailed(true)}
        />
      ) : null}
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(255,255,255,0.20),transparent_34%),linear-gradient(to_top,rgba(0,0,0,0.84),rgba(0,0,0,0.18),rgba(0,0,0,0.04))]" />
      {!showImage ? (
        <div className="absolute inset-0 flex items-center justify-center px-8 text-center">
          <span className="text-sm font-bold uppercase tracking-[0.28em] text-white/25">
            No Image
          </span>
        </div>
      ) : null}
    </div>
  );
}
