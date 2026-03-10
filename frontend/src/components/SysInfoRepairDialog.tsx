import { useState } from 'react'
import { useAppStore } from '@/store/appStore'
import { AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction } from '@/components/ui/alert-dialog'
import { Loader2 } from 'lucide-react'
import { ApproveSysInfoRepair, DetectIPod } from '../../wailsjs/go/main/App'

export function SysInfoRepairDialog() {
  const { ipod, sysInfoRepairOpen, setSysInfoRepairOpen, setIPod } = useAppStore()
  const [writing, setWriting] = useState(false)

  if (!ipod?.needsSysInfoRepair || !ipod.proposedSysInfo) return null

  const fields: { label: string; value: string }[] = []
  for (const line of ipod.proposedSysInfo.split('\n')) {
    const idx = line.indexOf(':')
    if (idx === -1) continue
    const key = line.slice(0, idx).trim()
    const val = line.slice(idx + 1).trim()
    if (!val) continue

    const labelMap: Record<string, string> = {
      pszSerialNumber: 'Serial Number',
      FirewireGuid: 'FireWire GUID',
      visibleBuildID: 'Build ID',
      BoardHwName: 'Board Name',
      iPodFamily: 'Family ID',
      updaterFamily: 'Updater Family',
    }
    fields.push({ label: labelMap[key] || key, value: val })
  }

  const familyGen = [ipod.family, ipod.generation].filter(Boolean).join(' ')

  const handleWrite = async () => {
    setWriting(true)
    try {
      await ApproveSysInfoRepair(ipod.proposedSysInfo!)
      const info = await DetectIPod()
      setIPod(info)
    } catch {}
    setWriting(false)
    setSysInfoRepairOpen(false)
  }

  return (
    <AlertDialog open={sysInfoRepairOpen} onOpenChange={setSysInfoRepairOpen}>
      <AlertDialogContent className="max-w-md">
        <AlertDialogHeader>
          <AlertDialogTitle>Device Information Missing</AlertDialogTitle>
          <AlertDialogDescription asChild>
            <div className="space-y-3">
              <p>
                This iPod's SysInfo file is missing or incomplete, likely due to a storage adapter (iFlash/CF).
                Device information was recovered directly from the iPod's hardware.
              </p>

              {familyGen && (
                <p className="font-medium text-foreground">{familyGen}</p>
              )}

              <div className="rounded-md border p-3 space-y-1.5 text-sm">
                {fields.map((f, i) => (
                  <div key={i} className="flex justify-between gap-4">
                    <span className="text-muted-foreground">{f.label}</span>
                    <span className="font-mono text-xs text-foreground">{f.value}</span>
                  </div>
                ))}
              </div>

              <p className="text-xs text-muted-foreground">
                Writing this information to the iPod will persist it across reconnections.
              </p>
            </div>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={writing}>Skip</AlertDialogCancel>
          <AlertDialogAction onClick={handleWrite} disabled={writing}>
            {writing && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
            Write to iPod
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
