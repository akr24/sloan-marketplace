import { useEffect, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { listings, type Listing, type ListingQuery, type ListingSort } from '../api/client'
import { todayISO } from '../api/format'
import ListingCard from '../components/ListingCard'

const POLL_MS = 30_000

// Filters live in the URL so back/forward and shared links keep them.
// The feed is newest-first by default, like a group chat.
export default function BrowsePage() {
  const [params, setParams] = useSearchParams()
  const q = params.get('q') ?? ''
  const week = params.get('when') === 'week'
  const maxPrice = params.get('max_price') ?? ''
  const minQty = params.get('min_qty') ?? ''
  const sort = (params.get('sort') ?? 'newest') as ListingSort

  const [search, setSearch] = useState(q)
  const [results, setResults] = useState<Listing[] | null>(null)
  const [incoming, setIncoming] = useState<Listing[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [showFilters, setShowFilters] = useState(false)

  function update(key: string, value: string) {
    const next = new URLSearchParams(params)
    if (value) next.set(key, value)
    else next.delete(key)
    setParams(next, { replace: true })
  }

  // Debounce typing so each keystroke isn't a request.
  useEffect(() => {
    const t = setTimeout(() => {
      if (search !== q) update('q', search.trim())
    }, 300)
    return () => clearTimeout(t)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search])

  const query = useMemo<ListingQuery>(
    () => ({ q, max_price: maxPrice, min_qty: minQty, sort, date_to: week ? todayISO(7) : undefined }),
    [q, maxPrice, minQty, sort, week],
  )

  useEffect(() => {
    let cancelled = false
    setIncoming(null)
    listings
      .search(query)
      .then((r) => !cancelled && (setResults(r), setError(null)))
      .catch((err: Error) => !cancelled && setError(err.message))

    // Check for new posts in the background and offer them with a tap,
    // rather than shuffling the list under the reader's thumb.
    const t = setInterval(() => {
      if (document.hidden) return
      listings
        .search(query)
        .then((r) => !cancelled && setIncoming(r))
        .catch(() => {})
    }, POLL_MS)
    return () => {
      cancelled = true
      clearInterval(t)
    }
  }, [query])

  const known = useMemo(() => new Set(results?.map((l) => l.id)), [results])
  const newCount = incoming?.filter((l) => !known.has(l.id)).length ?? 0

  const extraFilters = [minQty].filter(Boolean).length + (sort !== 'newest' ? 1 : 0) +
    (maxPrice && maxPrice !== '50' ? 1 : 0)

  return (
    <>
      <header className="page-head">
        <h1>Tickets</h1>
        <span className={`aside live-dot ${error ? 'off' : ''}`}>
          Live{results ? ` · ${results.length} available` : ''}
        </span>
      </header>

      <input
        type="search"
        className="search"
        placeholder="Search events, teams, venues"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        aria-label="Search listings"
      />

      <div className="chips" role="group" aria-label="Quick filters">
        <button
          type="button"
          className={`chip ${sort === 'newest' ? 'on' : ''}`}
          onClick={() => update('sort', sort === 'newest' ? 'soonest' : '')}
        >
          Newest
        </button>
        <button type="button" className={`chip ${week ? 'on' : ''}`} onClick={() => update('when', week ? '' : 'week')}>
          This week
        </button>
        <button
          type="button"
          className={`chip ${maxPrice === '50' ? 'on' : ''}`}
          onClick={() => update('max_price', maxPrice === '50' ? '' : '50')}
        >
          Under $50
        </button>
        <button
          type="button"
          className={`chip ${showFilters || extraFilters ? 'on' : ''}`}
          onClick={() => setShowFilters((s) => !s)}
          aria-expanded={showFilters}
        >
          Filters{extraFilters ? ` · ${extraFilters}` : ''}
        </button>
      </div>

      {showFilters && (
        <div className="filters">
          <label>
            Max price per ticket
            <select value={maxPrice} onChange={(e) => update('max_price', e.target.value)}>
              <option value="">Any price</option>
              <option value="0">Free only</option>
              <option value="25">Up to $25</option>
              <option value="50">Up to $50</option>
              <option value="100">Up to $100</option>
              <option value="200">Up to $200</option>
            </select>
          </label>
          <label>
            Tickets needed
            <select value={minQty} onChange={(e) => update('min_qty', e.target.value)}>
              <option value="">Any</option>
              <option value="2">2 or more</option>
              <option value="3">3 or more</option>
              <option value="4">4 or more</option>
            </select>
          </label>
          <label>
            Sort by
            <select value={sort} onChange={(e) => update('sort', e.target.value === 'newest' ? '' : e.target.value)}>
              <option value="newest">Newest listing</option>
              <option value="soonest">Soonest event</option>
              <option value="price_low">Price: low to high</option>
              <option value="price_high">Price: high to low</option>
            </select>
          </label>
          <button type="button" className="link" onClick={() => setParams(q ? { q } : {}, { replace: true })}>
            Clear filters
          </button>
        </div>
      )}

      {newCount > 0 && (
        <button
          type="button"
          className="new-pill"
          onClick={() => {
            setResults(incoming)
            setIncoming(null)
            window.scrollTo({ top: 0, behavior: 'smooth' })
          }}
        >
          ↑ {newCount} new listing{newCount === 1 ? '' : 's'}
        </button>
      )}

      {error && <p className="error">{error}</p>}
      {results === null && !error && <p className="center muted">Loading…</p>}
      {results?.length === 0 && (
        <div className="empty">
          <p>No tickets match yet.</p>
          <Link to="/sell" className="btn primary">
            Sell yours
          </Link>
        </div>
      )}
      <div className="feed">
        {results?.map((l) => (
          <ListingCard key={l.id} listing={l} />
        ))}
      </div>
      {results && results.length > 0 && (
        <p className="feed-note">New posts appear at the top. Sold and past events drop off, so everything here is available.</p>
      )}
    </>
  )
}
