import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { useAppStore } from '@/store/appStore'
import { GetABSLibraries, GetABSBooks, SetExclusions } from '../../wailsjs/go/main/App'

export function BooksBrowser() {
  const { libraries, setLibraries, books, setBooks, exclusions, setExclusions, absConnected } = useAppStore()
  const [loaded, setLoaded] = useState(false)
  const [selectedLib, setSelectedLib] = useState('')

  useEffect(() => {
    if (absConnected && !loaded) {
      GetABSLibraries().then(libs => {
        setLibraries(libs || [])
        if (libs && libs.length > 0) {
          setSelectedLib(libs[0].id)
        }
        setLoaded(true)
      })
    }
  }, [absConnected, loaded, setLibraries])

  useEffect(() => {
    if (selectedLib) {
      GetABSBooks(selectedLib).then(b => setBooks(b || []))
    }
  }, [selectedLib, setBooks])

  if (!absConnected) {
    return <div className="p-4 text-muted-foreground">Connect to Audiobookshelf in Setup first.</div>
  }

  const toggleBook = (id: string) => {
    const next = exclusions.books.includes(id)
      ? exclusions.books.filter(x => x !== id)
      : [...exclusions.books, id]
    const updated = { ...exclusions, books: next }
    setExclusions(updated)
    SetExclusions(updated)
  }

  const allBooksSelected = books.every(b => !exclusions.books.includes(b.id))
  const toggleAllBooks = () => {
    const updated = {
      ...exclusions,
      books: allBooksSelected ? books.map(b => b.id) : []
    }
    setExclusions(updated)
    SetExclusions(updated)
  }

  const formatDuration = (seconds: number) => {
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    return h > 0 ? `${h}h ${m}m` : `${m}m`
  }

  return (
    <div className="p-4 space-y-4">
      {libraries.length > 1 && (
        <div className="flex gap-2">
          {libraries.map(lib => (
            <Button
              key={lib.id}
              variant={selectedLib === lib.id ? 'default' : 'outline'}
              size="sm"
              onClick={() => setSelectedLib(lib.id)}
            >
              {lib.name}
            </Button>
          ))}
        </div>
      )}

      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Audiobooks</h2>
        <Button variant="ghost" size="sm" onClick={toggleAllBooks}>
          {allBooksSelected ? 'Deselect All' : 'Select All'}
        </Button>
      </div>

      <div className="space-y-1 max-h-[500px] overflow-y-auto">
        {books.map(book => (
          <label key={book.id} className="flex items-center gap-2 text-sm p-1 rounded hover:bg-accent">
            <input
              type="checkbox"
              checked={!exclusions.books.includes(book.id)}
              onChange={() => toggleBook(book.id)}
              className="rounded"
            />
            <span>{book.media.metadata.title}</span>
            <span className="text-muted-foreground">— {book.media.metadata.authorName}</span>
            <span className="text-muted-foreground ml-auto">{formatDuration(book.media.duration)}</span>
          </label>
        ))}
      </div>
    </div>
  )
}
