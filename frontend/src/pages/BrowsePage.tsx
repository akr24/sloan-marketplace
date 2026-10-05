import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { listings, type Listing, type ListingQuery, type ListingSort } from '../api/client'
import { todayISO } from '../api/format'
import ListingCard from '../components/ListingCard'

type When = '' | 'week' | 'month'

// Filters live in the URL so back/forward and shared links keep them.
export default function BrowsePage() {
  const [params, setParams] = useSearchParams()
  const q = params.get('q') ?? ''
  const when = (params.get('when') ?? '') as When
  const maxPrice = params.get('max_price') ?? ''
  const minQty = params.get('min_qty') ?? ''
  const sort = (params.get('sort') ?? 'soonest') as ListingSort

  const [search, setSearch] = useState(q)
  const [results, setResults] = useState<Listing[] | null>(null)
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

  useEffect(() => {
    const query: ListingQuery = { q, max_price: maxPrice, min_qty: minQty, sort }
    if (when === 'week') query.date_to = todayISO(7)
    if (when === 'month') query.date_to = todayISO(31)
    let cancelled = false
    listings
      .search(query)
      .then((r) => !cancelled && (setResults(r), setError(null)))
      .catch((err: Error) => !cancelled && setError(err.message))
    return () => {
      cancelled = true
    }
  }, [q, when, maxPrice, minQty, sort])

  const activeFilters = [when, maxPrice, minQty].filter(Boolean).length + (sort !== 'soonest' ? 1 : 0)

  return (
    <>
      <div className="toolbar">
        <input
          type="search"
          className="search"
          placeholder="Search events, venues…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          aria-label="Search listings"
        />
        <button
          type="button"
          className={`chip ${showFilters || activeFilters ? 'on' : ''}`}
          onClick={() => setShowFilters((s) => !s)}
          aria-expanded={showFilters}
        >
          Filters{activeFilters ? ` · ${activeFilters}` : ''}
        </button>
      </div>

      <div className="chips" role="group" aria-label="When">
        {(
          [
            ['', 'Any date'],
            ['week', 'Next 7 days'],
            ['month', 'Next 30 days'],
          ] as const
        ).map(([v, label]) => (
          <button key={v} type="button" className={`chip ${when === v ? 'on' : ''}`} onClick={() => update('when', v)}>
            {label}
          </button>
        ))}
      </div>

      {showFilters && (
        <div className="filters card">
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
            <select value={sort} onChange={(e) => update('sort', e.target.value === 'soonest' ? '' : e.target.value)}>
              <option value="soonest">Soonest event</option>
              <option value="newest">Newest listing</option>
              <option value="price_low">Price: low to high</option>
              <option value="price_high">Price: high to low</option>
            </select>
          </label>
          {activeFilters > 0 && (
            <button type="button" className="link" onClick={() => setParams(q ? { q } : {}, { replace: true })}>
              Clear filters
            </button>
          )}
        </div>
      )}

      {error && <p className="error">{error}</p>}
      {results === null && !error && <p className="center muted">Loading…</p>}
      {results?.length === 0 && (
        <div className="empty">
          <p>No tickets match yet.</p>
          <Link to="/sell" className="button primary">
            Sell yours
          </Link>
        </div>
      )}
      <div className="list">
        {results?.map((l) => (
          <ListingCard key={l.id} listing={l} />
        ))}
      </div>
    </>
  )
}
