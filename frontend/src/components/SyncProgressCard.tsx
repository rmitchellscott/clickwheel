import { useAppStore } from '@/store/appStore'
import { cn } from '@/lib/utils'
import { Progress } from '@/components/ui/progress'
import { Loader2, CheckCircle } from 'lucide-react'

export function SyncProgressCard() {
  const { syncing, syncProgress, syncComplete, page, setPage } = useAppStore()

  const visible = (syncing || syncComplete) && page !== 'sync'

  return (
    <div className={cn(
      'absolute bottom-4 left-4 right-4 z-10 transition-all duration-300',
      visible ? 'translate-y-0 opacity-100' : 'translate-y-full opacity-0 pointer-events-none'
    )}>
      <div className="border rounded-lg bg-background/95 backdrop-blur shadow-lg p-3 space-y-2">
        {syncComplete ? (
          <div className="flex items-center gap-2">
            <CheckCircle className="h-4 w-4 text-blue-500" />
            <span className="text-sm font-medium">Sync complete!</span>
          </div>
        ) : syncProgress ? (
          <>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                <span className="text-sm font-medium">Syncing...</span>
              </div>
              <div className="flex items-center gap-3">
                {syncProgress.eta && (
                  <span className="text-xs text-muted-foreground">{syncProgress.eta} remaining</span>
                )}
                <span
                  className="text-xs text-muted-foreground"
                  style={{ fontVariantNumeric: 'tabular-nums' }}
                >
                  {syncProgress.current} / {syncProgress.total}
                </span>
                <button
                  onClick={() => setPage('sync')}
                  className="text-xs text-blue-500 hover:underline"
                >
                  View
                </button>
              </div>
            </div>
            <Progress value={syncProgress.percent} />
            <p className="text-xs text-muted-foreground truncate">{syncProgress.message}</p>
          </>
        ) : (
          <div className="flex items-center gap-2">
            <Loader2 className="h-4 w-4 animate-spin" />
            <span className="text-sm font-medium">Syncing...</span>
          </div>
        )}
      </div>
    </div>
  )
}
