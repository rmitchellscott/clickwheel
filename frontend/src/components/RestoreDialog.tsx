import { useState, useEffect } from 'react'
import { useAppStore } from '@/store/appStore'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Progress } from '@/components/ui/progress'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { AlertTriangle, Loader2, CheckCircle, X, RotateCcw, ShieldAlert } from 'lucide-react'
import { StartRestore, CancelRestore, GetRecommendedFirmware, GetIPSWCatalog, DetectUSBIPods, CheckFullDiskAccess } from '../../wailsjs/go/main/App'
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
import type { IPSWEntry } from '@/store/appStore'

interface FirmwareMatch {
  entry: IPSWEntry
  index: number
}

export function RestoreDialog() {
  const { ipod, restoreModalOpen, setRestoreModalOpen, restoreProgress, restoring, setRestoring, restoreError, setRestoreError, setRestoreProgress } = useAppStore()
  const [firmwareMatches, setFirmwareMatches] = useState<FirmwareMatch[]>([])
  const [fullCatalog, setFullCatalog] = useState<IPSWEntry[]>([])
  const [selectedIndex, setSelectedIndex] = useState<number>(-1)
  const [deviceName, setDeviceName] = useState('iPod')
  const [confirmed, setConfirmed] = useState(false)
  const [step, setStep] = useState<'configure' | 'confirm' | 'fda' | 'progress' | 'complete' | 'error'>('configure')
  const [noAutoMatch, setNoAutoMatch] = useState(false)
  const [usbIPod, setUsbIPod] = useState<{model: {Name: string, Family: string, Generation: string}, mode: string, diskPath: string} | null>(null)
  const [fdaChecked, setFdaChecked] = useState(false)

  useEffect(() => {
    if (!restoreModalOpen) return
    setStep('configure')
    setConfirmed(false)
    setRestoreError(null)
    setRestoreProgress(null)
    setSelectedIndex(-1)
    setNoAutoMatch(false)
    setUsbIPod(null)

    if (ipod) {
      setDeviceName(ipod.name || 'iPod')
      GetRecommendedFirmware().then(matches => {
        if (matches && matches.length > 0) {
          setFirmwareMatches(matches)
          setSelectedIndex(matches[0].index)
        } else {
          setNoAutoMatch(true)
          GetIPSWCatalog().then(c => setFullCatalog(c || []))
        }
      }).catch(() => {
        setNoAutoMatch(true)
        GetIPSWCatalog().then(c => setFullCatalog(c || []))
      })
    } else {
      DetectUSBIPods().then(ipods => {
        if (ipods && ipods.length > 0) {
          const found = ipods[0]
          if (found.model) {
            setUsbIPod(found as any)
            setDeviceName(found.model.Name)
          }
        }
      })
      GetRecommendedFirmware().then(matches => {
        if (matches && matches.length > 0) {
          setFirmwareMatches(matches)
          setSelectedIndex(matches[0].index)
        } else {
          setNoAutoMatch(true)
          GetIPSWCatalog().then(c => setFullCatalog(c || []))
        }
      }).catch(() => {
        setNoAutoMatch(true)
        GetIPSWCatalog().then(c => setFullCatalog(c || []))
      })
    }
  }, [restoreModalOpen])

  useEffect(() => {
    if (restoring && restoreProgress) {
      setStep('progress')
    }
    if (!restoring && restoreProgress?.state === 'complete') {
      setStep('complete')
    }
    if (restoreError) {
      setStep('error')
    }
  }, [restoring, restoreProgress, restoreError])

  const startRestore = async () => {
    if (selectedIndex < 0) return
    setRestoring(true)
    setRestoreError(null)
    try {
      await StartRestore(selectedIndex, deviceName, usbIPod?.diskPath || '')
    } catch (e) {
      setRestoring(false)
      setRestoreError(String(e))
    }
  }

  const close = () => {
    if (restoring) return
    setRestoreModalOpen(false)
  }

  if (!restoreModalOpen) return null

  const selectedFirmware = noAutoMatch
    ? (selectedIndex >= 0 ? fullCatalog[selectedIndex] : null)
    : firmwareMatches.find(m => m.index === selectedIndex)?.entry ?? null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={() => !restoring && close()} />
      <div className="relative bg-card rounded-xl shadow-xl border w-full max-w-md mx-4">

        {step === 'configure' && (
          <div className="p-6 space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-semibold">Restore iPod</h3>
              <button onClick={close} className="text-muted-foreground hover:text-foreground">
                <X className="h-4 w-4" />
              </button>
            </div>

            {ipod && (
              <div className="bg-muted/50 rounded-lg p-3">
                <p className="text-sm font-medium">{ipod.name}</p>
                <p className="text-xs text-muted-foreground">
                  {[ipod.family, ipod.generation, ipod.displayCapacity].filter(Boolean).join(' · ')}
                </p>
              </div>
            )}

            {!ipod && usbIPod && (
              <div className="bg-muted/50 rounded-lg p-3">
                <p className="text-sm font-medium">{usbIPod.model.Name}</p>
                <p className="text-xs text-muted-foreground">
                  Detected via USB ({usbIPod.mode} mode) at {usbIPod.diskPath}
                </p>
              </div>
            )}

            {!ipod && !usbIPod && (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <AlertTriangle className="h-4 w-4 shrink-0 text-destructive" />
                No iPod detected. Connect your iPod and try again.
              </div>
            )}

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">iPod Name</label>
              <Input value={deviceName} onChange={e => setDeviceName(e.target.value)} />
            </div>

            {/* Auto-matched: single firmware, just show it */}
            {!noAutoMatch && firmwareMatches.length === 1 && selectedFirmware && (
              <div className="bg-muted/50 rounded-lg p-3">
                <p className="text-xs font-medium text-muted-foreground">Firmware</p>
                <p className="text-sm">{selectedFirmware.model} — v{selectedFirmware.version}</p>
              </div>
            )}

            {/* Auto-matched: multiple variants, let user pick */}
            {!noAutoMatch && firmwareMatches.length > 1 && (
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">Firmware variant</label>
                <Select value={String(selectedIndex)} onValueChange={v => setSelectedIndex(Number(v))}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select variant..." />
                  </SelectTrigger>
                  <SelectContent>
                    {firmwareMatches.map(m => (
                      <SelectItem key={m.index} value={String(m.index)}>
                        {m.entry.variant ? `${m.entry.variant} — ` : ''}v{m.entry.version}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            {/* No auto-match: full catalog fallback */}
            {noAutoMatch && (
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">Firmware</label>
                <Select value={selectedIndex >= 0 ? String(selectedIndex) : ''} onValueChange={v => setSelectedIndex(Number(v))}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select firmware..." />
                  </SelectTrigger>
                  <SelectContent>
                    {fullCatalog.map((fw, i) => (
                      <SelectItem key={i} value={String(i)}>
                        {fw.model} {fw.variant ? `(${fw.variant})` : ''} — v{fw.version}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            <div className="flex justify-end gap-2 pt-2">
              <Button variant="outline" onClick={close}>Cancel</Button>
              <Button
                variant="destructive"
                disabled={selectedIndex < 0 || (!ipod && !usbIPod)}
                onClick={() => setStep('confirm')}
              >
                Restore
              </Button>
            </div>
          </div>
        )}

        {step === 'confirm' && (
          <div className="p-6 space-y-4">
            <div className="flex items-center gap-3">
              <AlertTriangle className="h-5 w-5 text-destructive shrink-0" />
              <div>
                <h3 className="text-sm font-semibold">Erase and restore?</h3>
                <p className="text-xs text-muted-foreground mt-1">This will erase all data on the iPod. This cannot be undone.</p>
              </div>
            </div>

            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={confirmed}
                onChange={e => setConfirmed(e.target.checked)}
                className="rounded border-input"
              />
              I understand all data will be erased
            </label>

            <div className="flex justify-end gap-2 pt-2">
              <Button variant="outline" onClick={() => { setStep('configure'); setConfirmed(false) }}>Back</Button>
              <Button variant="destructive" disabled={!confirmed} onClick={async () => {
                const fda = await CheckFullDiskAccess()
                if (fda) {
                  startRestore()
                } else {
                  setStep('fda')
                }
              }}>
                Erase & Restore
              </Button>
            </div>
          </div>
        )}

        {step === 'fda' && (
          <div className="p-6 space-y-4">
            <div className="flex items-center gap-3">
              <ShieldAlert className="h-5 w-5 text-destructive shrink-0" />
              <div>
                <h3 className="text-sm font-semibold">Full Disk Access Required</h3>
                <p className="text-xs text-muted-foreground mt-1">
                  macOS requires Full Disk Access to write directly to the iPod's disk.
                </p>
              </div>
            </div>

            <div className="bg-muted/50 rounded-lg p-3 space-y-2 text-xs">
              <p className="font-medium">To grant access:</p>
              <ol className="list-decimal list-inside space-y-1 text-muted-foreground">
                <li>Open <span className="font-medium text-foreground">System Settings</span></li>
                <li>Go to <span className="font-medium text-foreground">Privacy & Security {'>'} Full Disk Access</span></li>
                <li>Enable <span className="font-medium text-foreground">clickwheel</span> (or add it with the + button)</li>
                <li>Restart the app if prompted</li>
              </ol>
            </div>

            {fdaChecked && (
              <p className="text-xs text-destructive">Full Disk Access has not been granted yet. You may need to restart the app after enabling it.</p>
            )}

            <div className="flex justify-end gap-2 pt-2">
              <Button variant="outline" onClick={() => { setStep('confirm'); setConfirmed(false) }}>Back</Button>
              <Button variant="outline" onClick={() => {
                BrowserOpenURL('x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles')
              }}>
                Open Settings
              </Button>
              <Button onClick={async () => {
                const fda = await CheckFullDiskAccess()
                if (fda) {
                  startRestore()
                } else {
                  setFdaChecked(true)
                }
              }}>
                Check Again
              </Button>
            </div>
          </div>
        )}

        {step === 'progress' && restoreProgress && (
          <div className="p-6 space-y-4">
            <div className="flex items-center gap-3">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
              <div className="flex-1 min-w-0">
                <h3 className="text-sm font-semibold">Restoring iPod...</h3>
                <p className="text-xs text-muted-foreground truncate">{restoreProgress.message}</p>
              </div>
              <span className="ml-auto text-sm font-medium tabular-nums shrink-0">
                {Math.round(restoreProgress.percent)}%
              </span>
            </div>
            <Progress value={restoreProgress.percent} />
            <p className="text-[11px] text-muted-foreground">Do not disconnect the iPod during restore.</p>
            <div className="flex justify-end">
              <Button variant="outline" size="sm" onClick={() => CancelRestore()}>Cancel</Button>
            </div>
          </div>
        )}

        {step === 'complete' && (
          <div className="p-6 space-y-4">
            <div className="flex items-center gap-3">
              <CheckCircle className="h-5 w-5 text-green-500" />
              <div>
                <h3 className="text-sm font-semibold">Restore complete</h3>
                <p className="text-xs text-muted-foreground">Your iPod has been restored successfully.</p>
              </div>
            </div>
            <div className="flex justify-end">
              <Button onClick={close}>Done</Button>
            </div>
          </div>
        )}

        {step === 'error' && (
          <div className="p-6 space-y-4">
            <div className="flex items-center gap-3">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              <div>
                <h3 className="text-sm font-semibold">Restore failed</h3>
                <p className="text-xs text-muted-foreground">{restoreError}</p>
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={close}>Close</Button>
              {restoreProgress?.canRetry && (
                <Button onClick={() => { setRestoreError(null); setStep('configure'); setConfirmed(false) }}>
                  <RotateCcw className="h-4 w-4 mr-1" /> Retry
                </Button>
              )}
            </div>
          </div>
        )}

      </div>
    </div>
  )
}
