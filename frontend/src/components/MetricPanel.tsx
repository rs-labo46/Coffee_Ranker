import type { ContentMetric } from "../types";

type MetricPanelProps = {
  metric?: ContentMetric;
};

function percent(value: number | undefined): string {
  if (value === undefined) {
    return "-";
  }
  return `${(value * 100).toFixed(1)}%`;
}

function numberText(value: number | undefined): string {
  if (value === undefined) {
    return "-";
  }
  return value.toLocaleString("ja-JP");
}

export function MetricPanel({ metric }: MetricPanelProps) {
  return (
    <section
      className="rounded-3xl border border-white/10 bg-white/[0.04] p-4"
      aria-label="行動指標"
    >
      <h3 className="text-sm font-bold text-white">行動指標</h3>
      <div className="mt-4 grid grid-cols-2 gap-3">
        <div className="rounded-2xl bg-black/20 p-3">
          <span className="text-xs text-stone-400">CTR</span>
          <strong className="mt-1 block text-lg text-white">
            {percent(metric?.click_rate)}
          </strong>
        </div>
        <div className="rounded-2xl bg-black/20 p-3">
          <span className="text-xs text-stone-400">Save</span>
          <strong className="mt-1 block text-lg text-white">
            {percent(metric?.save_rate)}
          </strong>
        </div>
        <div className="rounded-2xl bg-black/20 p-3">
          <span className="text-xs text-stone-400">Good</span>
          <strong className="mt-1 block text-lg text-white">
            {percent(metric?.good_rate)}
          </strong>
        </div>
        <div className="rounded-2xl bg-black/20 p-3">
          <span className="text-xs text-stone-400">View</span>
          <strong className="mt-1 block text-lg text-white">
            {numberText(metric?.content_view_count)}
          </strong>
        </div>
      </div>
    </section>
  );
}
