import { create } from 'zustand'

export interface IPodInfo {
  mountPoint: string
  name: string
  freeSpace: number
  totalSpace: number
  family: string
  generation: string
  capacity: string
  color: string
  model: string
  icon: string
  displayCapacity: string
}

export interface Playlist {
  id: string
  name: string
  songCount: number
}

export interface Album {
  id: string
  name: string
  artist: string
  songCount: number
  year: number
}

export interface Artist {
  id: string
  name: string
  albumCount: number
  songCount?: number
}

export interface Book {
  id: string
  title: string
  author: string
  duration: number
  size: number
}

export interface Podcast {
  id: string
  title: string
  author: string
  episodeCount: number
  size: number
}

export interface SyncProgress {
  phase: string
  current: number
  total: number
  message: string
  percent: number
  eta?: string
}

export interface IPodTrack {
  id: string
  title: string
  artist: string
  album: string
  genre: string
  duration: number
  playCount: number
  lastPlayed: number
  dateAdded: number
  size: number
  type: 'music' | 'audiobook' | 'podcast'
}

export interface IPodPlaylist {
  id: string
  name: string
  trackIds: string[]
}

export interface SyncSettings {
  syncPlayCounts: boolean
  syncBookPosition: boolean
  twoWayBookSync: boolean
  splitLongBooks: boolean
  splitHoursLimit: number
  musicFormat: string
  musicBitRate: number
  syncPodcastPosition: boolean
  twoWayPodcastSync: boolean
}

export interface SyncPlanSummary {
  addTracks: { title: string; artist: string; size: number }[]
  addBooks: { title: string; artist: string; size: number }[]
  addPodcasts: { title: string; artist: string; size: number }[]
  removeTracks: { title: string; artist: string; size: number }[]
  removeBooks: { title: string; artist: string; size: number }[]
  removePodcasts: { title: string; artist: string; size: number }[]
  playlists: string[]
  playlistsChanged: string[]
  playsToSync: number
  booksToIPod: string[]
  booksFromIPod: string[]
  podcastsToIPod: string[]
  podcastsFromIPod: string[]
}

export interface KnownDevice {
  deviceId: string
  name: string
  family?: string
  generation?: string
  capacity?: string
  color?: string
  model?: string
  icon?: string
}

type Page = 'ipod' | 'library' | 'books' | 'podcasts' | 'sync'

interface AppState {
  page: Page
  setPage: (page: Page) => void

  ipod: IPodInfo | null
  setIPod: (ipod: IPodInfo | null) => void

  activeDeviceId: string | null
  setActiveDeviceId: (id: string | null) => void
  knownDevices: KnownDevice[]
  setKnownDevices: (devices: KnownDevice[]) => void
  ipodTracks: IPodTrack[]
  setIPodTracks: (tracks: IPodTrack[]) => void
  ipodPlaylists: IPodPlaylist[]
  setIPodPlaylists: (playlists: IPodPlaylist[]) => void

  subsonicConfigured: boolean
  setSubsonicConfigured: (c: boolean) => void
  subsonicConnected: boolean
  setSubsonicConnected: (c: boolean) => void
  absConfigured: boolean
  setAbsConfigured: (c: boolean) => void
  absConnected: boolean
  setAbsConnected: (c: boolean) => void

  playlists: Playlist[]
  setPlaylists: (playlists: Playlist[]) => void
  albums: Album[]
  setAlbums: (albums: Album[]) => void
  artists: Artist[]
  setArtists: (artists: Artist[]) => void
  books: Book[]
  setBooks: (books: Book[]) => void

  includedPlaylists: Set<string>
  togglePlaylist: (id: string) => void
  toggleAllPlaylists: () => void

  includedAlbums: Set<string>
  toggleAlbum: (id: string) => void
  toggleAllAlbums: () => void

  includedArtists: Set<string>
  toggleArtist: (id: string) => void
  toggleAllArtists: () => void

  includedBooks: Set<string>
  toggleBook: (id: string) => void
  toggleAllBooks: () => void

  podcasts: Podcast[]
  setPodcasts: (podcasts: Podcast[]) => void
  includedPodcasts: Set<string>
  togglePodcast: (id: string) => void
  toggleAllPodcasts: () => void

  syncPlan: SyncPlanSummary | null
  setSyncPlan: (plan: SyncPlanSummary | null) => void
  syncPlanLoading: boolean
  setSyncPlanLoading: (loading: boolean) => void

  syncing: boolean
  setSyncing: (s: boolean) => void
  syncProgress: SyncProgress | null
  setSyncProgress: (p: SyncProgress | null) => void
  syncError: string | null
  setSyncError: (e: string | null) => void
  syncComplete: boolean
  setSyncComplete: (c: boolean) => void

