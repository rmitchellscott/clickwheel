import { useEffect, useCallback } from 'react'
import { useAppStore } from '@/store/appStore'
import { cn } from '@/lib/utils'
import { formatBytes } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { CheckCircle, XCircle, AlertTriangle, Play, Loader2, Music, BookOpen, ListMusic, HardDrive, RefreshCw, Podcast, Disc } from 'lucide-react'
import { StartSync, PreviewSync, CancelSync } from '../../wailsjs/go/main/App'
import { useState } from 'react'

export function SyncPage() {
  const {
    ipod, subsonicConnected, absConnected, syncing, setSyncing,
    syncProgress, setSyncProgress, syncError, setSyncError,
    syncPlan, setSyncPlan, syncPlanLoading, setSyncPlanLoading,
    selectedPlaylists, selectedAlbums, selectedBooks, selectedPodcasts,
    playlists, albums, books, podcasts,
  } = useAppStore()

  const [showPlan, setShowPlan] = useState(false)

  const loadPlan = useCallback(async () => {
    if (!ipod || (!subsonicConnected && !absConnected)) return
    setSyncPlanLoading(true)
    setSyncError(null)
    try {
      const plan = await PreviewSync()
      setSyncPlan(plan)
    } catch {
      setSyncPlan(null)
    }
    setSyncPlanLoading(false)
  }, [ipod, subsonicConnected, absConnected, setSyncPlan, setSyncPlanLoading, setSyncError])

  useEffect(() => {
    loadPlan()
  }, [loadPlan])

  const selectedPlaylistList = playlists.filter(p => selectedPlaylists.has(p.id))
  const selectedAlbumList = albums.filter(a => selectedAlbums.has(a.id))
  const selectedBookList = books.filter(b => selectedBooks.has(b.id))
  const selectedPodcastList = podcasts.filter(p => selectedPodcasts.has(p.id))

  const newTrackSize = syncPlan
    ? (syncPlan.addTracks || []).reduce((a, t) => a + (t.size || 0), 0)
    : 0
  const newBookSize = syncPlan
    ? (syncPlan.addBooks || []).reduce((a, b) => a + (b.size || 0), 0)
    : 0
  const newPodcastSize = syncPlan
    ? (syncPlan.addPodcasts || []).reduce((a, p) => a + (p.size || 0), 0)
    : 0
  const estimatedNewContent = newTrackSize + newBookSize + newPodcastSize
  const newTrackCount = syncPlan ? (syncPlan.addTracks || []).length : 0
  const newBookCount = syncPlan ? (syncPlan.addBooks || []).length : 0
  const newPodcastCount = syncPlan ? (syncPlan.addPodcasts || []).length : 0
  const removeTrackCount = syncPlan ? (syncPlan.removeTracks || 0) : 0
  const removeBookCount = syncPlan ? (syncPlan.removeBooks || 0) : 0
  const removePodcastCount = syncPlan ? (syncPlan.removePodcasts || 0) : 0
  const removeCount = removeTrackCount + removeBookCount + removePodcastCount
  const playsToSync = syncPlan ? (syncPlan.playsToSync || 0) : 0
  const booksToIPod = syncPlan ? (syncPlan.booksToIPod || []) : []
  const booksFromIPod = syncPlan ? (syncPlan.booksFromIPod || []) : []
  const podcastsToIPod = syncPlan ? (syncPlan.podcastsToIPod || []) : []
  const podcastsFromIPod = syncPlan ? (syncPlan.podcastsFromIPod || []) : []
  const playlistsChanged = syncPlan ? (syncPlan.playlistsChanged || []) : []

  const currentUsed = ipod ? ipod.totalSpace - ipod.freeSpace : 0
  const projectedUsed = currentUsed + estimatedNewContent
  const projectedFree = ipod ? ipod.totalSpace - projectedUsed : 0
  const willFit = ipod ? projectedUsed <= ipod.totalSpace : true
  const projectedPercent = ipod ? Math.min(100, (projectedUsed / ipod.totalSpace) * 100) : 0
  const currentPercent = ipod ? (currentUsed / ipod.totalSpace) * 100 : 0

  const inSync = syncPlan && newTrackCount === 0 && newBookCount === 0 && newPodcastCount === 0 && removeCount === 0 && playsToSync === 0 && booksToIPod.length === 0 && booksFromIPod.length === 0 && podcastsToIPod.length === 0 && podcastsFromIPod.length === 0 && playlistsChanged.length === 0

  const canSync = ipod && (subsonicConnected || absConnected)

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
    <div className="flex flex-col h-full">
      <div className="p-4 pb-3 border-b">
        <h2 className="text-lg font-semibold">Sync</h2>
        <p className="text-sm text-muted-foreground">Review and sync your selections to iPod</p>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        <div className="grid grid-cols-3 gap-3">
          <StatusCard
            label="iPod"
            value={ipod ? ipod.name : 'Not connected'}
            connected={!!ipod}
            icon={<HardDrive className="h-4 w-4" />}
            detail={ipod ? `${formatBytes(ipod.freeSpace)} free` : undefined}
          />
          <StatusCard
            label="Subsonic"
            value={subsonicConnected ? 'Connected' : 'Not connected'}
            connected={subsonicConnected}
            icon={<Music className="h-4 w-4" />}
          />
          <StatusCard
            label="Audiobookshelf"
            value={absConnected ? 'Connected' : 'Not connected'}
            connected={absConnected}
            icon={<BookOpen className="h-4 w-4" />}
          />
        </div>

        {ipod && (
          <div className="border rounded-lg p-4 space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">
                {syncPlanLoading ? 'Calculating...' : inSync ? 'Storage' : 'Storage after sync'}
              </span>
              {syncPlanLoading ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" />
              ) : !willFit ? (
                <div className="flex items-center gap-1.5 text-destructive">
                  <AlertTriangle className="h-3.5 w-3.5" />
                  <span className="text-xs font-medium">Won't fit &mdash; {formatBytes(projectedUsed - ipod.totalSpace)} over capacity</span>
                </div>
              ) : inSync ? (
                <span className="text-xs text-muted-foreground">{formatBytes(ipod.freeSpace)} free</span>
              ) : (
                <span className="text-xs text-muted-foreground">{formatBytes(projectedFree)} will remain free</span>
              )}
            </div>
            <div className="relative h-4 rounded-full bg-muted overflow-hidden">
              {!inSync && estimatedNewContent > 0 && (
                <div
                  className={cn("absolute inset-y-0 left-0 rounded-full transition-all",
                    willFit ? "bg-blue-400 dark:bg-blue-500" : "bg-destructive")}
                  style={{ width: `${projectedPercent}%` }}
                />
              )}
              <div
                className="absolute inset-y-0 left-0 rounded-full bg-foreground/70 dark:bg-foreground/60 transition-all"
                style={{ width: `${currentPercent}%` }}
              />
            </div>
            <div className="flex items-center gap-4 text-xs text-muted-foreground">
              <div className="flex items-center gap-1.5">
                <div className="h-2 w-2 rounded-full bg-foreground/70 dark:bg-foreground/60" />
                <span>Current ({formatBytes(currentUsed)})</span>
              </div>
              {!inSync && estimatedNewContent > 0 && (
                <div className="flex items-center gap-1.5">
                  <div className={cn("h-2 w-2 rounded-full", willFit ? "bg-blue-400 dark:bg-blue-500" : "bg-destructive")} />
                  <span>New content ({formatBytes(estimatedNewContent)})</span>
                </div>
              )}
              <div className="flex items-center gap-1.5">
                <div className="h-2 w-2 rounded-full bg-muted ring-1 ring-border" />
                <span>Free</span>
              </div>
              <span className="ml-auto font-medium">{formatBytes(ipod.totalSpace)} total</span>
            </div>
          </div>
        )}

        <div className="border rounded-lg">
          <button
            onClick={() => setShowPlan(!showPlan)}
            className="w-full flex items-center justify-between px-4 py-3 text-sm hover:bg-accent/30 transition-colors"
          >
            <span className="font-medium">Sync Summary</span>
            <div className="flex items-center gap-3 text-xs text-muted-foreground">
              {syncPlan && !syncPlanLoading ? (
                inSync ? (
                  <span className="text-blue-500">Up to date</span>
                ) : (
                  <>
                    {newTrackCount > 0 && <span>{newTrackCount} new track{newTrackCount !== 1 ? 's' : ''}</span>}
                    {newBookCount > 0 && <span>{newBookCount} new book{newBookCount !== 1 ? 's' : ''}</span>}
                    {newPodcastCount > 0 && <span>{newPodcastCount} new episode{newPodcastCount !== 1 ? 's' : ''}</span>}
                    {removeTrackCount > 0 && <span>{removeTrackCount} track{removeTrackCount !== 1 ? 's' : ''} to remove</span>}
                    {removeBookCount > 0 && <span>{removeBookCount} book{removeBookCount !== 1 ? 's' : ''} to remove</span>}
                    {removePodcastCount > 0 && <span>{removePodcastCount} episode{removePodcastCount !== 1 ? 's' : ''} to remove</span>}
                    {playsToSync > 0 && <span>{playsToSync} play{playsToSync !== 1 ? 's' : ''} to report</span>}
                    {booksToIPod.length > 0 && <span>{booksToIPod.length} book position{booksToIPod.length !== 1 ? 's' : ''} → iPod</span>}
                    {booksFromIPod.length > 0 && <span>{booksFromIPod.length} book position{booksFromIPod.length !== 1 ? 's' : ''} → server</span>}
                    {podcastsToIPod.length > 0 && <span>{podcastsToIPod.length} podcast position{podcastsToIPod.length !== 1 ? 's' : ''} → iPod</span>}
                    {podcastsFromIPod.length > 0 && <span>{podcastsFromIPod.length} podcast position{podcastsFromIPod.length !== 1 ? 's' : ''} → server</span>}
                    {playlistsChanged.length > 0 && <span>{playlistsChanged.length} playlist{playlistsChanged.length !== 1 ? 's' : ''} changed</span>}
                  </>
                )
              ) : !syncPlanLoading ? (
                <>
                  <span>{selectedPlaylistList.length} playlist{selectedPlaylistList.length !== 1 ? 's' : ''}</span>
                  <span>{selectedAlbumList.length} album{selectedAlbumList.length !== 1 ? 's' : ''}</span>
                  <span>{selectedBookList.length} audiobook{selectedBookList.length !== 1 ? 's' : ''}</span>
                  {selectedPodcastList.length > 0 && <span>{selectedPodcastList.length} podcast{selectedPodcastList.length !== 1 ? 's' : ''}</span>}
                </>
              ) : null}
            </div>
          </button>

          {showPlan && (
            <div className="border-t px-4 py-3 space-y-3 text-sm">
              {syncPlan && (syncPlan.addTracks || []).length > 0 && (
                <div>
                  <div className="flex items-center gap-2 mb-1.5">
                    <Music className="h-3.5 w-3.5 text-muted-foreground" />
                    <span className="font-medium text-xs uppercase tracking-wider text-muted-foreground">
                      New Tracks ({syncPlan.addTracks.length} &middot; {formatBytes(newTrackSize)})
                    </span>
                  </div>
                  <div className="grid grid-cols-2 gap-1">
                    {syncPlan.addTracks.map((t, i) => (
                      <div key={i} className="text-sm px-2 py-0.5">
                        <span className="text-foreground">{t.title}</span>
                        <span className="text-muted-foreground text-xs ml-1">- {t.artist}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {syncPlan && (syncPlan.addBooks || []).length > 0 && (
                <div>
                  <div className="flex items-center gap-2 mb-1.5">
                    <BookOpen className="h-3.5 w-3.5 text-muted-foreground" />
                    <span className="font-medium text-xs uppercase tracking-wider text-muted-foreground">
                      New Books ({syncPlan.addBooks.length} &middot; {formatBytes(newBookSize)})
                    </span>
                  </div>
                  <div className="space-y-1">
                    {syncPlan.addBooks.map((b, i) => (
                      <div key={i} className="text-sm px-2 py-0.5">
                        <span>{b.title}</span>
                        <span className="text-muted-foreground text-xs ml-1">- {b.artist} ({formatBytes(b.size)})</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {syncPlan && (syncPlan.addPodcasts || []).length > 0 && (
                <div>
                  <div className="flex items-center gap-2 mb-1.5">
                    <Podcast className="h-3.5 w-3.5 text-muted-foreground" />
                    <span className="font-medium text-xs uppercase tracking-wider text-muted-foreground">
                      New Episodes ({syncPlan.addPodcasts.length} &middot; {formatBytes(newPodcastSize)})
                    </span>
                  </div>
                  <div className="space-y-1">
                    {syncPlan.addPodcasts.map((p, i) => (
                      <div key={i} className="text-sm px-2 py-0.5">
                        <span>{p.title}</span>
                        <span className="text-muted-foreground text-xs ml-1">- {p.artist}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {syncPlan && removeTrackCount > 0 && (
                <div className="text-sm text-muted-foreground px-2">
                  {removeTrackCount} track{removeTrackCount !== 1 ? 's' : ''} will be removed from iPod
                </div>
              )}

              {syncPlan && removeBookCount > 0 && (
                <div className="text-sm text-muted-foreground px-2">
                  {removeBookCount} book{removeBookCount !== 1 ? 's' : ''} will be removed from iPod
                </div>
              )}

              {syncPlan && removePodcastCount > 0 && (
                <div className="text-sm text-muted-foreground px-2">
                  {removePodcastCount} episode{removePodcastCount !== 1 ? 's' : ''} will be removed from iPod
                </div>
              )}

              {syncPlan && playsToSync > 0 && (
                <div className="text-sm text-muted-foreground px-2">
                  {playsToSync} play{playsToSync !== 1 ? 's' : ''} to report to server
                </div>
              )}

              {syncPlan && booksToIPod.length > 0 && (
                <div className="text-sm text-muted-foreground px-2">
                  Update {booksToIPod.length} book position{booksToIPod.length !== 1 ? 's' : ''} on iPod
                </div>
              )}

              {syncPlan && booksFromIPod.length > 0 && (
                <div className="text-sm text-muted-foreground px-2">
                  Update {booksFromIPod.length} book position{booksFromIPod.length !== 1 ? 's' : ''} on server
                </div>
              )}

              {syncPlan && podcastsToIPod.length > 0 && (
                <div className="text-sm text-muted-foreground px-2">
                  Update {podcastsToIPod.length} podcast position{podcastsToIPod.length !== 1 ? 's' : ''} on iPod
                </div>
              )}

              {syncPlan && podcastsFromIPod.length > 0 && (
                <div className="text-sm text-muted-foreground px-2">
                  Update {podcastsFromIPod.length} podcast position{podcastsFromIPod.length !== 1 ? 's' : ''} on server
                </div>
              )}

              {syncPlan && playlistsChanged.length > 0 && (
                <div className="text-sm text-muted-foreground px-2">
                  Playlists to update: {playlistsChanged.join(', ')}
                </div>
              )}

              {inSync && (
                <div className="text-sm text-muted-foreground px-2 flex items-center gap-2">
                  <CheckCircle className="h-3.5 w-3.5 text-blue-500" />
                  iPod is up to date with your selections
                </div>
              )}

              {!syncPlan && !syncPlanLoading && (
                <div className="space-y-3">
                  {selectedPlaylistList.length > 0 && (
                    <div>
                      <div className="flex items-center gap-2 mb-1.5">
                        <ListMusic className="h-3.5 w-3.5 text-muted-foreground" />
                        <span className="font-medium text-xs uppercase tracking-wider text-muted-foreground">Playlists</span>
                      </div>
                      <div className="grid grid-cols-2 gap-1">
                        {selectedPlaylistList.map(p => (
                          <div key={p.id} className="text-sm text-muted-foreground px-2 py-0.5">
                            {p.name} <span className="text-xs">({p.songCount})</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {selectedAlbumList.length > 0 && (
                    <div>
                      <div className="flex items-center gap-2 mb-1.5">
                        <Disc className="h-3.5 w-3.5 text-muted-foreground" />
                        <span className="font-medium text-xs uppercase tracking-wider text-muted-foreground">Albums</span>
                      </div>
                      <div className="grid grid-cols-2 gap-1">
                        {selectedAlbumList.map(a => (
                          <div key={a.id} className="text-sm text-muted-foreground px-2 py-0.5">
                            {a.name} <span className="text-xs">- {a.artist}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {selectedBookList.length > 0 && (
                    <div>
                      <div className="flex items-center gap-2 mb-1.5">
                        <BookOpen className="h-3.5 w-3.5 text-muted-foreground" />
                        <span className="font-medium text-xs uppercase tracking-wider text-muted-foreground">Audiobooks</span>
                      </div>
                      <div className="space-y-1">
                        {selectedBookList.map(b => (
                          <div key={b.id} className="text-sm px-2 py-0.5">
                            <span>{b.title}</span>
                            <span className="text-muted-foreground text-xs ml-1">- {b.author} ({formatBytes(b.size)})</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {selectedPodcastList.length > 0 && (
                    <div>
                      <div className="flex items-center gap-2 mb-1.5">
                        <Podcast className="h-3.5 w-3.5 text-muted-foreground" />
                        <span className="font-medium text-xs uppercase tracking-wider text-muted-foreground">Podcasts</span>
                      </div>
                      <div className="grid grid-cols-2 gap-1">
                        {selectedPodcastList.map(p => (
                          <div key={p.id} className="text-sm text-muted-foreground px-2 py-0.5">
                            {p.title} <span className="text-xs">({p.episodeCount} episode{p.episodeCount !== 1 ? 's' : ''})</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}

              {syncPlanLoading && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground px-2">
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  Building sync plan...
                </div>
              )}
            </div>
          )}
        </div>

        {syncing && syncProgress && (
          <div className="border rounded-lg p-4 space-y-3">
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
                  className="text-xs text-muted-foreground text-right"
                  style={{
                    fontVariantNumeric: 'tabular-nums',
                    minWidth: `${(syncProgress.total.toString().length * 2 + 3)}ch`,
                  }}
                >
                  {syncProgress.current} / {syncProgress.total}
                </span>
              </div>
            </div>
            <Progress value={syncProgress.percent} />
            <p className="text-xs text-muted-foreground">{syncProgress.message}</p>
          </div>
        )}

        {!syncing && syncProgress?.phase === 'done' && (
          <div className="border border-blue-200 dark:border-blue-900 bg-blue-50 dark:bg-blue-950/30 rounded-lg p-4 flex items-center gap-3">
            <CheckCircle className="h-5 w-5 text-blue-500" />
            <div>
              <p className="text-sm font-medium text-blue-800 dark:text-blue-400">Sync complete!</p>
              <p className="text-xs text-blue-600 dark:text-blue-500">All content has been transferred to your iPod.</p>
            </div>
          </div>
        )}

        {syncError && (
          <div className="text-sm text-destructive flex items-center gap-1">
            <XCircle className="h-4 w-4" />
            {syncError}
          </div>
        )}

        <div className="flex items-center gap-3">
          {syncing ? (
            <Button variant="destructive" size="lg" onClick={() => CancelSync()} className="gap-2">
              Cancel Sync
            </Button>
          ) : (
            <>
              <Button onClick={startSync} disabled={!canSync || !!inSync} size="lg" className="gap-2">
                <Play className="h-4 w-4" />
                Start Sync
              </Button>
              {canSync && !syncPlanLoading && (
                <Button variant="outline" size="lg" onClick={loadPlan} className="gap-2">
                  <RefreshCw className="h-4 w-4" />
                  Refresh Plan
                </Button>
              )}
              {!canSync && (
                <span className="text-sm text-muted-foreground">
                  {!ipod ? 'Connect an iPod to sync' : 'Connect to a server first'}
                </span>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}

function StatusCard({ label, value, connected, icon, detail }: {
  label: string
  value: string
  connected: boolean
  icon: React.ReactNode
  detail?: string
}) {
  return (
    <div className={cn(
      'border rounded-lg p-3 space-y-1',
      connected ? 'border-blue-200 dark:border-blue-900' : 'border-border'
    )}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-muted-foreground">
          {icon}
          <span className="text-xs font-medium uppercase tracking-wider">{label}</span>
        </div>
        {connected
          ? <CheckCircle className="h-3.5 w-3.5 text-blue-500" />
          : <XCircle className="h-3.5 w-3.5 text-muted-foreground/40" />
        }
      </div>
      <p className="text-sm font-medium truncate">{value}</p>
      {detail && <p className="text-xs text-muted-foreground">{detail}</p>}
    </div>
  )
}
