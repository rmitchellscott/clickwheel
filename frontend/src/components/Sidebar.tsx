import { useAppStore } from '@/store/appStore'
import { formatBytes } from '@/lib/utils'
import { cn } from '@/lib/utils'
import { Music, BookOpen, Podcast, RefreshCw, Settings, Circle, Loader2, ChevronDown } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { IPodIcon } from '@/components/IPodIcon'

function EjectIcon({ className }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" className={className}>
      <polygon points="12 5 5 15 19 15" />
      <line x1="5" y1="19" x2="19" y2="19" />
    </svg>
  )
}
import { useState } from 'react'
import { DetectIPod, EjectIPod, TestSubsonicConnection, TestABSConnection, SwitchDevice, GetKnownDevices, GetActiveDeviceID, GetInclusions } from '../../wailsjs/go/main/App'

const navItems = [
  { id: 'library' as const, label: 'Music', icon: Music },
  { id: 'books' as const, label: 'Audiobooks', icon: BookOpen },
  { id: 'podcasts' as const, label: 'Podcasts', icon: Podcast },
  { id: 'sync' as const, label: 'Sync', icon: RefreshCw },
]

export function Sidebar() {
  const {
    page, setPage, ipod, setIPod, syncing,
    subsonicConfigured, subsonicConnected, setSubsonicConnected,
    absConfigured, absConnected, setAbsConnected,
    settingsOpen, setSettingsOpen,
    syncPlan,
    activeDeviceId, knownDevices,
  } = useAppStore()

  const [scanningIPod, setScanningIPod] = useState(false)
  const [ejecting, setEjecting] = useState(false)
  const [reconnectingNav, setReconnectingNav] = useState(false)
  const [reconnectingAbs, setReconnectingAbs] = useState(false)
  const [switcherOpen, setSwitcherOpen] = useState(false)

  const activeKnownDevice = knownDevices.find(d => d.deviceId === activeDeviceId)

  const scanForIPod = async () => {
    if (scanningIPod) return
    setScanningIPod(true)
    try {
      const info = await DetectIPod()
      setIPod(info)
    } catch {
      setIPod(null)
    }
    setScanningIPod(false)
  }

  const ejectIPod = async (e: React.MouseEvent) => {
    e.stopPropagation()
    if (ejecting || syncing) return
    setEjecting(true)
    try {
      await EjectIPod()
      setIPod(null)
      if (page === 'ipod') setPage('library')
    } catch {}
    setEjecting(false)
  }

  const reconnectNav = async () => {
    if (reconnectingNav) return
    setReconnectingNav(true)
    try {
      await TestSubsonicConnection()
      setSubsonicConnected(true)
    } catch {
      setSubsonicConnected(false)
    }
    setReconnectingNav(false)
  }

  const switchDevice = async (deviceId: string) => {
    try {
      await SwitchDevice(deviceId)
      const [id, devices] = await Promise.all([GetActiveDeviceID(), GetKnownDevices()])
      if (id) useAppStore.getState().setActiveDeviceId(id)
      if (devices) useAppStore.getState().setKnownDevices(devices)
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
    } catch {}
    setSwitcherOpen(false)
  }

  const reconnectAbs = async () => {
    if (reconnectingAbs) return
    setReconnectingAbs(true)
    try {
      await TestABSConnection()
      setAbsConnected(true)
    } catch {
      setAbsConnected(false)
    }
    setReconnectingAbs(false)
  }

  const usedSpace = ipod ? ipod.totalSpace - ipod.freeSpace : 0
  const usedPercent = ipod ? ((usedSpace / ipod.totalSpace) * 100) : 0

  const estimatedNew = syncPlan
    ? (syncPlan.addTracks || []).reduce((a, t) => a + (t.size || 0), 0)
      + (syncPlan.addBooks || []).reduce((a, b) => a + (b.size || 0), 0)
    : 0
  const projectedPercent = ipod ? Math.min(100, ((usedSpace + estimatedNew) / ipod.totalSpace) * 100) : 0
  const willFit = ipod ? (usedSpace + estimatedNew) <= ipod.totalSpace : true
  const projectedFree = ipod ? ipod.totalSpace - usedSpace - estimatedNew : 0

  return (
    <aside className="w-56 bg-sidebar text-sidebar-foreground border-r border-sidebar-border flex flex-col h-full shrink-0">
      {ipod && (
        <button
          onClick={() => setPage('ipod')}
          className={cn(
            'mx-3 mt-3 mb-3 p-3 rounded-lg text-left transition-colors',
            page === 'ipod'
              ? 'bg-sidebar-accent ring-1 ring-sidebar-accent-foreground/20'
              : 'bg-sidebar-accent hover:ring-1 hover:ring-sidebar-accent-foreground/10'
          )}
        >
          <div className="flex items-center gap-2.5 mb-2">
            <IPodIcon size={36} icon={ipod.icon} className="shrink-0" />
            <div className="min-w-0 flex-1">
              <div className="flex items-center min-w-0">
                <span className="text-sm font-medium text-sidebar-accent-foreground truncate min-w-0">{ipod.name}</span>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      onClick={ejectIPod}
                      disabled={ejecting || syncing}
                      className="shrink-0 ml-auto h-6 w-6 p-0 text-sidebar-foreground/40 hover:text-sidebar-foreground/70 hover:bg-sidebar-border/50"
                    >
                      {ejecting
                        ? <Loader2 className="h-3.5 w-3.5 animate-spin" />
                        : <EjectIcon className="h-3.5 w-3.5" />
                      }
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="right">
                    {syncing ? 'Cannot eject during sync' : 'Eject iPod'}
                  </TooltipContent>
                </Tooltip>
              </div>
              <div className="text-[11px] text-sidebar-foreground/60">{[ipod.family, ipod.generation].filter(Boolean).join(' \u00B7 ')}</div>
              {ipod.displayCapacity && <div className="text-[11px] text-sidebar-foreground/60">{ipod.displayCapacity}</div>}
            </div>
          </div>
          <div className="relative h-1.5 rounded-full bg-sidebar-border overflow-hidden">
            {estimatedNew > 0 && (
              <div
                className={cn(
                  "absolute inset-y-0 left-0 rounded-full transition-all",
                  willFit ? "bg-blue-400" : "bg-destructive/60"
                )}
                style={{ width: `${projectedPercent}%` }}
              />
            )}
            <div
              className="absolute inset-y-0 left-0 rounded-full bg-sidebar-accent-foreground/60 transition-all"
              style={{ width: `${usedPercent}%` }}
            />
          </div>
          <div className="flex justify-between items-center mt-1.5">
            <span className="text-[11px] text-sidebar-foreground/70">{formatBytes(usedSpace)} used</span>
            {estimatedNew > 0 ? (
              <span className={cn("text-[11px]", willFit ? "text-sidebar-foreground/70" : "text-destructive")}>
                {willFit ? `${formatBytes(projectedFree)} free after` : "Won't fit"}
              </span>
            ) : (
              <span className="text-[11px] text-sidebar-foreground/70">{formatBytes(ipod.freeSpace)} free</span>
            )}
          </div>
        </button>
      )}

      {!ipod && activeKnownDevice && (
        <div className="mx-3 mt-3 mb-3 p-3 rounded-lg bg-sidebar-accent opacity-70">
          <div className="flex items-center gap-2.5 mb-1">
            <IPodIcon size={36} icon={activeKnownDevice.icon} className="shrink-0 opacity-60" />
            <div className="min-w-0 flex-1">
              <div className="flex items-center min-w-0">
                <span className="text-sm font-medium text-sidebar-accent-foreground truncate min-w-0">
                  {activeKnownDevice.name || 'iPod'}
                </span>
                {knownDevices.length > 1 && (
                  <Button
                    variant="ghost"
                    onClick={() => setSwitcherOpen(!switcherOpen)}
                    className="shrink-0 ml-auto h-6 w-6 p-0 text-sidebar-foreground/40 hover:text-sidebar-foreground/70 hover:bg-sidebar-border/50"
                  >
                    <ChevronDown className={cn("h-3.5 w-3.5 transition-transform", switcherOpen && "rotate-180")} />
                  </Button>
                )}
              </div>
              <div className="text-[11px] text-sidebar-foreground/60">
                {[activeKnownDevice.family, activeKnownDevice.generation].filter(Boolean).join(' \u00B7 ')}
              </div>
              {activeKnownDevice.capacity && (
                <div className="text-[11px] text-sidebar-foreground/60">{activeKnownDevice.capacity}</div>
              )}
            </div>
          </div>
          {switcherOpen && knownDevices.length > 1 && (
            <div className="mt-1.5 pt-1.5 border-t border-sidebar-border/50 space-y-0.5">
              {knownDevices.map(d => (
                <button
                  key={d.deviceId}
                  onClick={() => switchDevice(d.deviceId)}
                  className={cn(
                    'w-full flex items-center gap-2 px-1.5 py-1 rounded text-xs text-left hover:bg-sidebar-border/50 transition-colors',
                    d.deviceId === activeDeviceId && 'bg-sidebar-border/50 font-medium'
                  )}
                >
                  <IPodIcon size={16} icon={d.icon} className="shrink-0" />
                  <span className="truncate">{d.name || 'iPod'}</span>
                </button>
              ))}
            </div>
          )}
          <div className="flex items-center justify-between mt-1">
            <span className="text-[11px] text-sidebar-foreground/50">Offline</span>
            <button
              onClick={scanForIPod}
              disabled={scanningIPod}
              className="text-[10px] text-sidebar-foreground/40 hover:text-sidebar-foreground/60 transition-colors"
            >
              {scanningIPod ? 'Scanning...' : 'Scan'}
            </button>
          </div>
        </div>
      )}

      {!ipod && !activeKnownDevice && (
        <button
          onClick={scanForIPod}
          disabled={scanningIPod}
          className="mx-3 mt-3 mb-3 p-3 rounded-lg bg-sidebar-accent text-center hover:ring-1 hover:ring-sidebar-accent-foreground/10 transition-colors"
        >
          {scanningIPod ? (
            <Loader2 className="h-5 w-5 mx-auto mb-1 animate-spin text-sidebar-foreground/40" />
          ) : (
            <IPodIcon size={32} className="mx-auto mb-1 opacity-30" />
          )}
          <p className="text-xs text-sidebar-foreground/50">
            {scanningIPod ? 'Scanning...' : 'No iPod connected'}
          </p>
          {!scanningIPod && (
            <p className="text-[10px] text-sidebar-foreground/35 mt-0.5">Click to scan</p>
          )}
        </button>
      )}

      <nav className="flex-1 px-2 space-y-0.5">
        {navItems.map(item => (
          <button
            key={item.id}
            onClick={() => setPage(item.id)}
            className={cn(
              'w-full flex items-center gap-2.5 px-3 py-2 rounded-md text-sm transition-colors',
              page === item.id
                ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium'
                : 'text-sidebar-foreground hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground'
            )}
          >
            <item.icon className="h-4 w-4" />
            {item.label}
          </button>
        ))}
      </nav>

      <div className="p-3 border-t border-sidebar-border space-y-2">
        {subsonicConfigured && (
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                onClick={reconnectNav}
                disabled={reconnectingNav}
                className="w-full flex items-center gap-2 px-1 rounded hover:bg-sidebar-accent/50 transition-colors py-0.5"
              >
                {reconnectingNav
                  ? <Loader2 className="h-2.5 w-2.5 animate-spin text-sidebar-foreground/50" />
                  : <Circle className={cn('h-2 w-2 fill-current', subsonicConnected ? 'text-blue-400' : 'text-red-500')} />
                }
                <span className="text-xs text-sidebar-foreground/70">Subsonic</span>
                {!subsonicConnected && !reconnectingNav && (
                  <RefreshCw className="h-2.5 w-2.5 ml-auto text-sidebar-foreground/40" />
                )}
              </button>
            </TooltipTrigger>
            <TooltipContent side="right">
              {reconnectingNav ? 'Reconnecting...' : subsonicConnected ? 'Connected' : 'Reconnect'}
            </TooltipContent>
          </Tooltip>
        )}
        {absConfigured && (
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                onClick={reconnectAbs}
                disabled={reconnectingAbs}
                className="w-full flex items-center gap-2 px-1 rounded hover:bg-sidebar-accent/50 transition-colors py-0.5"
              >
                {reconnectingAbs
                  ? <Loader2 className="h-2.5 w-2.5 animate-spin text-sidebar-foreground/50" />
                  : <Circle className={cn('h-2 w-2 fill-current', absConnected ? 'text-blue-400' : 'text-red-500')} />
                }
                <span className="text-xs text-sidebar-foreground/70">Audiobookshelf</span>
                {!absConnected && !reconnectingAbs && (
                  <RefreshCw className="h-2.5 w-2.5 ml-auto text-sidebar-foreground/40" />
                )}
              </button>
            </TooltipTrigger>
            <TooltipContent side="right">
              {reconnectingAbs ? 'Reconnecting...' : absConnected ? 'Connected' : 'Reconnect'}
            </TooltipContent>
          </Tooltip>
        )}
        <button
          onClick={() => setSettingsOpen(!settingsOpen)}
          className="w-full flex items-center gap-2.5 px-2 py-1.5 rounded-md text-xs text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground transition-colors mt-1"
        >
          <Settings className="h-3.5 w-3.5" />
          Settings
        </button>
      </div>
    </aside>
  )
}