  settingsOpen: boolean
  setSettingsOpen: (open: boolean) => void

  searchQuery: string
  setSearchQuery: (q: string) => void

  timezone: string
  setTimezone: (tz: string) => void
}

function toggleInSet(set: Set<string>, id: string): Set<string> {
  const next = new Set(set)
  next.has(id) ? next.delete(id) : next.add(id)
  return next
}

export const useAppStore = create<AppState>((set) => ({
  page: 'library',
  setPage: (page) => set({ page, searchQuery: '' }),

  ipod: null,
  setIPod: (ipod) => set({ ipod }),

  activeDeviceId: null,
  setActiveDeviceId: (activeDeviceId) => set({ activeDeviceId }),
  knownDevices: [],
  setKnownDevices: (knownDevices) => set({ knownDevices }),

  ipodTracks: [],
  setIPodTracks: (ipodTracks) => set({ ipodTracks }),
  ipodPlaylists: [],
  setIPodPlaylists: (ipodPlaylists) => set({ ipodPlaylists }),

  subsonicConfigured: false,
  setSubsonicConfigured: (subsonicConfigured) => set({ subsonicConfigured }),
  subsonicConnected: false,
  setSubsonicConnected: (subsonicConnected) => set({ subsonicConnected }),
  absConfigured: false,
  setAbsConfigured: (absConfigured) => set({ absConfigured }),
  absConnected: false,
  setAbsConnected: (absConnected) => set({ absConnected }),

  playlists: [],
  setPlaylists: (playlists) => set({ playlists }),
  albums: [],
  setAlbums: (albums) => set({ albums }),
  artists: [],
  setArtists: (artists) => set({ artists }),
  books: [],
  setBooks: (books) => set({ books }),

  includedPlaylists: new Set(),
  togglePlaylist: (id) => set(state => ({ includedPlaylists: toggleInSet(state.includedPlaylists, id) })),
  toggleAllPlaylists: () => set(state => {
    const allIncluded = state.playlists.length > 0 && state.playlists.every(p => state.includedPlaylists.has(p.id))
    return { includedPlaylists: allIncluded ? new Set() : new Set(state.playlists.map(p => p.id)) }
  }),

  includedAlbums: new Set(),
  toggleAlbum: (id) => set(state => ({ includedAlbums: toggleInSet(state.includedAlbums, id) })),
  toggleAllAlbums: () => set(state => {
    const allIncluded = state.albums.length > 0 && state.albums.every(a => state.includedAlbums.has(a.id))
    return { includedAlbums: allIncluded ? new Set() : new Set(state.albums.map(a => a.id)) }
  }),

  includedArtists: new Set(),
  toggleArtist: (id) => set(state => ({ includedArtists: toggleInSet(state.includedArtists, id) })),
  toggleAllArtists: () => set(state => {
    const allIncluded = state.artists.length > 0 && state.artists.every(a => state.includedArtists.has(a.id))
    return { includedArtists: allIncluded ? new Set() : new Set(state.artists.map(a => a.id)) }
  }),

  includedBooks: new Set(),
  toggleBook: (id) => set(state => ({ includedBooks: toggleInSet(state.includedBooks, id) })),
  toggleAllBooks: () => set(state => {
    const allIncluded = state.books.length > 0 && state.books.every(b => state.includedBooks.has(b.id))
    return { includedBooks: allIncluded ? new Set() : new Set(state.books.map(b => b.id)) }
  }),

  podcasts: [],
  setPodcasts: (podcasts) => set({ podcasts }),
  includedPodcasts: new Set(),
  togglePodcast: (id) => set(state => ({ includedPodcasts: toggleInSet(state.includedPodcasts, id) })),
  toggleAllPodcasts: () => set(state => {
    const allIncluded = state.podcasts.length > 0 && state.podcasts.every(p => state.includedPodcasts.has(p.id))
    return { includedPodcasts: allIncluded ? new Set() : new Set(state.podcasts.map(p => p.id)) }
  }),

  syncPlan: null,
  setSyncPlan: (syncPlan) => set({ syncPlan }),
  syncPlanLoading: false,
  setSyncPlanLoading: (syncPlanLoading) => set({ syncPlanLoading }),

  syncing: false,
  setSyncing: (syncing) => set({ syncing }),
  syncProgress: null,
  setSyncProgress: (syncProgress) => set({ syncProgress }),
  syncError: null,
  setSyncError: (syncError) => set({ syncError }),
  syncComplete: false,
  setSyncComplete: (syncComplete) => set({ syncComplete }),

  settingsOpen: false,
  setSettingsOpen: (settingsOpen) => set({ settingsOpen }),

  searchQuery: '',
  setSearchQuery: (searchQuery) => set({ searchQuery }),

  timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
  setTimezone: (timezone) => set({ timezone }),
}))
