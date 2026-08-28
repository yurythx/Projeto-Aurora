export default function MonitoramentoLoading() {
  return (
    <div className="flex flex-col gap-6 animate-pulse">
      <div className="h-28 w-full bg-surface-hover/30 rounded-xl border border-surface-border" />
      <div className="h-40 w-full bg-surface-hover/30 rounded-xl border border-surface-border" />
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="h-36 bg-surface-hover/30 rounded-xl border border-surface-border" />
        <div className="h-36 bg-surface-hover/30 rounded-xl border border-surface-border" />
        <div className="h-36 bg-surface-hover/30 rounded-xl border border-surface-border" />
      </div>
    </div>
  );
}
