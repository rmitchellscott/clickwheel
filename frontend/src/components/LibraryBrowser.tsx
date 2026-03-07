import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { useAppStore } from '@/store/appStore'
import { GetNavidromePlaylists, GetNavidromeAlbums, SetExclusions } from '../../wailsjs/go/main/App'

export function LibraryBrowser() {
  const { playlists, setPlaylists, albums, setAlbums, exclusions, setExclusions, navidromeConnected } = useAppStore()
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    if (navidromeConnected && !loaded) {
      Promise.all([GetNavidromePlaylists(), GetNavidromeAlbums()]).then(([pl, al]) => {
        setPlaylists(pl || [])
        setAlbums(al || [])
        setLoaded(true)
      })
    }
  }, [navidromeConnected, loaded, setPlaylists, setAlbums])

  if (!navidromeConnected) {
    return <div className="p-4 text-muted-foreground">Connect to Navidrome in Setup first.</div>
  }

  const togglePlaylist = (id: string) => {
    const next = exclusions.playlists.includes(id)
      ? exclusions.playlists.filter(x => x !== id)
      : [...exclusions.playlists, id]
    const updated = { ...exclusions, playlists: next }
    setExclusions(updated)
    SetExclusions(updated)
  }

  const toggleAlbum = (id: string) => {
    const next = exclusions.albums.includes(id)
      ? exclusions.albums.filter(x => x !== id)
      : [...exclusions.albums, id]
    const updated = { ...exclusions, albums: next }
    setExclusions(updated)
    SetExclusions(updated)
  }

  const allPlaylistsSelected = playlists.every(p => !exclusions.playlists.includes(p.id))
  const toggleAllPlaylists = () => {
    const updated = {
      ...exclusions,
      playlists: allPlaylistsSelected ? playlists.map(p => p.id) : []
    }
    setExclusions(updated)
    SetExclusions(updated)
  }

  return (
    <div className="p-4 space-y-4">
      <section>
        <div className="flex items-center justify-between mb-2">
          <h2 className="text-lg font-semibold">Playlists</h2>
          <Button variant="ghost" size="sm" onClick={toggleAllPlaylists}>
            {allPlaylistsSelected ? 'Deselect All' : 'Select All'}
          </Button>
        </div>
        <div className="space-y-1">
          {playlists.map(pl => (
            <label key={pl.id} className="flex items-center gap-2 text-sm p-1 rounded hover:bg-accent">
              <input
                type="checkbox"
                checked={!exclusions.playlists.includes(pl.id)}
                onChange={() => togglePlaylist(pl.id)}
                className="rounded"
              />
              <span>{pl.name}</span>
              <span className="text-muted-foreground ml-auto">{pl.songCount} songs</span>
            </label>
          ))}
        </div>
      </section>

      <section>
        <h2 className="text-lg font-semibold mb-2">Albums</h2>
        <div className="space-y-1 max-h-[400px] overflow-y-auto">
          {albums.map(al => (
            <label key={al.id} className="flex items-center gap-2 text-sm p-1 rounded hover:bg-accent">
              <input
                type="checkbox"
                checked={!exclusions.albums.includes(al.id)}
                onChange={() => toggleAlbum(al.id)}
                className="rounded"
              />
              <span>{al.name}</span>
              <span className="text-muted-foreground">— {al.artist}</span>
              <span className="text-muted-foreground ml-auto">{al.year || ''}</span>
            </label>
          ))}
        </div>
      </section>
    </div>
  )
}
