import { useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { useAppStore } from '@/store/appStore'
import { StartSync } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { Loader2, CheckCircle, XCircle } from 'lucide-react'

export function SyncPanel() {
  const {
    ipod, navidromeConnected, absConnected,
    syncing, setSyncing,
    syncProgress, setSyncProgress,
    syncError, setSyncError,
  } = useAppStore()

  useEffect(() => {
    const unsub1 = EventsOn('sync:progress', (progress) => {
      setSyncProgress(progress)
    })
    const unsub2 = EventsOn('sync:error', (error) => {
      setSyncError(error)
      setSyncing(false)
    })
    const unsub3 = EventsOn('sync:done', () => {
      setSyncing(false)
      setSyncProgress({ phase: 'done', current: 0, total: 0, message: 'Sync complete!', percent: 100 })
    })
    return () => { unsub1(); unsub2(); unsub3() }
  }, [setSyncProgress, setSyncError, setSyncing])

  const canSync = ipod && (navidromeConnected || absConnected)

  const startSync = async () => {
    setSyncing(true)
    setSyncError(null)
    setSyncProgress(null)
    try {
      await StartSync()
    } catch (e) {
      setSyncError(String(e))
      setSyncing(false)
    }
  }

  return (
    <div className="p-4 space-y-4">
      <h2 className="text-lg font-semibold">Sync</h2>

      <div className="space-y-2 text-sm">
        <div className="flex items-center gap-2">
          {ipod ? <CheckCircle className="h-4 w-4 text-green-600" /> : <XCircle className="h-4 w-4 text-muted-foreground" />}
          iPod: {ipod ? ipod.name : 'Not connected'}
        </div>
        <div className="flex items-center gap-2">
          {navidromeConnected ? <CheckCircle className="h-4 w-4 text-green-600" /> : <XCircle className="h-4 w-4 text-muted-foreground" />}
          Navidrome: {navidromeConnected ? 'Connected' : 'Not connected'}
        </div>
        <div className="flex items-center gap-2">
          {absConnected ? <CheckCircle className="h-4 w-4 text-green-600" /> : <XCircle className="h-4 w-4 text-muted-foreground" />}
          Audiobookshelf: {absConnected ? 'Connected' : 'Not connected'}
        </div>
      </div>

      <Button onClick={startSync} disabled={!canSync || syncing}>
        {syncing && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
        {syncing ? 'Syncing...' : 'Start Sync'}
      </Button>

      {syncProgress && (
        <div className="space-y-2">
          <p className="text-sm">{syncProgress.message}</p>
          {syncProgress.percent > 0 && (
            <Progress value={syncProgress.percent} />
          )}
          {syncProgress.total > 0 && (
            <p className="text-xs text-muted-foreground">
              {syncProgress.current} / {syncProgress.total}
            </p>
          )}
        </div>
      )}

      {syncError && (
        <div className="text-sm text-destructive flex items-center gap-1">
          <XCircle className="h-4 w-4" />
          {syncError}
        </div>
      )}
    </div>
  )
}
