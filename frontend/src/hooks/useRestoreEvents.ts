import { useEffect } from 'react'
import { useAppStore } from '@/store/appStore'
import { EventsOn } from '../../wailsjs/runtime/runtime'

export function useRestoreEvents() {
  const { setRestoreProgress, setRestoring, setRestoreError } = useAppStore()

  useEffect(() => {
    const unsub1 = EventsOn('restore:progress', (progress) => {
      setRestoreProgress(progress)
    })
    const unsub2 = EventsOn('restore:error', (error) => {
      setRestoreError(error)
      setRestoring(false)
    })
    const unsub3 = EventsOn('restore:done', () => {
      setRestoring(false)
      setRestoreProgress({
        state: 'complete',
        message: 'Restore complete!',
        percent: 100,
        canRetry: false,
      })
    })
    return () => { unsub1(); unsub2(); unsub3() }
  }, [setRestoreProgress, setRestoring, setRestoreError])
}
