type PageLoadingProps = {
  /** Visible status text; keep short. Omit for spinner-only. */
  label?: string
}

/** Centered page-level loading indicator. */
export function PageLoading({ label }: PageLoadingProps) {
  return (
    <div className="page-loading" role="status" aria-live="polite" aria-busy="true">
      <div className="page-loading-spinner" aria-hidden />
      <span className="page-loading-sr">{label || '加载中'}</span>
    </div>
  )
}
