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
import { GetConfig, GetInclusions, GetTimezone, TestSubsonicConnection, TestABSConnection, DetectIPod, DetectUSBIPods, GetSubsonicPlaylists, GetSubsonicAlbums, GetSubsonicArtists, GetABSLibraries, GetABSBooks, GetABSPodcasts, GetActiveDeviceID, GetKnownDevices } from '../wailsjs/go/main/App'
import { useSyncEvents } from '@/hooks/useSyncEvents'
import { useRestoreEvents } from '@/hooks/useRestoreEvents'
import { RestoreDialog } from '@/components/RestoreDialog'

function App() {
  const { page, setIPod, setSubsonicConfigured, setSubsonicConnected, setAbsConfigured, setAbsConnected, setPlaylists, setAlbums, setArtists, setBooks, setPodcasts } = useAppStore()

  useSyncEvents()
  useRestoreEvents()

  useEffect(() => {
    GetActiveDeviceID().then(id => {
      if (id) useAppStore.getState().setActiveDeviceId(id)
    }).catch(() => {})

    GetKnownDevices().then(devices => {
      if (devices) useAppStore.getState().setKnownDevices(devices)
    }).catch(() => {})

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
          setPlaylists(pl || [])
          setAlbums(al || [])
          setArtists(ar || [])
        } catch {
          setSubsonicConnected(false)
        }
      }

      if (absConfigured) {
        try {
          await TestABSConnection()
          setAbsConnected(true)
          const libs = await GetABSLibraries()
          if (libs && libs.length > 0) {
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
                })
              }
            }
          }
        } catch {
          setAbsConnected(false)
        }
      }

      GetInclusions().then(inc => {
        if (!inc) return
        useAppStore.setState({
          includedPlaylists: new Set(inc.playlists || []),
          includedAlbums: new Set(inc.albums || []),
          includedArtists: new Set(inc.artists || []),
          includedBooks: new Set(inc.books || []),
          includedPodcasts: new Set(inc.podcasts || []),
        })
      })
    })

    GetTimezone().then(tz => {
      if (tz) useAppStore.getState().setTimezone(tz)
    }).catch(() => {})

    DetectIPod().then(info => {
      if (info) {
        setIPod(info)
      } else {
        DetectUSBIPods().then(ipods => {
          if (ipods && ipods.length > 0 && ipods[0].model) {
            const dev = { model: ipods[0].model!.Name, generation: ipods[0].model!.Family + ' ' + ipods[0].model!.Generation, productId: 0, mode: ipods[0].mode, restorable: ipods[0].model!.Restorable ?? false }
            useAppStore.getState().setUSBDevice(dev)
            if (dev.restorable) {
              useAppStore.getState().setRestoreModalOpen(true)
            }
          }
        }).catch(() => {})
      }
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
          GetActiveDeviceID().then(id => {
            if (id) useAppStore.getState().setActiveDeviceId(id)
          }).catch(() => {})
          GetKnownDevices().then(devices => {
            if (devices) useAppStore.getState().setKnownDevices(devices)
          }).catch(() => {})
          GetInclusions().then(inc => {
            if (!inc) return
            useAppStore.setState({
              includedPlaylists: new Set(inc.playlists || []),
              includedAlbums: new Set(inc.albums || []),
              includedArtists: new Set(inc.artists || []),
              includedBooks: new Set(inc.books || []),
              includedPodcasts: new Set(inc.podcasts || []),
            })
          }).catch(() => {})
        } else if (!info && current) {
          setIPod(null)
        }
        if (!info) {
          const ipods = await DetectUSBIPods().catch(() => null)
          if (ipods && ipods.length > 0 && ipods[0].model) {
            const prev = useAppStore.getState().usbDevice
            const dev = { model: ipods[0].model!.Name, generation: ipods[0].model!.Family + ' ' + ipods[0].model!.Generation, productId: 0, mode: ipods[0].mode, restorable: ipods[0].model!.Restorable ?? false }
            useAppStore.getState().setUSBDevice(dev)
            if (dev.restorable && !prev) {
              useAppStore.getState().setRestoreModalOpen(true)
            }
          } else {
            useAppStore.getState().setUSBDevice(null)
          }
        } else {
          useAppStore.getState().setUSBDevice(null)
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
        <RestoreDialog />
      </div>
    </TooltipProvider>
  )
}

export default App
