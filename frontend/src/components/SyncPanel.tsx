import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { useAppStore } from '@/store/appStore'
import { StartSync, PreviewSync } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { Loader2, CheckCircle, XCircle, RefreshCw } from 'lucide-react'

interface PlanSummaryItem {
  title: string
  artist: string
  size: number
}

interface PlanSummary {
  addTracks: PlanSummaryItem[] | null
  addBooks: PlanSummaryItem[] | null
  remove: number
  playlists: string[] | null
}

export function SyncPanel() {
  const {
    ipod, navidromeConnected, absConnected,
    syncing, setSyncing,
    syncProgress, setSyncProgress,
    syncError, setSyncError,
  } = useAppStore()

  const [plan, setPlan] = useState<PlanSummary | null>(null)
  const [planLoading, setPlanLoading] = useState(false)
  const [planError, setPlanError] = useState<string | null>(null)

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
      setPlan(null)
    })
    return () => { unsub1(); unsub2(); unsub3() }
  }, [setSyncProgress, setSyncError, setSyncing])

  const canSync = ipod && (navidromeConnected || absConnected)

  const loadPlan = async () => {
    setPlanLoading(true)
    setPlanError(null)
    try {
      const result = await PreviewSync()
      setPlan(result)
    } catch (e) {
      setPlanError(String(e))
    }
    setPlanLoading(false)
  }

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

  const addTracks = plan?.addTracks || []
  const addBooks = plan?.addBooks || []
  const playlists = plan?.playlists || []
  const totalAdd = addTracks.length + addBooks.length
  const nothingToDo = plan && totalAdd === 0 && plan.remove === 0

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

      {!syncing && !plan && (
        <Button onClick={loadPlan} disabled={!canSync || planLoading}>
          {planLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          {planLoading ? 'Building plan...' : 'Preview Sync'}
        </Button>
      )}

      {planError && (
        <div className="text-sm text-destructive flex items-center gap-1">
          <XCircle className="h-4 w-4" />
          {planError}
        </div>
      )}

      {plan && !syncing && (
        <div className="space-y-3 border rounded-md p-3">
          <div className="flex items-center justify-between">
            <h3 className="font-medium text-sm">Sync Plan</h3>
            <Button variant="ghost" size="sm" onClick={loadPlan} disabled={planLoading}>
              <RefreshCw className="h-3 w-3" />
            </Button>
          </div>

          {nothingToDo ? (
            <p className="text-sm text-muted-foreground">Everything is up to date.</p>
          ) : (
            <div className="space-y-2 text-sm">
              {addTracks.length > 0 && (
                <div>
                  <p className="font-medium">Add {addTracks.length} track{addTracks.length !== 1 ? 's' : ''}</p>
                  <ul className="ml-4 text-muted-foreground max-h-32 overflow-y-auto">
                    {addTracks.map((t, i) => (
                      <li key={i}>{t.artist} — {t.title}{t.size > 0 ? ` (${formatBytes(t.size)})` : ''}</li>
                    ))}
                  </ul>
                </div>
              )}
              {addBooks.length > 0 && (
                <div>
                  <p className="font-medium">Add {addBooks.length} audiobook{addBooks.length !== 1 ? 's' : ''}</p>
                  <ul className="ml-4 text-muted-foreground">
                    {addBooks.map((b, i) => (
                      <li key={i}>{b.artist} — {b.title}</li>
                    ))}
                  </ul>
                </div>
              )}
              {plan.remove > 0 && (
                <p className="font-medium">Remove {plan.remove} track{plan.remove !== 1 ? 's' : ''}</p>
              )}
            </div>
          )}

          {playlists.length > 0 && (
            <div className="text-sm">
              <p className="font-medium">Playlists ({playlists.length})</p>
              <ul className="ml-4 text-muted-foreground">
                {playlists.map((name, i) => <li key={i}>{name}</li>)}
              </ul>
            </div>
          )}

          <div className="flex gap-2 pt-1">
            <Button onClick={startSync} disabled={syncing}>
              Start Sync
            </Button>
            <Button variant="outline" onClick={() => setPlan(null)}>
              Cancel
            </Button>
          </div>
        </div>
      )}

      {syncing && syncProgress && (
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

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}
