import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { useAppStore } from '@/store/appStore'
import { ServerSetup } from '@/components/ServerSetup'
import { LibraryBrowser } from '@/components/LibraryBrowser'
import { BooksBrowser } from '@/components/BooksBrowser'
import { SyncPanel } from '@/components/SyncPanel'

function App() {
  const { tab, setTab } = useAppStore()

  return (
    <div className="h-screen flex flex-col">
      <Tabs value={tab} onValueChange={(v) => setTab(v as typeof tab)} className="flex flex-col flex-1">
        <TabsList className="mx-4 mt-3 w-fit">
          <TabsTrigger value="setup">Setup</TabsTrigger>
          <TabsTrigger value="music">Music</TabsTrigger>
          <TabsTrigger value="books">Books</TabsTrigger>
          <TabsTrigger value="sync">Sync</TabsTrigger>
        </TabsList>
        <TabsContent value="setup" className="flex-1 overflow-auto"><ServerSetup /></TabsContent>
        <TabsContent value="music" className="flex-1 overflow-auto"><LibraryBrowser /></TabsContent>
        <TabsContent value="books" className="flex-1 overflow-auto"><BooksBrowser /></TabsContent>
        <TabsContent value="sync" className="flex-1 overflow-auto"><SyncPanel /></TabsContent>
      </Tabs>
    </div>
  )
}

export default App
