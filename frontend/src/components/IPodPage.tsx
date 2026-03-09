import { useState, useMemo, useEffect } from 'react'
import { useAppStore } from '@/store/appStore'
import { cn } from '@/lib/utils'
import { formatBytes } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { Search, Music, BookOpen, Podcast, ArrowUpDown, Clock, TrendingUp, ChevronUp, ChevronDown, Download, Check, X, FolderDown, ListMusic, Folder, Loader2, CheckCircle, RotateCcw, Pencil } from 'lucide-react'
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'
import { IPodIcon } from '@/components/IPodIcon'
import { GetIPodTracks, GetIPodPlaylists, BrowseDirectory, CopyTracksToComputer, RenameIPod, MountIPod } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import type { IPodTrack } from '@/store/appStore'

type SortKey = 'title' | 'artist' | 'album' | 'playCount' | 'lastPlayed' | 'dateAdded'
type SortDir = 'asc' | 'desc'
type TrackFilter = 'all' | 'music' | 'audiobook' | 'podcast'
type IPodTab = 'overview' | 'playlists' | 'library'

function formatAbsolute(date: Date, tz: string): string {
  return date.toLocaleString(undefined, {
    timeZone: tz,
    weekday: 'short', year: 'numeric', month: 'short', day: 'numeric',
    hour: 'numeric', minute: '2-digit',
  })
}

function formatShortDate(date: Date, tz: string): string {
  const now = new Date()
  const thisYear = now.getFullYear() === date.getFullYear()
  return date.toLocaleDateString(undefined, {
    timeZone: tz,
    month: 'short', day: 'numeric',
    ...(!thisYear && { year: 'numeric' }),
  })
}

function TimeAgo({ ts, className }: { ts: number; className?: string }) {
  const tz = useAppStore(s => s.timezone)
  if (ts === 0) return <span className={className}>Never</span>

  const date = new Date(ts * 1000)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const days = Math.floor(diffMs / 86400000)

  if (days >= 14) {
    return <span className={className}>{formatShortDate(date, tz)}</span>
  }

  const mins = Math.floor(diffMs / 60000)
  const hours = Math.floor(mins / 60)
  let relative: string
  if (mins < 60) relative = `${mins}m ago`
  else if (hours < 24) relative = `${hours}h ago`
  else if (days === 1) relative = 'Yesterday'
  else if (days < 7) relative = `${days}d ago`
  else relative = `${Math.floor(days / 7)}w ago`

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={cn(className, 'cursor-default')}>{relative}</span>
      </TooltipTrigger>
      <TooltipContent>{formatAbsolute(date, tz)}</TooltipContent>
    </Tooltip>
  )
}

