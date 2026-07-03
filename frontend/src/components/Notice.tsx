import type { Notice as NoticeType } from "../types";

type NoticeProps = {
  notice: NoticeType | null;
};

function toneClass(tone: NoticeType["tone"]): string {
  switch (tone) {
    case "success":
      return "border-emerald-400/30 bg-emerald-400/10 text-emerald-100";
    case "error":
      return "border-red-400/30 bg-red-400/10 text-red-100";
    case "info":
      return "border-sky-400/30 bg-sky-400/10 text-sky-100";
  }
}

export function Notice({ notice }: NoticeProps) {
  if (notice === null) {
    return null;
  }

  return (
    <div
      className={`rounded-2xl border px-4 py-3 text-sm leading-6 ${toneClass(notice.tone)}`}
    >
      {notice.message}
    </div>
  );
}
