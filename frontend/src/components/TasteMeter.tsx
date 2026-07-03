type TasteMeterProps = {
  label: string
  value?: number
}

export function TasteMeter({ label, value }: TasteMeterProps) {
  const normalized = value ?? 0

  return (
    <div className="rounded-2xl border border-white/10 bg-black/20 p-3">
      <div className="flex items-center justify-between gap-3">
        <span className="text-xs font-semibold text-stone-300">{label}</span>
        <strong className="text-sm text-white">{value ?? '-'}</strong>
      </div>
      <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-white/10" aria-hidden="true">
        <span
          className="block h-full rounded-full bg-amber-300"
          style={{ width: `${Math.min(Math.max(normalized, 0), 5) * 20}%` }}
        />
      </div>
    </div>
  )
}
