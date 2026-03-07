import { create } from 'zustand'

interface IPodInfo {
  mountPoint: string
  name: string
  freeSpace: number
  totalSpace: number
}

interface Playlist {
  id: string
  name: string
  songCount: number
}

interface Album {
  id: string
  name: string
  artist: string
  songCount: number
  year: number
}

interface Library {
  id: string
  name: string
}

interface Book {
  id: string
  media: {
    metadata: {
      title: string
      authorName: string
    }
    duration: number
  }
}

interface SyncProgress {
  phase: string
  current: number
  total: number
  message: string
  percent: number
}

interface Exclusions {
  playlists: string[]
  albums: string[]
  books: string[]
}

type Tab = 'setup' | 'music' | 'books' | 'sync'

interface AppState {
  tab: Tab
  setTab: (tab: Tab) => void

  ipod: IPodInfo | null
  setIPod: (ipod: IPodInfo | null) => void

  navidromeConnected: boolean
  setNavidromeConnected: (connected: boolean) => void

  absConnected: boolean
  setABSConnected: (connected: boolean) => void

  playlists: Playlist[]
  setPlaylists: (playlists: Playlist[]) => void

  albums: Album[]
  setAlbums: (albums: Album[]) => void

  libraries: Library[]
  setLibraries: (libraries: Library[]) => void

  books: Book[]
  setBooks: (books: Book[]) => void

  exclusions: Exclusions
  setExclusions: (exclusions: Exclusions) => void

  syncing: boolean
  setSyncing: (syncing: boolean) => void

  syncProgress: SyncProgress | null
  setSyncProgress: (progress: SyncProgress | null) => void

  syncError: string | null
  setSyncError: (error: string | null) => void
}

export const useAppStore = create<AppState>((set) => ({
  tab: 'setup',
  setTab: (tab) => set({ tab }),

  ipod: null,
  setIPod: (ipod) => set({ ipod }),

  navidromeConnected: false,
  setNavidromeConnected: (navidromeConnected) => set({ navidromeConnected }),

  absConnected: false,
  setABSConnected: (absConnected) => set({ absConnected }),

  playlists: [],
  setPlaylists: (playlists) => set({ playlists }),

  albums: [],
  setAlbums: (albums) => set({ albums }),

  libraries: [],
  setLibraries: (libraries) => set({ libraries }),

  books: [],
  setBooks: (books) => set({ books }),

  exclusions: { playlists: [], albums: [], books: [] },
  setExclusions: (exclusions) => set({ exclusions }),

  syncing: false,
  setSyncing: (syncing) => set({ syncing }),

  syncProgress: null,
  setSyncProgress: (syncProgress) => set({ syncProgress }),

  syncError: null,
  setSyncError: (syncError) => set({ syncError }),
}))
