import { useAppStore } from '@/store/appStore'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Search, Music, ListMusic, ChevronDown, ChevronRight, ChevronUp, Check, X, User, ArrowUpDown } from 'lucide-react'
import { useState, useMemo, useEffect } from 'react'
import { GetSubsonicPlaylists, GetSubsonicAlbums, GetSubsonicArtists, SetInclusions, GetInclusions } from '../../wailsjs/go/main/App'

type Section = 'playlists' | 'artists' | 'albums'
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

function nextSort<K extends string>(key: K, currentKey: K, currentDir: SortDir, defaultDir: SortDir = 'asc'): { key: K; dir: SortDir } {
  if (currentKey === key) return { key, dir: currentDir === 'asc' ? 'desc' : 'asc' }
  return { key, dir: defaultDir }
}

export function LibraryPage() {
  const {
    subsonicConfigured, subsonicConnected, setSettingsOpen,
    playlists, setPlaylists, artists, setArtists, albums, setAlbums,
    searchQuery, setSearchQuery,
    includedPlaylists, togglePlaylist, toggleAllPlaylists,
    includedArtists, toggleArtist, toggleAllArtists,
    includedAlbums, toggleAlbum, toggleAllAlbums,
  } = useAppStore()

  const [expandedSection, setExpandedSection] = useState<Section | null>('playlists')
  const [loaded, setLoaded] = useState(false)

  const [plSortKey, setPlSortKey] = useState<'name' | 'songCount'>('name')
  const [plSortDir, setPlSortDir] = useState<SortDir>('asc')

  const [arSortKey, setArSortKey] = useState<'name' | 'albumCount'>('name')
  const [arSortDir, setArSortDir] = useState<SortDir>('asc')

  const [alSortKey, setAlSortKey] = useState<'name' | 'artist' | 'songCount' | 'year'>('name')
  const [alSortDir, setAlSortDir] = useState<SortDir>('asc')

  useEffect(() => {
    if (subsonicConnected && !loaded) {
      Promise.all([GetSubsonicPlaylists(), GetSubsonicAlbums(), GetSubsonicArtists()]).then(([pl, al, ar]) => {
        const pls = pl || []
        const als = al || []
        const ars = ar || []
        setPlaylists(pls)
        setAlbums(als)
        setArtists(ars)
        setLoaded(true)

        GetInclusions().then(inc => {
          if (!inc) return
          useAppStore.setState({
            includedPlaylists: new Set(inc.playlists || []),
            includedAlbums: new Set(inc.albums || []),
            includedArtists: new Set(inc.artists || []),
          })
        })
      })
    }
  }, [subsonicConnected, loaded])

  useEffect(() => {
    if (!subsonicConnected) return
    const state = useAppStore.getState()
    GetInclusions().then(inc => {
      SetInclusions({
        playlists: [...state.includedPlaylists],
        albums: [...state.includedAlbums],
        artists: [...state.includedArtists],
        books: inc?.books || [],
        podcasts: inc?.podcasts || [],
      })
    })
  }, [includedPlaylists, includedAlbums, includedArtists])

  const filteredPlaylists = useMemo(() => {
    let items = playlists
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      items = items.filter(p => p.name.toLowerCase().includes(q))
    }
    return [...items].sort((a, b) => {
      const cmp = plSortKey === 'name' ? a.name.localeCompare(b.name) : a.songCount - b.songCount
      return plSortDir === 'asc' ? cmp : -cmp
    })
  }, [playlists, searchQuery, plSortKey, plSortDir])

  const filteredArtists = useMemo(() => {
    let items = artists
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      items = items.filter(a => a.name.toLowerCase().includes(q))
    }
    return [...items].sort((a, b) => {
      const cmp = arSortKey === 'name' ? a.name.localeCompare(b.name) : a.albumCount - b.albumCount
      return arSortDir === 'asc' ? cmp : -cmp
    })
  }, [artists, searchQuery, arSortKey, arSortDir])

  const filteredAlbums = useMemo(() => {
    let items = albums
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      items = items.filter(a => a.name.toLowerCase().includes(q) || a.artist.toLowerCase().includes(q))
    }
    return [...items].sort((a, b) => {
      let cmp = 0
      switch (alSortKey) {
        case 'name': cmp = a.name.localeCompare(b.name); break
        case 'artist': cmp = a.artist.localeCompare(b.artist); break
        case 'songCount': cmp = a.songCount - b.songCount; break
        case 'year': cmp = a.year - b.year; break
      }
      return alSortDir === 'asc' ? cmp : -cmp
    })
  }, [albums, searchQuery, alSortKey, alSortDir])

  const allPlaylistsIncluded = playlists.length > 0 && playlists.every(p => includedPlaylists.has(p.id))
  const allArtistsIncluded = artists.length > 0 && artists.every(a => includedArtists.has(a.id))
  const allAlbumsIncluded = albums.length > 0 && albums.every(a => includedAlbums.has(a.id))

  const toggleSection = (s: Section) => {
    setExpandedSection(expandedSection === s ? null : s)
  }

  if (!subsonicConfigured) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center max-w-xs">
          <Music className="h-10 w-10 mx-auto mb-3 text-muted-foreground/30" />
          <h2 className="text-lg font-semibold mb-1">Connect to Subsonic</h2>
          <p className="text-sm text-muted-foreground mb-4">Add your Subsonic-compatible server in settings to browse and sync your music library.</p>
          <Button onClick={() => setSettingsOpen(true)}>Open Settings</Button>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between p-4 pb-3 border-b">
        <div>
          <h2 className="text-lg font-semibold">Music Library</h2>
          <p className="text-sm text-muted-foreground">
            {includedPlaylists.size} playlist{includedPlaylists.size !== 1 ? 's' : ''}, {includedArtists.size} artist{includedArtists.size !== 1 ? 's' : ''}, {includedAlbums.size} album{includedAlbums.size !== 1 ? 's' : ''} selected
          </p>
        </div>
        <div className="relative w-64">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search music..."
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
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="border-b">
          <button
            onClick={() => toggleSection('playlists')}
            className="w-full flex items-center gap-2 px-4 py-3 hover:bg-accent/50 transition-colors"
          >
            {expandedSection === 'playlists' ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            <ListMusic className="h-4 w-4 text-muted-foreground" />
            <span className="font-medium text-sm">Playlists</span>
            <Badge variant="secondary" className="ml-2">{filteredPlaylists.length}</Badge>
            <div className="ml-auto">
              <Button variant="ghost" size="sm" className="text-xs h-7 w-24"
                onClick={(e) => { e.stopPropagation(); toggleAllPlaylists() }}>
                {allPlaylistsIncluded ? 'Deselect All' : 'Select All'}
              </Button>
            </div>
          </button>
          {expandedSection === 'playlists' && (
            <div className="pb-2">
              <div className="flex items-center gap-3 px-5 py-1 mb-1">
                <div className="w-4" />
                <SortHeader label="Name" active={plSortKey === 'name'} dir={plSortDir} className="flex-1"
                  onClick={() => { const s = nextSort('name', plSortKey, plSortDir); setPlSortKey(s.key); setPlSortDir(s.dir) }} />
                <SortHeader label="Songs" active={plSortKey === 'songCount'} dir={plSortDir}
                  onClick={() => { const s = nextSort('songCount', plSortKey, plSortDir, 'desc'); setPlSortKey(s.key); setPlSortDir(s.dir) }} />
              </div>
              <div className="px-2">
                {filteredPlaylists.map(pl => (
                  <button key={pl.id} onClick={() => togglePlaylist(pl.id)}
                    className={cn('w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors',
                      includedPlaylists.has(pl.id) ? 'bg-accent/70' : 'hover:bg-accent/30')}>
                    <div className={cn('h-4 w-4 rounded border flex items-center justify-center shrink-0 transition-colors',
                      includedPlaylists.has(pl.id) ? 'bg-primary border-primary text-primary-foreground' : 'border-input')}>
                      {includedPlaylists.has(pl.id) && <Check className="h-3 w-3" />}
                    </div>
                    <span className="truncate">{pl.name}</span>
                    <span className="ml-auto text-xs text-muted-foreground">{pl.songCount} songs</span>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>

        <div className="border-b">
          <button
            onClick={() => toggleSection('artists')}
            className="w-full flex items-center gap-2 px-4 py-3 hover:bg-accent/50 transition-colors"
          >
            {expandedSection === 'artists' ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            <User className="h-4 w-4 text-muted-foreground" />
            <span className="font-medium text-sm">Artists</span>
            <Badge variant="secondary" className="ml-2">{filteredArtists.length}</Badge>
            <div className="ml-auto">
              <Button variant="ghost" size="sm" className="text-xs h-7 w-24"
                onClick={(e) => { e.stopPropagation(); toggleAllArtists() }}>
                {allArtistsIncluded ? 'Deselect All' : 'Select All'}
              </Button>
            </div>
          </button>
          {expandedSection === 'artists' && (
            <div className="pb-2">
              <div className="flex items-center gap-3 px-5 py-1 mb-1">
                <div className="w-4" />
                <SortHeader label="Name" active={arSortKey === 'name'} dir={arSortDir} className="flex-1"
                  onClick={() => { const s = nextSort('name', arSortKey, arSortDir); setArSortKey(s.key); setArSortDir(s.dir) }} />
                <SortHeader label="Albums" active={arSortKey === 'albumCount'} dir={arSortDir}
                  onClick={() => { const s = nextSort('albumCount', arSortKey, arSortDir, 'desc'); setArSortKey(s.key); setArSortDir(s.dir) }} />
              </div>
              <div className="px-2">
                {filteredArtists.map(ar => (
                  <button key={ar.id} onClick={() => toggleArtist(ar.id)}
                    className={cn('w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors',
                      includedArtists.has(ar.id) ? 'bg-accent/70' : 'hover:bg-accent/30')}>
                    <div className={cn('h-4 w-4 rounded border flex items-center justify-center shrink-0 transition-colors',
                      includedArtists.has(ar.id) ? 'bg-primary border-primary text-primary-foreground' : 'border-input')}>
                      {includedArtists.has(ar.id) && <Check className="h-3 w-3" />}
                    </div>
                    <span className="truncate">{ar.name}</span>
                    <div className="ml-auto flex items-center gap-3 shrink-0">
                      <span className="text-xs text-muted-foreground">{ar.albumCount} album{ar.albumCount !== 1 ? 's' : ''}</span>
                      {(ar.songCount ?? 0) > 0 && <span className="text-xs text-muted-foreground">{ar.songCount} songs</span>}
                    </div>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>

        <div className="border-b">
          <button
            onClick={() => toggleSection('albums')}
            className="w-full flex items-center gap-2 px-4 py-3 hover:bg-accent/50 transition-colors"
          >
            {expandedSection === 'albums' ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            <Music className="h-4 w-4 text-muted-foreground" />
            <span className="font-medium text-sm">Albums</span>
            <Badge variant="secondary" className="ml-2">{filteredAlbums.length}</Badge>
            <div className="ml-auto">
              <Button variant="ghost" size="sm" className="text-xs h-7 w-24"
                onClick={(e) => { e.stopPropagation(); toggleAllAlbums() }}>
                {allAlbumsIncluded ? 'Deselect All' : 'Select All'}
              </Button>
            </div>
          </button>
          {expandedSection === 'albums' && (
            <div className="pb-2">
              <div className="flex items-center gap-3 px-5 py-1 mb-1">
                <div className="w-4" />
                <SortHeader label="Name" active={alSortKey === 'name'} dir={alSortDir} className="flex-1"
                  onClick={() => { const s = nextSort('name', alSortKey, alSortDir); setAlSortKey(s.key); setAlSortDir(s.dir) }} />
                <SortHeader label="Artist" active={alSortKey === 'artist'} dir={alSortDir}
                  onClick={() => { const s = nextSort('artist', alSortKey, alSortDir); setAlSortKey(s.key); setAlSortDir(s.dir) }} />
                <SortHeader label="Tracks" active={alSortKey === 'songCount'} dir={alSortDir}
                  onClick={() => { const s = nextSort('songCount', alSortKey, alSortDir, 'desc'); setAlSortKey(s.key); setAlSortDir(s.dir) }} />
                <SortHeader label="Year" active={alSortKey === 'year'} dir={alSortDir}
                  onClick={() => { const s = nextSort('year', alSortKey, alSortDir, 'desc'); setAlSortKey(s.key); setAlSortDir(s.dir) }} />
              </div>
              <div className="px-2">
                {filteredAlbums.map(al => (
                  <button key={al.id} onClick={() => toggleAlbum(al.id)}
                    className={cn('w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors',
                      includedAlbums.has(al.id) ? 'bg-accent/70' : 'hover:bg-accent/30')}>
                    <div className={cn('h-4 w-4 rounded border flex items-center justify-center shrink-0 transition-colors',
                      includedAlbums.has(al.id) ? 'bg-primary border-primary text-primary-foreground' : 'border-input')}>
                      {includedAlbums.has(al.id) && <Check className="h-3 w-3" />}
                    </div>
                    <div className="flex flex-col items-start min-w-0">
                      <span className="truncate w-full text-left">{al.name}</span>
                      <span className="text-xs text-muted-foreground">{al.artist}</span>
                    </div>
                    <div className="ml-auto flex items-center gap-3 shrink-0">
                      <span className="text-xs text-muted-foreground">{al.songCount} tracks</span>
                      <span className="text-xs text-muted-foreground w-10 text-right">{al.year}</span>
                    </div>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
