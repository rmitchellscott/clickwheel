import { useAppStore } from '@/store/appStore'
import { cn } from '@/lib/utils'
import { formatBytes } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Search, Check, Podcast, X, ChevronUp, ChevronDown, ArrowUpDown } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { GetABSLibraries, GetABSPodcasts, SetInclusions, GetInclusions } from '../../wailsjs/go/main/App'

type SortKey = 'title' | 'author' | 'episodes' | 'size'
type SortDir = 'asc' | 'desc'

function SortHeader({ label, active, dir, onClick, className }: {
  label: string; active: boolean; dir: SortDir; onClick: () => void; className?: string
}) {
  return (
    <button onClick={onClick} className={cn('flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors', className)}>
      {label}
      {active
        ? (dir === 'asc' ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />)
        : <ArrowUpDown className="h-3 w-3 opacity-30" />
      }
    </button>
  )
}

export function PodcastsPage() {
  const {
    absConfigured, absConnected, setSettingsOpen,
    podcasts, setPodcasts, searchQuery, setSearchQuery,
    includedPodcasts, togglePodcast, toggleAllPodcasts,
  } = useAppStore()

  const [sortKey, setSortKey] = useState<SortKey>('title')
  const [sortDir, setSortDir] = useState<SortDir>('asc')

  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    if (absConnected && !loaded) {
      GetABSLibraries().then(libs => {
        if (!libs) return
        const podLibs = libs.filter(l => l.mediaType === 'podcast')
        if (podLibs.length === 0) return
        GetABSPodcasts(podLibs[0].id).then(pods => {
          const podList = (pods || []).map(p => ({
            id: p.id,
            title: p.media?.metadata?.title || 'Unknown',
            author: p.media?.metadata?.author || 'Unknown',
            episodeCount: p.media?.numEpisodes || 0,
            size: p.media?.size || 0,
          }))
          setPodcasts(podList)
          setLoaded(true)

          GetInclusions().then(inc => {
            useAppStore.setState({
              includedPodcasts: new Set(inc?.podcasts || []),
            })
          })
        })
      })
    }
  }, [absConnected, loaded])

  useEffect(() => {
    if (!absConnected || podcasts.length === 0) return
    const state = useAppStore.getState()
    GetInclusions().then(inc => {
      SetInclusions({
        playlists: inc?.playlists || [],
        albums: inc?.albums || [],
        artists: inc?.artists || [],
        books: inc?.books || [],
        podcasts: [...state.includedPodcasts],
      })
    })
  }, [includedPodcasts])

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc')
    } else {
      setSortKey(key)
      setSortDir(key === 'title' || key === 'author' ? 'asc' : 'desc')
    }
  }

  const filteredPodcasts = useMemo(() => {
    let items = podcasts
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      items = items.filter(p =>
        p.title.toLowerCase().includes(q) || p.author.toLowerCase().includes(q)
      )
    }
    return [...items].sort((a, b) => {
      let cmp = 0
      switch (sortKey) {
        case 'title': cmp = a.title.localeCompare(b.title); break
        case 'author': cmp = a.author.localeCompare(b.author); break
        case 'episodes': cmp = a.episodeCount - b.episodeCount; break
        case 'size': cmp = a.size - b.size; break
      }
      return sortDir === 'asc' ? cmp : -cmp
    })
  }, [podcasts, searchQuery, sortKey, sortDir])

  const allIncluded = podcasts.length > 0 && podcasts.every(p => includedPodcasts.has(p.id))
  const selectedSize = podcasts
    .filter(p => includedPodcasts.has(p.id))
    .reduce((acc, p) => acc + p.size, 0)
  const selectedEpisodes = podcasts
    .filter(p => includedPodcasts.has(p.id))
    .reduce((acc, p) => acc + p.episodeCount, 0)

  if (!absConfigured) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center max-w-xs">
          <Podcast className="h-10 w-10 mx-auto mb-3 text-muted-foreground/30" />
          <h2 className="text-lg font-semibold mb-1">Connect to Audiobookshelf</h2>
          <p className="text-sm text-muted-foreground mb-4">Add your Audiobookshelf server in settings to browse and sync your podcasts.</p>
          <Button onClick={() => setSettingsOpen(true)}>Open Settings</Button>
        </div>
      </div>
    )
  }

  if (podcasts.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center max-w-xs">
          <Podcast className="h-10 w-10 mx-auto mb-3 text-muted-foreground/30" />
          <h2 className="text-lg font-semibold mb-1">No Podcasts</h2>
          <p className="text-sm text-muted-foreground mb-4">Podcasts will appear here once your Audiobookshelf podcast libraries are connected.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between p-4 pb-3 border-b">
        <div>
          <h2 className="text-lg font-semibold">Podcasts</h2>
          <p className="text-sm text-muted-foreground">
            {includedPodcasts.size} show{includedPodcasts.size !== 1 ? 's' : ''} &middot; {selectedEpisodes} episodes &middot; {formatBytes(selectedSize)}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <div className="relative w-64">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search podcasts..."
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
              className="pl-9 pr-8"
            />
            {searchQuery && (
              <button onClick={() => setSearchQuery('')} className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground">
                <X className="h-4 w-4" />
              </button>
            )}
          </div>
          <Button variant="ghost" size="sm" className="w-24" onClick={toggleAllPodcasts}>
            {allIncluded ? 'Deselect All' : 'Select All'}
          </Button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="flex items-center gap-3 px-5 py-2 border-b bg-muted/30">
          <div className="w-4" />
          <div className="w-4" />
          <SortHeader label="Title" active={sortKey === 'title'} dir={sortDir} className="flex-1"
            onClick={() => handleSort('title')} />
          <SortHeader label="Episodes" active={sortKey === 'episodes'} dir={sortDir} className="w-32 justify-end"
            onClick={() => handleSort('episodes')} />
          <SortHeader label="Size" active={sortKey === 'size'} dir={sortDir} className="w-16 justify-end"
            onClick={() => handleSort('size')} />
        </div>

        <div className="p-2">
          <div className="grid gap-1">
            {filteredPodcasts.map(podcast => (
              <button
                key={podcast.id}
                onClick={() => togglePodcast(podcast.id)}
                className={cn(
                  'w-full flex items-center gap-3 px-3 py-3 rounded-lg text-sm transition-colors text-left',
                  includedPodcasts.has(podcast.id) ? 'bg-accent/70' : 'hover:bg-accent/30'
                )}
              >
                <div className={cn(
                  'h-4 w-4 rounded border flex items-center justify-center shrink-0 transition-colors',
                  includedPodcasts.has(podcast.id) ? 'bg-primary border-primary text-primary-foreground' : 'border-input'
                )}>
                  {includedPodcasts.has(podcast.id) && <Check className="h-3 w-3" />}
                </div>
                <Podcast className="h-4 w-4 text-muted-foreground shrink-0" />
                <div className="flex-1 min-w-0">
                  <div className="font-medium truncate">{podcast.title}</div>
                  <div className="text-xs text-muted-foreground">{podcast.author}</div>
                </div>
                <div className="flex items-center gap-4 shrink-0 text-xs text-muted-foreground">
                  <span>{podcast.episodeCount} episode{podcast.episodeCount !== 1 ? 's' : ''}</span>
                  <span className="w-16 text-right">{formatBytes(podcast.size)}</span>
                </div>
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
