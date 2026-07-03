type LoadingStateProps = {
  error: string | null
  onRetry: () => void
}

export function LoadingState({ error, onRetry }: LoadingStateProps) {
  if (error !== null) {
    return (
      <div className="flex h-[calc(100svh-7.5rem)] min-h-[560px] flex-col justify-end rounded-[2rem] border border-red-400/20 bg-red-950/30 p-8 shadow-2xl shadow-black/40">
        <p className="text-xs font-bold uppercase tracking-[0.28em] text-red-200">API Error</p>
        <h2 className="mt-3 text-3xl font-black tracking-tight text-white">API取得に失敗しています</h2>
        <p className="mt-3 max-w-md text-sm leading-6 text-red-100/80">{error}</p>
        <button
          type="button"
          className="mt-6 w-fit rounded-full bg-white px-5 py-3 text-sm font-bold text-stone-950 transition hover:bg-stone-200"
          onClick={onRetry}
        >
          再読み込み
        </button>
      </div>
    )
  }

  return (
    <div className="flex h-[calc(100svh-7.5rem)] min-h-[560px] flex-col items-center justify-center rounded-[2rem] border border-white/10 bg-stone-900 shadow-2xl shadow-black/40">
      <span className="h-10 w-10 animate-spin rounded-full border-2 border-amber-300 border-t-transparent" aria-hidden="true" />
      <p className="mt-5 text-sm font-semibold text-stone-300">コーヒーフィードを読み込み中</p>
    </div>
  )
}
