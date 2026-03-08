import { useEffect } from 'react'
import { useAppStore } from '@/store/appStore'
import { Sidebar } from '@/components/Sidebar'
import { IPodPage } from '@/components/IPodPage'
import { LibraryPage } from '@/components/LibraryPage'
import { BooksPage } from '@/components/BooksPage'
import { PodcastsPage } from '@/components/PodcastsPage'
import { SyncPage } from '@/components/SyncPage'
import { SyncProgressCard } from '@/components/SyncProgressCard'
import { SettingsDialog } from '@/components/SettingsDialog'
import { TooltipProvider } from '@/components/ui/tooltip'
import { GetConfig, GetExclusions, GetTimezone, TestSubsonicConnection, TestABSConnection, DetectIPod, GetSubsonicPlaylists, GetSubsonicAlbums, GetSubsonicArtists, GetABSLibraries, GetABSBooks, GetABSPodcasts } from '../wailsjs/go/main/App'
import { useSyncEvents } from '@/hooks/useSyncEvents'

function App() {
  const { page, setIPod, setSubsonicConfigured, setSubsonicConnected, setAbsConfigured, setAbsConnected, setPlaylists, setAlbums, setArtists, setBooks, setPodcasts } = useAppStore()

  useSyncEvents()

  useEffect(() => {
    GetConfig().then(async (cfg) => {
      const navConfigured = !!(cfg.subsonic?.serverUrl)
      const absConfigured = !!(cfg.abs?.serverUrl)
      setSubsonicConfigured(navConfigured)
      setAbsConfigured(absConfigured)

      if (navConfigured) {
        try {
          await TestSubsonicConnection()
          setSubsonicConnected(true)
          const [pl, al, ar] = await Promise.all([
            GetSubsonicPlaylists(),
            GetSubsonicAlbums(),
            GetSubsonicArtists(),
          ])
          const playlists = pl || []
          const albums = al || []
          const artists = ar || []
          setPlaylists(playlists)
          setAlbums(albums)
          setArtists(artists)

          GetExclusions().then(exc => {
            if (!exc) return
            const exPlaylists = exc.playlists || []
            const exAlbums = exc.albums || []
            const exArtists = exc.artists || []
            const exBooks = exc.books || []
            useAppStore.setState({
              selectedPlaylists: new Set(playlists.filter(p => !exPlaylists.includes(p.id)).map(p => p.id)),
              selectedAlbums: new Set(albums.filter(a => !exAlbums.includes(a.id)).map(a => a.id)),
              selectedArtists: new Set(artists.filter(a => !exArtists.includes(a.id)).map(a => a.id)),
            })

            if (absConfigured) {
              GetABSLibraries().then(libs => {
                if (!libs || libs.length === 0) return
                const exPodcasts = exc.podcasts || []

                for (const lib of libs) {
                  if (lib.mediaType === 'podcast') {
                    GetABSPodcasts(lib.id).then(pods => {
                      const podcasts = (pods || []).map(p => ({
                        id: p.id,
                        title: p.media?.metadata?.title || 'Unknown',
                        author: p.media?.metadata?.author || 'Unknown',
                        episodeCount: p.media?.numEpisodes || 0,
                        size: p.media?.size || 0,
                      }))
                      setPodcasts(podcasts)
                      useAppStore.setState({
                        selectedPodcasts: new Set(podcasts.filter(p => !exPodcasts.includes(p.id)).map(p => p.id)),
                      })
                    })
                  } else {
                    GetABSBooks(lib.id).then(bks => {
                      const books = (bks || []).map(b => ({
                        id: b.id,
                        title: b.media?.metadata?.title || 'Unknown',
                        author: b.media?.metadata?.authorName || 'Unknown',
                        duration: b.media?.duration || 0,
                        size: b.size || 0,
                      }))
                      setBooks(books)
                      useAppStore.setState({
                        selectedBooks: new Set(books.filter(b => !exBooks.includes(b.id)).map(b => b.id)),
                      })
                    })
                  }
                }
              })
            }
          })
        } catch {
          setSubsonicConnected(false)
        }
      }

      if (absConfigured) {
        try {
          await TestABSConnection()
          setAbsConnected(true)
        } catch {
          setAbsConnected(false)
        }
      }
    })

    GetTimezone().then(tz => {
      if (tz) useAppStore.getState().setTimezone(tz)
    }).catch(() => {})

    DetectIPod().then(info => {
      if (info) setIPod(info)
    }).catch(() => {})
  }, [])

  useEffect(() => {
    const interval = setInterval(async () => {
      if (useAppStore.getState().syncing) return
      try {
        const info = await DetectIPod()
        const current = useAppStore.getState().ipod
        if (info && !current) {
          setIPod(info)
        } else if (!info && current) {
          setIPod(null)
        }
      } catch {
        if (useAppStore.getState().ipod) setIPod(null)
      }
    }, 10000)
    return () => clearInterval(interval)
  }, [])

  return (
    <TooltipProvider>
      <div className="h-screen flex">
        <Sidebar />
        <main className="flex-1 overflow-hidden relative">
          {page === 'ipod' && <IPodPage />}
          {page === 'library' && <LibraryPage />}
          {page === 'books' && <BooksPage />}
          {page === 'podcasts' && <PodcastsPage />}
          {page === 'sync' && <SyncPage />}
          <SyncProgressCard />
        </main>
        <SettingsDialog />
      </div>
    </TooltipProvider>
  )
}

export default App
