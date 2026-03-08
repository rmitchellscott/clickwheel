import { useAppStore } from '@/store/appStore'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import { X, CheckCircle, XCircle, Loader2, Info } from 'lucide-react'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { useState, useEffect, useRef, useCallback } from 'react'
import { applyTheme, type Theme } from '@/main'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { GetConfig, SaveSubsonicConfig, TestSubsonicConnection, SaveABSConfig, TestABSConnection, GetSyncSettings, SaveSyncSettings } from '../../wailsjs/go/main/App'
import type { SyncSettings } from '@/store/appStore'

function Toggle({ checked, onChange, disabled }: { checked: boolean; onChange: (v: boolean) => void; disabled?: boolean }) {
  return (
    <button
      onClick={() => !disabled && onChange(!checked)}
      className={cn(
        'relative inline-flex h-5 w-9 shrink-0 rounded-full border-2 border-transparent transition-colors',
        checked ? 'bg-primary' : 'bg-input',
        disabled && 'opacity-50 cursor-not-allowed'
      )}
    >
      <span className={cn(
        'pointer-events-none block h-4 w-4 rounded-full bg-background shadow-sm ring-0 transition-transform',
        checked ? 'translate-x-4' : 'translate-x-0'
      )} />
    </button>
  )
}

function SettingRow({ label, description, children }: { label: React.ReactNode; description?: string; children: React.ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-4 py-2">
      <div className="min-w-0">
        <div className="text-sm">{label}</div>
        {description && <div className="text-xs text-muted-foreground mt-0.5">{description}</div>}
      </div>
      <div className="shrink-0 pt-0.5">{children}</div>
    </div>
  )
}

type Section = 'general' | 'servers' | 'music' | 'books'

const sections: { key: Section; label: string }[] = [
  { key: 'general', label: 'General' },
  { key: 'servers', label: 'Servers' },
  { key: 'music', label: 'Music' },
  { key: 'books', label: 'Audiobooks & Podcasts' },
]

