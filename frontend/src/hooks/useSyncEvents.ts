import { useEffect } from 'react'
import { useAppStore } from '@/store/appStore'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { PreviewSync, DetectIPods } from '../../wailsjs/go/main/App'

export function useSyncEvents() {
  const { setSyncProgress, setSyncError, setSyncing, setSyncPlan, setSyncComplete } = useAppStore()

  useEffect(() => {
    const unsub1 = EventsOn('sync:progress', (progress) => {
      setSyncProgress(progress)
    })
    const unsub2 = EventsOn('sync:error', (error) => {
      setSyncError(error)
      setSyncing(false)
      setTimeout(() => {
        if (useAppStore.getState().syncError === error) setSyncError(null)
      }, 5000)
    })
    const unsub3 = EventsOn('sync:done', () => {
      setSyncing(false)
      setSyncProgress({ phase: 'done', current: 0, total: 0, message: 'Sync complete!', percent: 100 })
      setSyncComplete(true)
      PreviewSync().then(plan => setSyncPlan(plan)).catch(() => {})
      DetectIPods().then(all => {
        const list = all || []
        useAppStore.getState().setConnectedIPods(list)
        const activeId = useAppStore.getState().activeDeviceId
        const active = list.find(d => d.deviceId === activeId) ?? list[0] ?? null
        useAppStore.getState().setIPod(active)
      }).catch(() => {})
      setTimeout(() => {
        setSyncComplete(false)
        setSyncProgress(null)
      }, 3000)
    })
    return () => { unsub1(); unsub2(); unsub3() }
  }, [setSyncProgress, setSyncError, setSyncing, setSyncPlan, setSyncComplete])
}
