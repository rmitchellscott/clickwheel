import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAppStore } from '@/store/appStore'
import { SaveNavidromeConfig, TestNavidromeConnection, SaveABSConfig, TestABSConnection, DetectIPod, GetConfig, GetExclusions } from '../../wailsjs/go/main/App'
import { CheckCircle, XCircle, Loader2, Usb } from 'lucide-react'

export function ServerSetup() {
  const { ipod, setIPod, navidromeConnected, setNavidromeConnected, absConnected, setABSConnected, setExclusions } = useAppStore()

  const [navURL, setNavURL] = useState('')
  const [navUser, setNavUser] = useState('')
  const [navPass, setNavPass] = useState('')
  const [navTesting, setNavTesting] = useState(false)
  const [navError, setNavError] = useState('')

  const [absURL, setAbsURL] = useState('')
  const [absToken, setAbsToken] = useState('')
  const [absTesting, setAbsTesting] = useState(false)
  const [absError, setAbsError] = useState('')

  const [detecting, setDetecting] = useState(false)

  useState(() => {
    GetConfig().then((cfg) => {
      if (cfg.navidrome?.serverUrl) setNavURL(cfg.navidrome.serverUrl)
      if (cfg.navidrome?.username) setNavUser(cfg.navidrome.username)
      if (cfg.navidrome?.password) setNavPass(cfg.navidrome.password)
      if (cfg.abs?.serverUrl) setAbsURL(cfg.abs.serverUrl)
      if (cfg.abs?.token) setAbsToken(cfg.abs.token)
    })
    GetExclusions().then((exc) => {
      if (exc) setExclusions(exc)
    })
  })

  const testNavidrome = async () => {
    setNavTesting(true)
    setNavError('')
    try {
      await SaveNavidromeConfig(navURL, navUser, navPass)
      await TestNavidromeConnection()
      setNavidromeConnected(true)
    } catch (e) {
      setNavError(String(e))
      setNavidromeConnected(false)
    }
    setNavTesting(false)
  }

  const testABS = async () => {
    setAbsTesting(true)
    setAbsError('')
    try {
      await SaveABSConfig(absURL, absToken)
      await TestABSConnection()
      setABSConnected(true)
    } catch (e) {
      setAbsError(String(e))
      setABSConnected(false)
    }
    setAbsTesting(false)
  }

  const detectIPod = async () => {
    setDetecting(true)
    try {
      const info = await DetectIPod()
      setIPod(info)
    } catch (e) {
      setIPod(null)
    }
    setDetecting(false)
  }

  return (
    <div className="space-y-6 p-4">
      <section className="space-y-3">
        <h2 className="text-lg font-semibold">Navidrome</h2>
        <div className="grid gap-2">
          <Input placeholder="Server URL" value={navURL} onChange={(e) => setNavURL(e.target.value)} />
          <Input placeholder="Username" value={navUser} onChange={(e) => setNavUser(e.target.value)} />
          <Input placeholder="Password" type="password" value={navPass} onChange={(e) => setNavPass(e.target.value)} />
          <div className="flex items-center gap-2">
            <Button onClick={testNavidrome} disabled={navTesting} size="sm">
              {navTesting && <Loader2 className="mr-1 h-4 w-4 animate-spin" />}
              Test Connection
            </Button>
            {navidromeConnected && <CheckCircle className="h-4 w-4 text-green-600" />}
            {navError && <span className="text-sm text-destructive flex items-center gap-1"><XCircle className="h-4 w-4" />{navError}</span>}
          </div>
        </div>
      </section>

      <section className="space-y-3">
        <h2 className="text-lg font-semibold">Audiobookshelf</h2>
        <div className="grid gap-2">
          <Input placeholder="Server URL" value={absURL} onChange={(e) => setAbsURL(e.target.value)} />
          <Input placeholder="API Token" type="password" value={absToken} onChange={(e) => setAbsToken(e.target.value)} />
          <div className="flex items-center gap-2">
            <Button onClick={testABS} disabled={absTesting} size="sm">
              {absTesting && <Loader2 className="mr-1 h-4 w-4 animate-spin" />}
              Test Connection
            </Button>
            {absConnected && <CheckCircle className="h-4 w-4 text-green-600" />}
            {absError && <span className="text-sm text-destructive flex items-center gap-1"><XCircle className="h-4 w-4" />{absError}</span>}
          </div>
        </div>
      </section>

      <section className="space-y-3">
        <h2 className="text-lg font-semibold">iPod</h2>
        <div className="flex items-center gap-2">
          <Button onClick={detectIPod} disabled={detecting} size="sm">
            {detecting ? <Loader2 className="mr-1 h-4 w-4 animate-spin" /> : <Usb className="mr-1 h-4 w-4" />}
            Detect iPod
          </Button>
          {ipod && (
            <span className="text-sm text-muted-foreground">
              {ipod.name} — {formatBytes(ipod.freeSpace)} free / {formatBytes(ipod.totalSpace)}
            </span>
          )}
          {!ipod && !detecting && <span className="text-sm text-muted-foreground">No iPod detected</span>}
        </div>
      </section>
    </div>
  )
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}