function formatTrackDuration(seconds: number): string {
  if (seconds >= 3600) {
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    return `${h}:${String(m).padStart(2, '0')}:00`
  }
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${m}:${String(s).padStart(2, '0')}`
}

export function IPodPage() {
  const { ipod, setIPod, ipodTracks, setIPodTracks, ipodPlaylists, setIPodPlaylists, setRestoreModalOpen } = useAppStore()
  const [tab, setTab] = useState<IPodTab>('overview')
  const [expandedPlaylist, setExpandedPlaylist] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [editingName, setEditingName] = useState(false)
  const [nameInput, setNameInput] = useState('')
  const [sortKey, setSortKey] = useState<SortKey>('lastPlayed')
  const [sortDir, setSortDir] = useState<SortDir>('desc')
  const [filter, setFilter] = useState<TrackFilter>('all')
  const [selectedTracks, setSelectedTracks] = useState<Set<string>>(new Set())
  const [copyModal, setCopyModal] = useState<'pick' | 'copying' | 'done' | null>(null)
  const [copyDest, setCopyDest] = useState('')
  const [copyProgress, setCopyProgress] = useState({ current: 0, total: 0, currentFile: '' })
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    if (!ipod) {
      setLoaded(false)
      setIPodTracks([])
      setIPodPlaylists([])
      return
    }
    if (!loaded) {
      Promise.all([GetIPodTracks(), GetIPodPlaylists()]).then(([tracks, playlists]) => {
        setIPodTracks((tracks || []) as IPodTrack[])
        setIPodPlaylists(playlists || [])
        setLoaded(true)
      }).catch(() => {})
    }
  }, [ipod, loaded])

  useEffect(() => {
    const unsub1 = EventsOn('copy:progress', (data: any) => {
      setCopyProgress({ current: data.current, total: data.total, currentFile: data.currentFile })
    })
    const unsub2 = EventsOn('copy:done', () => {
      setCopyModal('done')
    })
    const unsub3 = EventsOn('copy:error', (err: string) => {
      setCopyModal(null)
      alert('Copy error: ' + err)
    })
    return () => { unsub1(); unsub2(); unsub3() }
  }, [])

  const musicTracks = ipodTracks.filter(t => t.type === 'music')
  const audiobookTracks = ipodTracks.filter(t => t.type === 'audiobook')
  const podcastTracks = ipodTracks.filter(t => t.type === 'podcast')

  const musicSize = musicTracks.reduce((a, t) => a + t.size, 0)
  const audiobookSize = audiobookTracks.reduce((a, t) => a + t.size, 0)
  const podcastSize = podcastTracks.reduce((a, t) => a + t.size, 0)
  const totalContentSize = musicSize + audiobookSize + podcastSize
  const otherSize = ipod ? (ipod.totalSpace - ipod.freeSpace) - totalContentSize : 0

  const totalPlayCount = ipodTracks.reduce((a, t) => a + t.playCount, 0)
  const totalDuration = ipodTracks.reduce((a, t) => a + t.duration, 0)
  const totalHours = Math.round(totalDuration / 3600)

  const recentlyPlayed = useMemo(() =>
    [...ipodTracks]
      .filter(t => t.lastPlayed > 0)
      .sort((a, b) => b.lastPlayed - a.lastPlayed)
      .slice(0, 8),
    [ipodTracks]
  )

  const mostPlayed = useMemo(() =>
    [...ipodTracks]
      .sort((a, b) => b.playCount - a.playCount)
      .slice(0, 8),
    [ipodTracks]
  )

  const filteredTracks = useMemo(() => {
    let tracks = filter === 'all' ? ipodTracks : ipodTracks.filter(t => t.type === filter)
    if (search) {
      const q = search.toLowerCase()
      tracks = tracks.filter(t =>
        t.title.toLowerCase().includes(q) ||
        t.artist.toLowerCase().includes(q) ||
        t.album.toLowerCase().includes(q)
      )
    }
    return [...tracks].sort((a, b) => {
      let cmp = 0
      switch (sortKey) {
        case 'title': cmp = a.title.localeCompare(b.title); break
        case 'artist': cmp = a.artist.localeCompare(b.artist); break
        case 'album': cmp = a.album.localeCompare(b.album); break
        case 'playCount': cmp = a.playCount - b.playCount; break
        case 'lastPlayed': cmp = a.lastPlayed - b.lastPlayed; break
        case 'dateAdded': cmp = a.dateAdded - b.dateAdded; break
      }
      return sortDir === 'asc' ? cmp : -cmp
    })
  }, [ipodTracks, filter, search, sortKey, sortDir])

  const toggleTrack = (id: string) => {
    setSelectedTracks(prev => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }

  const toggleAllVisible = () => {
    const allSelected = filteredTracks.every(t => selectedTracks.has(t.id))
    setSelectedTracks(allSelected ? new Set() : new Set(filteredTracks.map(t => t.id)))
  }

  const selectedSize = ipodTracks
    .filter(t => selectedTracks.has(t.id))
    .reduce((a, t) => a + t.size, 0)

  const startCopyPicker = () => setCopyModal('pick')

  const browseDest = async () => {
    try {
      const dir = await BrowseDirectory()
      if (dir) setCopyDest(dir)
    } catch {}
  }

  const startCopy = async () => {
    const ids = Array.from(selectedTracks)
    setCopyModal('copying')
    setCopyProgress({ current: 0, total: ids.length, currentFile: '' })
    try {
      await CopyTracksToComputer(ids, copyDest)
    } catch (e) {
      setCopyModal(null)
      alert('Copy error: ' + String(e))
    }
  }

  const closeCopyModal = () => {
    setCopyModal(null)
    if (copyModal === 'done') setSelectedTracks(new Set())
  }

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc')
    } else {
      setSortKey(key)
      setSortDir(key === 'title' || key === 'artist' || key === 'album' ? 'asc' : 'desc')
    }
  }

  const SortIcon = ({ column }: { column: SortKey }) => {
    if (sortKey !== column) return <ArrowUpDown className="h-3 w-3 opacity-30" />
    return sortDir === 'asc' ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />
  }

  const usbDevice = useAppStore(s => s.usbDevice)

  if (!ipod) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground">
        <div className="text-center">
          <IPodIcon size={64} className="mx-auto mb-3 opacity-30" />
          {usbDevice ? (
            <>
              <p className="font-medium text-foreground">{usbDevice.model} detected</p>
              {usbDevice.diskPath ? (
                <>
                  <p className="text-sm mt-1">This iPod is connected but not mounted.</p>
                  <div className="flex items-center gap-2 mt-3 justify-center">
                    <Button variant="outline" size="sm" onClick={() => MountIPod(usbDevice.diskPath).catch(e => console.error('mount failed:', e))}>
                      Mount
                    </Button>
                    {usbDevice.restorable && (
                      <Button variant="destructive" size="sm" onClick={() => setRestoreModalOpen(true)}>
                        <RotateCcw className="h-4 w-4 mr-1" /> Restore
                      </Button>
                    )}
                  </div>
                </>
              ) : (
                <div className="mt-3 max-w-xs space-y-4">
                  <p className="text-sm">Try disconnecting and reconnecting the iPod.</p>
                  {usbDevice.restorable && (
                    <div className="border-t pt-4">
                      <p className="text-sm text-muted-foreground">To restore, enter disk mode first:</p>
                      <ol className="text-sm text-muted-foreground mt-2 space-y-1 text-center">
                        <li>Hold <strong className="text-foreground">Menu + Select</strong> to reboot</li>
                        <li>Then hold <strong className="text-foreground">Select + Play</strong></li>
                        <li>Wait for the "OK to disconnect" screen</li>
                      </ol>
                    </div>
                  )}
                </div>
              )}
            </>
          ) : (
            <p>No iPod connected</p>
          )}
        </div>
      </div>
    )
  }

  const restorableFamilies = new Set(['iPod', 'iPod U2', 'iPod Photo', 'iPod Photo U2', 'iPod Mini', 'iPod Video', 'iPod Video U2', 'iPod Nano'])
  const isRestorable = restorableFamilies.has(ipod.family) && !['1st Gen', '2nd Gen', '3rd Gen'].includes(ipod.generation)
    && !(ipod.family === 'iPod Nano' && ipod.generation !== '1st Gen')

  const usedSpace = ipod.totalSpace - ipod.freeSpace
  const segments = [
    { label: 'Music', size: musicSize, color: 'bg-blue-500' },
    { label: 'Audiobooks', size: audiobookSize, color: 'bg-purple-500' },
    { label: 'Podcasts', size: podcastSize, color: 'bg-orange-500' },
    { label: 'Other', size: Math.max(0, otherSize), color: 'bg-muted-foreground/30' },
    { label: 'Free', size: ipod.freeSpace, color: 'bg-muted/50' },
  ]

  return (
    <div className="flex flex-col h-full">
      <div className="p-4 pb-0 border-b">
        <div className="flex items-center gap-4">
          <IPodIcon size={56} icon={ipod.icon} className="shrink-0" />
          <div className="flex-1 min-w-0">
            {editingName ? (
              <form onSubmit={async (e) => {
                e.preventDefault()
                if (nameInput.trim()) {
                  try {
                    await RenameIPod(nameInput.trim())
                    setIPod({ ...ipod, name: nameInput.trim() })
                  } catch {}
                }
                setEditingName(false)
              }} className="flex items-center gap-2">
                <Input
                  value={nameInput}
                  onChange={e => setNameInput(e.target.value)}
                  className="h-7 text-lg font-semibold w-48"
                  autoFocus
                  onBlur={() => setEditingName(false)}
                  onKeyDown={e => e.key === 'Escape' && setEditingName(false)}
                />
              </form>
            ) : (
              <div className="flex items-center gap-1.5 group">
                <h2 className="text-lg font-semibold">{ipod.name}</h2>
                <button
                  onClick={() => { setNameInput(ipod.name); setEditingName(true) }}
                  className="opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-foreground transition-opacity"
                >
                  <Pencil className="h-3.5 w-3.5" />
                </button>
              </div>
            )}
            <p className="text-sm text-muted-foreground">{[ipod.family, ipod.generation, ipod.displayCapacity].filter(Boolean).join(' \u00B7 ')} &middot; {formatBytes(usedSpace)} of {formatBytes(ipod.totalSpace)} used</p>
          </div>
          <div className="flex items-center gap-3 shrink-0">
            <div className="text-xs text-muted-foreground">
              {ipodTracks.length} items &middot; {totalHours}h of content
            </div>
            {isRestorable && (
              <Button variant="outline" size="sm" className="gap-1.5 text-xs h-7" onClick={() => setRestoreModalOpen(true)}>
                <RotateCcw className="h-3 w-3" /> Restore
              </Button>
            )}
          </div>
        </div>

        <div className="mt-3 flex rounded-full overflow-hidden h-3">
          {segments.map((seg, i) => (
            seg.size > 0 && (
              <div key={i} className={cn(seg.color, 'transition-all')}
                style={{ width: `${(seg.size / ipod.totalSpace) * 100}%` }}
                title={`${seg.label}: ${formatBytes(seg.size)}`} />
            )
          ))}
        </div>
        <div className="flex items-center gap-4 mt-2 flex-wrap">
          {segments.filter(s => s.size > 0).map((seg, i) => (
            <div key={i} className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <div className={cn('h-2 w-2 rounded-full', seg.color)} />
              <span>{seg.label}</span>
              <span className="font-medium">{formatBytes(seg.size)}</span>
            </div>
          ))}
        </div>

        <div className="flex gap-4 mt-3">
          {([
            { id: 'overview' as IPodTab, label: 'Overview' },
            { id: 'playlists' as IPodTab, label: `Playlists (${ipodPlaylists.length})` },
            { id: 'library' as IPodTab, label: `Library (${ipodTracks.length})` },
          ]).map(t => (
            <button key={t.id} onClick={() => setTab(t.id)}
              className={cn(
                'pb-2 text-sm font-medium border-b-2 transition-colors',
                tab === t.id ? 'border-foreground text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'
              )}>
              {t.label}
            </button>
          ))}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        {tab === 'overview' && (
          <div className="p-4 space-y-6">
            <div className="grid grid-cols-[1fr_auto_1fr] gap-6">
              <div>
                <div className="flex items-center gap-2 mb-2">
                  <Clock className="h-4 w-4 text-muted-foreground" />
                  <h3 className="text-sm font-medium">Recently Played</h3>
                </div>
                <div className="space-y-0.5">
                  {recentlyPlayed.map(t => (
                    <div key={t.id} className="flex items-center gap-2 px-2 py-1.5 rounded-md text-sm hover:bg-accent/30">
                      <TypeIcon type={t.type} />
                      <div className="flex-1 min-w-0">
                        <div className="truncate font-medium text-xs">{t.title}</div>
                        <div className="truncate text-xs text-muted-foreground">{t.artist}</div>
                      </div>
                      <TimeAgo ts={t.lastPlayed} className="text-[11px] text-muted-foreground shrink-0" />
                    </div>
                  ))}
                  {recentlyPlayed.length === 0 && (
                    <p className="text-xs text-muted-foreground px-2">No play history</p>
                  )}
                </div>
              </div>
              <div className="w-px bg-border" />
              <div>
                <div className="flex items-center gap-2 mb-2">
                  <TrendingUp className="h-4 w-4 text-muted-foreground" />
                  <h3 className="text-sm font-medium">Most Played</h3>
                </div>
                <div className="space-y-0.5">
                  {mostPlayed.map((t, i) => (
                    <div key={t.id} className="flex items-center gap-2 px-2 py-1.5 rounded-md text-sm hover:bg-accent/30">
                      <span className="text-xs text-muted-foreground w-4 text-right shrink-0">{i + 1}</span>
                      <div className="flex-1 min-w-0">
                        <div className="truncate font-medium text-xs">{t.title}</div>
                        <div className="truncate text-xs text-muted-foreground">{t.artist}</div>
                      </div>
                      <span className="text-[11px] text-muted-foreground shrink-0">{t.playCount} plays</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {tab === 'playlists' && (
          <div className="p-4 space-y-1">
            {ipodPlaylists.map(pl => {
              const tracks = pl.trackIds.map(id => ipodTracks.find(t => t.id === id)).filter(Boolean) as IPodTrack[]
              const isExpanded = expandedPlaylist === pl.id
              return (
                <div key={pl.id}>
                  <button onClick={() => setExpandedPlaylist(isExpanded ? null : pl.id)}
                    className="w-full flex items-center gap-2.5 px-3 py-2 rounded-md text-sm hover:bg-accent/30 transition-colors">
                    {isExpanded ? <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" /> : <ChevronUp className="h-3.5 w-3.5 text-muted-foreground rotate-90" />}
                    <ListMusic className="h-4 w-4 text-muted-foreground" />
                    <span className="font-medium">{pl.name}</span>
                    <span className="text-xs text-muted-foreground ml-auto">{tracks.length} tracks</span>
                  </button>
                  {isExpanded && (
                    <div className="ml-10 border-l border-border/50 space-y-0.5 my-1 pl-3">
                      {tracks.map((t, i) => (
                        <div key={t.id} className="flex items-center gap-2 px-2 py-1.5 rounded-md text-sm hover:bg-accent/30">
                          <span className="text-xs text-muted-foreground w-4 text-right shrink-0">{i + 1}</span>
                          <TypeIcon type={t.type} />
                          <div className="flex-1 min-w-0">
                            <span className="truncate text-xs font-medium">{t.title}</span>
                          </div>
                          <span className="text-xs text-muted-foreground shrink-0">{t.artist}</span>
                          <span className="text-xs text-muted-foreground shrink-0 w-12 text-right tabular-nums">{formatTrackDuration(t.duration)}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )
            })}
            {ipodPlaylists.length === 0 && (
              <div className="text-center py-8 text-muted-foreground">
                <ListMusic className="h-8 w-8 mx-auto mb-2 opacity-30" />
                <p className="text-sm">No playlists on this iPod</p>
              </div>
            )}
          </div>
        )}

        {tab === 'library' && (
          <div>
            {selectedTracks.size > 0 && (
              <div className="flex items-center justify-between px-4 py-2 border-b bg-primary/5">
                <div className="flex items-center gap-3">
                  <button onClick={() => setSelectedTracks(new Set())} className="text-muted-foreground hover:text-foreground">
                    <X className="h-4 w-4" />
                  </button>
                  <span className="text-sm font-medium">
                    {selectedTracks.size} item{selectedTracks.size !== 1 ? 's' : ''} selected
                  </span>
                  <span className="text-xs text-muted-foreground">({formatBytes(selectedSize)})</span>
                </div>
                <Button size="sm" className="gap-1.5" onClick={startCopyPicker}>
                  <Download className="h-3.5 w-3.5" /> Copy to Computer
                </Button>
              </div>
            )}

            <div className="flex items-center justify-between px-4 py-2 border-b bg-muted/30">
              <div className="flex items-center gap-1">
                {(['all', 'music', 'audiobook', 'podcast'] as TrackFilter[]).map(f => (
                  <Button key={f} variant={filter === f ? 'secondary' : 'ghost'} size="sm" className="text-xs h-7"
                    onClick={() => setFilter(f)}>
                    {f === 'all' ? 'All' : f === 'music' ? 'Music' : f === 'audiobook' ? 'Audiobooks' : 'Podcasts'}
                  </Button>
                ))}
              </div>
              <div className="flex items-center gap-2 ml-auto">
                <div className="relative w-56">
                  <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                  <Input placeholder="Filter tracks..." value={search} onChange={e => setSearch(e.target.value)} className="pl-8 pr-7 h-7 text-xs" />
                  {search && (
                    <button onClick={() => setSearch('')} className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground">
                      <X className="h-3.5 w-3.5" />
                    </button>
                  )}
                </div>
                <Button variant="outline" size="sm" className="gap-1.5 h-7 text-xs" onClick={() => {
                  setSelectedTracks(new Set(ipodTracks.map(t => t.id)))
                  startCopyPicker()
                }}>
                  <FolderDown className="h-3 w-3" /> Copy All
                </Button>
              </div>
            </div>

            <div className="text-xs">
              <div className="grid grid-cols-[28px_1fr_1fr_1fr_48px_64px_72px_80px_80px] gap-2 px-4 py-1.5 border-b bg-muted text-muted-foreground font-medium sticky top-0">
                <button onClick={toggleAllVisible}
                  className={cn('h-4 w-4 rounded border flex items-center justify-center shrink-0 transition-colors mt-0.5',
                    filteredTracks.length > 0 && filteredTracks.every(t => selectedTracks.has(t.id))
                      ? 'bg-primary border-primary text-primary-foreground' : 'border-input hover:border-foreground/50')}>
                  {filteredTracks.length > 0 && filteredTracks.every(t => selectedTracks.has(t.id)) && <Check className="h-3 w-3" />}
                </button>
                <button className="flex items-center gap-1 hover:text-foreground" onClick={() => handleSort('title')}>Title <SortIcon column="title" /></button>
                <button className="flex items-center gap-1 hover:text-foreground" onClick={() => handleSort('artist')}>Artist <SortIcon column="artist" /></button>
                <button className="flex items-center gap-1 hover:text-foreground" onClick={() => handleSort('album')}>Album <SortIcon column="album" /></button>
                <div>Format</div>
                <button className="flex items-center gap-1 justify-end hover:text-foreground" onClick={() => handleSort('playCount')}>Plays <SortIcon column="playCount" /></button>
                <button className="flex items-center gap-1 justify-end hover:text-foreground" onClick={() => handleSort('lastPlayed')}>Played <SortIcon column="lastPlayed" /></button>
                <button className="flex items-center gap-1 justify-end hover:text-foreground" onClick={() => handleSort('dateAdded')}>Added <SortIcon column="dateAdded" /></button>
                <div className="text-right">Duration</div>
              </div>

              {filteredTracks.map(t => (
                <div key={t.id} onClick={() => toggleTrack(t.id)}
                  className={cn(
                    "grid grid-cols-[28px_1fr_1fr_1fr_48px_64px_72px_80px_80px] gap-2 px-4 py-1.5 border-b border-border/50 hover:bg-accent/30 transition-colors items-center cursor-pointer",
                    selectedTracks.has(t.id) && "bg-primary/5"
                  )}>
                  <div className={cn('h-4 w-4 rounded border flex items-center justify-center shrink-0 transition-colors',
                    selectedTracks.has(t.id) ? 'bg-primary border-primary text-primary-foreground' : 'border-input')}>
                    {selectedTracks.has(t.id) && <Check className="h-3 w-3" />}
                  </div>
                  <div className="flex items-center gap-2 min-w-0">
                    <TypeIcon type={t.type} />
                    <span className="truncate">{t.title}</span>
                  </div>
                  <span className="truncate text-muted-foreground">{t.artist}</span>
                  <span className="truncate text-muted-foreground">{t.album}</span>
                  <span className="uppercase text-muted-foreground">{t.format}</span>
                  <span className="text-right tabular-nums">{t.playCount}</span>
                  <TimeAgo ts={t.lastPlayed} className="text-right text-muted-foreground" />
                  <TimeAgo ts={t.dateAdded} className="text-right text-muted-foreground" />
                  <span className="text-right text-muted-foreground tabular-nums">{formatTrackDuration(t.duration)}</span>
                </div>
              ))}

              <div className="px-4 py-2 text-muted-foreground border-b">
                {filteredTracks.length} item{filteredTracks.length !== 1 ? 's' : ''} &middot; {totalPlayCount} total plays
              </div>
            </div>
          </div>
        )}
      </div>

      {copyModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/50" onClick={() => copyModal !== 'copying' && closeCopyModal()} />
          <div className="relative bg-card rounded-xl shadow-xl border w-full max-w-md mx-4">
            {copyModal === 'pick' && (
              <div className="p-6 space-y-4">
                <div>
                  <h3 className="text-lg font-semibold">Copy to Computer</h3>
                  <p className="text-sm text-muted-foreground mt-1">
                    {selectedTracks.size} item{selectedTracks.size !== 1 ? 's' : ''} &middot; {formatBytes(selectedSize)}
                  </p>
                </div>
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-muted-foreground">Destination</label>
                  <div className="flex gap-2">
                    <Input value={copyDest} onChange={e => setCopyDest(e.target.value)} className="flex-1 text-sm" />
                    <Button variant="outline" size="icon" title="Browse..." onClick={browseDest}>
                      <Folder className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
                <div className="flex justify-end gap-2 pt-2">
                  <Button variant="outline" onClick={closeCopyModal}>Cancel</Button>
                  <Button onClick={startCopy} disabled={!copyDest}>
                    <Download className="h-4 w-4" /> Start Copy
                  </Button>
                </div>
              </div>
            )}

            {copyModal === 'copying' && (
              <div className="p-6 space-y-4">
                <div className="flex items-center gap-3">
                  <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                  <div>
                    <h3 className="text-sm font-semibold">Copying files...</h3>
                    <p className="text-xs text-muted-foreground">{copyProgress.current} of {copyProgress.total}</p>
                  </div>
                  <span className="ml-auto text-sm font-medium tabular-nums">
                    {copyProgress.total > 0 ? Math.round((copyProgress.current / copyProgress.total) * 100) : 0}%
                  </span>
                </div>
                <Progress value={copyProgress.total > 0 ? (copyProgress.current / copyProgress.total) * 100 : 0} />
                <p className="text-xs text-muted-foreground truncate">{copyProgress.currentFile}</p>
                <p className="text-[11px] text-muted-foreground truncate">{copyDest}</p>
              </div>
            )}

            {copyModal === 'done' && (
              <div className="p-6 space-y-4">
                <div className="flex items-center gap-3">
                  <CheckCircle className="h-5 w-5 text-blue-500" />
                  <div>
                    <h3 className="text-sm font-semibold">Copy complete</h3>
                    <p className="text-xs text-muted-foreground">{copyProgress.total} files copied to {copyDest}</p>
                  </div>
                </div>
                <div className="flex justify-end">
                  <Button onClick={closeCopyModal}>Done</Button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

function TypeIcon({ type }: { type: string }) {
  switch (type) {
    case 'music': return <Music className="h-3 w-3 text-blue-500 shrink-0" />
    case 'audiobook': return <BookOpen className="h-3 w-3 text-purple-500 shrink-0" />
    case 'podcast': return <Podcast className="h-3 w-3 text-orange-500 shrink-0" />
    default: return null
  }
}