export function SettingsDialog() {
  const { settingsOpen, setSettingsOpen, setSubsonicConfigured, setSubsonicConnected, setAbsConfigured, setAbsConnected, ipod, activeDeviceId, knownDevices } = useAppStore()
  const activeDevice = knownDevices.find(d => d.deviceId === activeDeviceId)

  const [activeSection, setActiveSection] = useState<Section>('general')

  const [navURL, setNavURL] = useState('')
  const [navUser, setNavUser] = useState('')
  const [navPass, setNavPass] = useState('')
  const [navTesting, setNavTesting] = useState(false)
  const [navOk, setNavOk] = useState<boolean | null>(null)
  const [navError, setNavError] = useState('')

  const [absURL, setAbsURL] = useState('')
  const [absToken, setAbsToken] = useState('')
  const [absTesting, setAbsTesting] = useState(false)
  const [absOk, setAbsOk] = useState<boolean | null>(null)
  const [absError, setAbsError] = useState('')

  const [syncPlayCounts, setSyncPlayCounts] = useState(true)
  const [musicFormat, setMusicFormat] = useState('aac')
  const [musicBitRate, setMusicBitRate] = useState('256')
  const [syncBookPosition, setSyncBookPosition] = useState(true)
  const [twoWayBookSync, setTwoWayBookSync] = useState(false)
  const [splitBooks, setSplitBooks] = useState(true)
  const [splitHours, setSplitHours] = useState('8')
  const [syncPodcastPosition, setSyncPodcastPosition] = useState(true)
  const [twoWayPodcastSync, setTwoWayPodcastSync] = useState(false)

  const [theme, setThemeState] = useState<Theme>(() => {
    const stored = localStorage.getItem('theme') as Theme | null
    return stored === 'light' || stored === 'dark' ? stored : 'system'
  })

  const hydrated = useRef(false)
  const debounceTimer = useRef<ReturnType<typeof setTimeout>>()

  const setTheme = (t: Theme) => {
    setThemeState(t)
    localStorage.setItem('theme', t)
    applyTheme(t)
  }

  useEffect(() => {
    if (!settingsOpen) {
      hydrated.current = false
      return
    }
    GetConfig().then(cfg => {
      if (cfg.subsonic?.serverUrl) setNavURL(cfg.subsonic.serverUrl)
      if (cfg.subsonic?.username) setNavUser(cfg.subsonic.username)
      if (cfg.subsonic?.password) setNavPass(cfg.subsonic.password)
      if (cfg.abs?.serverUrl) setAbsURL(cfg.abs.serverUrl)
      if (cfg.abs?.token) setAbsToken(cfg.abs.token)
    })
    GetSyncSettings().then(s => {
      setSyncPlayCounts(s.syncPlayCounts)
      setMusicFormat(s.musicFormat || 'aac')
      setMusicBitRate(String(s.musicBitRate || 256))
      setSyncBookPosition(s.syncBookPosition)
      setTwoWayBookSync(s.twoWayBookSync)
      setSplitBooks(s.splitLongBooks)
      setSplitHours(String(s.splitHoursLimit || 8))
      setSyncPodcastPosition(s.syncPodcastPosition)
      setTwoWayPodcastSync(s.twoWayPodcastSync)
      hydrated.current = true
    })
  }, [settingsOpen])

  const saveSyncSettings = useCallback(() => {
    const settings: SyncSettings = {
      syncPlayCounts,
      syncBookPosition,
      twoWayBookSync,
      splitLongBooks: splitBooks,
      splitHoursLimit: parseInt(splitHours) || 8,
      musicFormat,
      musicBitRate: parseInt(musicBitRate) || 256,
      syncPodcastPosition,
      twoWayPodcastSync,
    }
    SaveSyncSettings(settings)
  }, [syncPlayCounts, syncBookPosition, twoWayBookSync, splitBooks, splitHours, musicFormat, musicBitRate, syncPodcastPosition, twoWayPodcastSync])

  useEffect(() => {
    if (!hydrated.current) return
    clearTimeout(debounceTimer.current)
    debounceTimer.current = setTimeout(saveSyncSettings, 300)
    return () => clearTimeout(debounceTimer.current)
  }, [saveSyncSettings])

  if (!settingsOpen) return null

  const testNav = async () => {
    setNavTesting(true)
    setNavError('')
    setNavOk(null)
    try {
      await SaveSubsonicConfig(navURL, navUser, navPass)
      await TestSubsonicConnection()
      setNavOk(true)
      setSubsonicConfigured(true)
      setSubsonicConnected(true)
    } catch (e) {
      setNavOk(false)
      setNavError(String(e))
    }
    setNavTesting(false)
  }

  const testAbs = async () => {
    setAbsTesting(true)
    setAbsError('')
    setAbsOk(null)
    try {
      await SaveABSConfig(absURL, absToken)
      await TestABSConnection()
      setAbsOk(true)
      setAbsConfigured(true)
      setAbsConnected(true)
    } catch (e) {
      setAbsOk(false)
      setAbsError(String(e))
    }
    setAbsTesting(false)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={() => setSettingsOpen(false)} />
      <div className="relative bg-card rounded-xl shadow-xl border w-full max-w-2xl mx-4 max-h-[85vh] flex flex-col">
        <div className="flex items-center justify-between p-4 border-b">
          <div>
            <h2 className="text-lg font-semibold">Settings</h2>
            {(ipod || activeDevice) && (
              <p className="text-sm text-muted-foreground">{ipod?.name ?? activeDevice?.name}</p>
            )}
          </div>
          <Button variant="ghost" size="icon" onClick={() => setSettingsOpen(false)}>
            <X className="h-4 w-4" />
          </Button>
        </div>

        <div className="flex h-[32rem]">
          <nav className="w-44 border-r p-2 shrink-0">
            {sections.map(s => (
              <button
                key={s.key}
                onClick={() => setActiveSection(s.key)}
                className={cn(
                  'w-full text-left text-sm px-3 py-1.5 rounded-md transition-colors',
                  activeSection === s.key
                    ? 'bg-accent font-medium'
                    : 'text-muted-foreground hover:bg-accent/50'
                )}
              >
                {s.label}
              </button>
            ))}
          </nav>

          <main className="flex-1 p-6 overflow-y-auto">
            {activeSection === 'general' && (
              <div className="space-y-1">
                <SettingRow label="Theme" description="Choose light, dark, or follow your system setting">
                  <Select value={theme} onValueChange={v => setTheme(v as Theme)}>
                    <SelectTrigger className="w-[140px] h-8 text-sm">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="system">System</SelectItem>
                      <SelectItem value="light">Light</SelectItem>
                      <SelectItem value="dark">Dark</SelectItem>
                    </SelectContent>
                  </Select>
                </SettingRow>
              </div>
            )}

            {activeSection === 'servers' && (
              <div className="space-y-6">
                <section className="space-y-3">
                  <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wider">Subsonic</h3>
                  <div className="space-y-3">
                    <div className="space-y-1">
                      <label className="text-xs font-medium text-muted-foreground">Server URL</label>
                      <Input placeholder="https://music.example.com" value={navURL} onChange={e => setNavURL(e.target.value)} />
                    </div>
                    <div className="space-y-1">
                      <label className="text-xs font-medium text-muted-foreground">Username</label>
                      <Input placeholder="Username" value={navUser} onChange={e => setNavUser(e.target.value)} />
                    </div>
                    <div className="space-y-1">
                      <label className="text-xs font-medium text-muted-foreground">Password</label>
                      <Input placeholder="Password" type="password" value={navPass} onChange={e => setNavPass(e.target.value)} />
                    </div>
                    <div className="flex items-center gap-2">
                      <Button size="sm" variant="outline" onClick={testNav} disabled={navTesting}>
                        {navTesting && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
                        Test & Save
                      </Button>
                      {navOk === true && !navTesting && <CheckCircle className="h-4 w-4 text-blue-500" />}
                      {navOk === false && !navTesting && (
                        <span className="text-sm text-destructive flex items-center gap-1"><XCircle className="h-4 w-4" />{navError}</span>
                      )}
                    </div>
                  </div>
                </section>

                <div className="border-t" />

                <section className="space-y-3">
                  <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wider">Audiobookshelf</h3>
                  <div className="space-y-3">
                    <div className="space-y-1">
                      <label className="text-xs font-medium text-muted-foreground">Server URL</label>
                      <Input placeholder="https://abs.example.com" value={absURL} onChange={e => setAbsURL(e.target.value)} />
                    </div>
                    <div className="space-y-1">
                      <label className="text-xs font-medium text-muted-foreground">API Token</label>
                      <Input placeholder="API Token" type="password" value={absToken} onChange={e => setAbsToken(e.target.value)} />
                    </div>
                    <div className="flex items-center gap-2">
                      <Button size="sm" variant="outline" onClick={testAbs} disabled={absTesting}>
                        {absTesting && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
                        Test & Save
                      </Button>
                      {absOk === true && !absTesting && <CheckCircle className="h-4 w-4 text-blue-500" />}
                      {absOk === false && !absTesting && (
                        <span className="text-sm text-destructive flex items-center gap-1"><XCircle className="h-4 w-4" />{absError}</span>
                      )}
                    </div>
                  </div>
                </section>
              </div>
            )}

            {activeSection === 'music' && (
              <div className="space-y-1">
                <SettingRow
                  label="Transcode format"
                  description="Format to transcode music to when streaming from your server"
                >
                  <Select value={musicFormat} onValueChange={setMusicFormat}>
                    <SelectTrigger className="w-[140px] h-8 text-sm">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="aac">AAC (.m4a)</SelectItem>
                      <SelectItem value="alac">ALAC (.m4a)</SelectItem>
                      <SelectItem value="mp3">MP3</SelectItem>
                      <SelectItem value="raw">Original</SelectItem>
                    </SelectContent>
                  </Select>
                </SettingRow>
                {musicFormat !== 'raw' && musicFormat !== 'alac' && (
                  <SettingRow
                    label="Max bit rate"
                    description="Maximum bit rate for transcoded files (kbps)"
                  >
                    <Select value={musicBitRate} onValueChange={setMusicBitRate}>
                      <SelectTrigger className="w-[110px] h-8 text-sm">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="128">128 kbps</SelectItem>
                        <SelectItem value="192">192 kbps</SelectItem>
                        <SelectItem value="256">256 kbps</SelectItem>
                        <SelectItem value="320">320 kbps</SelectItem>
                      </SelectContent>
                    </Select>
                  </SettingRow>
                )}
                <SettingRow
                  label="Sync play counts"
                  description="Write play counts from iPod back to your server after each sync"
                >
                  <Toggle checked={syncPlayCounts} onChange={setSyncPlayCounts} />
                </SettingRow>
              </div>
            )}

            {activeSection === 'books' && (
              <div className="space-y-6">
                <section className="space-y-1">
                  <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wider mb-2">Audiobooks</h3>
                  <SettingRow
                    label="Sync playback position"
                    description="Update iPod bookmarks from Audiobookshelf progress on each sync"
                  >
                    <Toggle checked={syncBookPosition} onChange={setSyncBookPosition} />
                  </SettingRow>
                  <SettingRow
                    label={
                      <span className="flex items-center gap-1.5">
                        Two-way sync
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Info className="h-3.5 w-3.5 text-muted-foreground/50 cursor-help" />
                            </TooltipTrigger>
                            <TooltipContent side="right" className="max-w-64">
                              When both devices have progress, the most recent position wins. Listening on the iPod between syncs will update your Audiobookshelf server automatically.
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </span>
                    }
                    description="Also push iPod listening progress back to Audiobookshelf"
                  >
                    <Toggle checked={twoWayBookSync} onChange={setTwoWayBookSync} disabled={!syncBookPosition} />
                  </SettingRow>
                  <SettingRow
                    label="Split long audiobooks"
                    description={splitBooks
                      ? undefined
                      : 'Long single-file audiobooks can cause instability on older iPods'}
                  >
                    <Toggle checked={splitBooks} onChange={setSplitBooks} />
                  </SettingRow>
                  {splitBooks && (
                    <div className="flex items-start justify-between gap-4 py-2 pl-6">
                      <div className="min-w-0">
                        <div className="text-sm">Split threshold</div>
                        <div className="text-xs text-muted-foreground mt-0.5">Split audiobooks longer than this into parts</div>
                      </div>
                      <div className="shrink-0 pt-0.5">
                        <Select value={splitHours} onValueChange={setSplitHours}>
                          <SelectTrigger className="w-[100px] h-8 text-sm">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="4">4 hours</SelectItem>
                            <SelectItem value="6">6 hours</SelectItem>
                            <SelectItem value="8">8 hours</SelectItem>
                            <SelectItem value="12">12 hours</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </div>
                  )}
                </section>

                <div className="border-t" />

                <section className="space-y-1">
                  <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wider mb-2">Podcasts</h3>
                  <SettingRow
                    label="Sync playback position"
                    description="Update iPod bookmarks from Audiobookshelf podcast progress on each sync"
                  >
                    <Toggle checked={syncPodcastPosition} onChange={setSyncPodcastPosition} />
                  </SettingRow>
                  <SettingRow
                    label={
                      <span className="flex items-center gap-1.5">
                        Two-way sync
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Info className="h-3.5 w-3.5 text-muted-foreground/50 cursor-help" />
                            </TooltipTrigger>
                            <TooltipContent side="right" className="max-w-64">
                              When both devices have progress, the furthest position wins. Listening on the iPod between syncs will update your Audiobookshelf server automatically.
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </span>
                    }
                    description="Also push iPod listening progress back to Audiobookshelf"
                  >
                    <Toggle checked={twoWayPodcastSync} onChange={setTwoWayPodcastSync} disabled={!syncPodcastPosition} />
                  </SettingRow>
                </section>
              </div>
            )}
          </main>
        </div>
      </div>
    </div>
  )
}
