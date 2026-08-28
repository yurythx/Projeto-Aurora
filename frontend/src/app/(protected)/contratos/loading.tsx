export default function ContratosLoading() {
  return (
    <div className="flex h-[calc(100vh-64px)] flex-col gap-6 overflow-hidden pb-6 animate-pulse">
      <div className="flex shrink-0 items-center justify-between">
        <div className="space-y-2">
          <div className="h-6 w-64 bg-surface-hover rounded-md" />
          <div className="h-4 w-96 bg-surface-hover/60 rounded-md" />
        </div>
        <div className="h-9 w-32 bg-surface-hover rounded-md" />
      </div>

      <div className="flex flex-1 gap-4 overflow-x-auto pb-2">
        {[1, 2, 3, 4, 5, 6].map((i) => (
          <div key={i} className="flex w-80 shrink-0 flex-col rounded-lg bg-surface-hover/20 p-4 border border-border/30 space-y-4">
            <div className="flex items-center justify-between">
              <div className="h-4 w-36 bg-surface-hover rounded" />
              <div className="h-5 w-5 bg-surface-hover rounded-full" />
            </div>
            <div className="space-y-3">
              <div className="h-28 w-full bg-surface-hover/40 rounded-lg" />
              <div className="h-28 w-full bg-surface-hover/40 rounded-lg" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
