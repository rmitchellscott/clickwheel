import { useAppStore } from '@/store/appStore'
import { cn } from '@/lib/utils'
import { formatBytes, formatDuration } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Search, Check, X, BookOpen, ChevronUp, ChevronDown, ArrowUpDown } from 'lucide-react'
import { useMemo, useEffect, useState } from 'react'
import { GetABSLibraries, GetABSBooks, GetABSProgress, SetInclusions, GetInclusions } from '../../wailsjs/go/main/App'

type SortKey = 'title' | 'author' | 'duration' | 'size'
type SortDir = 'asc' | 'desc'

function SortHeader({ label, active, dir, onClick, className }: {
  label: string; active: boolean; dir: SortDir; onClick: () => void; className?: string
}) {
  return (
    <button onClick={onClick} className={cn('flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors', className)}>
      {label}
      {active
        ? (dir === 'asc' ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />)
        : <ArrowUpDown className="h-3 w-3 opacity-30" />
      }
    </button>
  )
}

export function BooksPage() {
  const {
    absConfigured, absConnected, setSettingsOpen,
    books, setBooks, searchQuery, setSearchQuery,
    includedBooks, toggleBook, toggleAllBooks,
  } = useAppStore()

  const [loaded, setLoaded] = useState(false)
  const [sortKey, setSortKey] = useState<SortKey>('title')
  const [sortDir, setSortDir] = useState<SortDir>('asc')
  const [progress, setProgress] = useState<Record<string, number>>({})

  useEffect(() => {
    if (absConnected && !loaded) {
      GetABSLibraries().then(libs => {
        if (!libs || libs.length === 0) return
        GetABSBooks(libs[0].id).then(bks => {
          const bookList = (bks || []).map(b => ({
            id: b.id,
            title: b.media?.metadata?.title || 'Unknown',
            author: b.media?.metadata?.authorName || 'Unknown',
            duration: b.media?.duration || 0,
            size: b.size || 0,
          }))
          setBooks(bookList)
          setLoaded(true)

          GetABSProgress().then(prog => {
            if (!prog) return
            const pct: Record<string, number> = {}
            for (const [id, mp] of Object.entries(prog)) {
              pct[id] = Math.round(((mp as Record<string, number>).progress || 0) * 100)
            }
            setProgress(pct)
          })

          GetInclusions().then(inc => {
            useAppStore.setState({
              includedBooks: new Set(inc?.books || []),
            })
          })
        })
      })
    }
  }, [absConnected, loaded])

  useEffect(() => {
    if (!absConnected || books.length === 0) return
    const state = useAppStore.getState()
    GetInclusions().then(inc => {
      SetInclusions({
        playlists: inc?.playlists || [],
        albums: inc?.albums || [],
        artists: inc?.artists || [],
        books: [...state.includedBooks],
        podcasts: inc?.podcasts || [],
      })
    })
  }, [includedBooks])

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc')
    } else {
      setSortKey(key)
      setSortDir(key === 'title' || key === 'author' ? 'asc' : 'desc')
    }
  }

  const filteredBooks = useMemo(() => {
    let items = books
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      items = items.filter(b =>
        b.title.toLowerCase().includes(q) || b.author.toLowerCase().includes(q)
      )
    }
    return [...items].sort((a, b) => {
      let cmp = 0
      switch (sortKey) {
        case 'title': cmp = a.title.localeCompare(b.title); break
        case 'author': cmp = a.author.localeCompare(b.author); break
        case 'duration': cmp = a.duration - b.duration; break
        case 'size': cmp = a.size - b.size; break
      }
      return sortDir === 'asc' ? cmp : -cmp
    })
  }, [books, searchQuery, sortKey, sortDir])

  const allIncluded = books.length > 0 && books.every(b => includedBooks.has(b.id))
  const selectedSize = books
    .filter(b => includedBooks.has(b.id))
    .reduce((acc, b) => acc + b.size, 0)

  if (!absConfigured) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center max-w-xs">
          <BookOpen className="h-10 w-10 mx-auto mb-3 text-muted-foreground/30" />
          <h2 className="text-lg font-semibold mb-1">Connect to Audiobookshelf</h2>
          <p className="text-sm text-muted-foreground mb-4">Add your Audiobookshelf server in settings to browse and sync your audiobooks.</p>
          <Button onClick={() => setSettingsOpen(true)}>Open Settings</Button>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between p-4 pb-3 border-b">
        <div>
          <h2 className="text-lg font-semibold">Audiobooks</h2>
          <p className="text-sm text-muted-foreground">
            {includedBooks.size} selected &middot; {formatBytes(selectedSize)}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <div className="relative w-64">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search audiobooks..."
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
              className="pl-9 pr-8"
            />
            {searchQuery && (
              <button onClick={() => setSearchQuery('')} className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground">
                <X className="h-4 w-4" />
              </button>
            )}
          </div>
          <Button variant="ghost" size="sm" className="w-24" onClick={toggleAllBooks}>
            {allIncluded ? 'Deselect All' : 'Select All'}
          </Button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="flex items-center gap-3 px-5 py-2 border-b bg-muted/30">
          <div className="w-4" />
          <SortHeader label="Title" active={sortKey === 'title'} dir={sortDir} className="flex-1"
            onClick={() => handleSort('title')} />
          <SortHeader label="Author" active={sortKey === 'author'} dir={sortDir} className="w-40"
            onClick={() => handleSort('author')} />
          <SortHeader label="Duration" active={sortKey === 'duration'} dir={sortDir} className="w-20 justify-end"
            onClick={() => handleSort('duration')} />
          <SortHeader label="Size" active={sortKey === 'size'} dir={sortDir} className="w-16 justify-end"
            onClick={() => handleSort('size')} />
        </div>

        <div className="p-2">
          <div className="grid gap-1">
            {filteredBooks.map(book => (
              <button
                key={book.id}
                onClick={() => toggleBook(book.id)}
                className={cn(
                  'w-full flex items-center gap-3 px-3 py-3 rounded-lg text-sm transition-colors text-left',
                  includedBooks.has(book.id) ? 'bg-accent/70' : 'hover:bg-accent/30'
                )}
              >
                <div className={cn(
                  'h-4 w-4 rounded border flex items-center justify-center shrink-0 transition-colors',
                  includedBooks.has(book.id) ? 'bg-primary border-primary text-primary-foreground' : 'border-input'
                )}>
                  {includedBooks.has(book.id) && <Check className="h-3 w-3" />}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="font-medium truncate">{book.title}</div>
                  <div className="text-xs text-muted-foreground">
                    {book.author}
                    {progress[book.id] != null && progress[book.id] > 0 && (
                      <span className="ml-2 text-muted-foreground/60">{progress[book.id]}%</span>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-4 shrink-0 text-xs text-muted-foreground">
                  <span className="w-20 text-right">{formatDuration(book.duration)}</span>
                  <span className="w-16 text-right">{formatBytes(book.size)}</span>
                </div>
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
