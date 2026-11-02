import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { listings, type Listing, type ListingQuery, type ListingSort } from '../api/client'
import { todayISO } from '../api/format'
import Icon from '../components/Icon'
import ListingCard from '../components/ListingCard'

type When = '' | 'tonight' | 'week' | 'month'

// How often the feed checks for new and sold listings.
const REFRESH_MS = 30_000

const HEADINGS: Record<ListingSort, string> = {
  newest: 'Just dropped',
  soonest: 'Coming up',
  price_low: 'Cheapest first',
  price_high: 'Priciest first',
}

// The buyer's live feed. Filters live in the URL so back/forward and shared
// links keep them.
export default function BrowsePage() {
  const [params, setParams] = useSearchParams()
  const q = params.get('q') ?? ''
  const when = (params.get('when') ?? '') as When
  const maxPrice = params.get('max_price') ?? ''
  const minQty = params.get('min_qty') ?? ''
  const sort = (params.get('sort') ?? 'newest') as ListingSort

  const [search, setSearch] = useState(q)
  const [results, setResults] = useState<Listing[] | null>(null)
  const [pending, setPending] = useState<Listing[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [showFilters, setShowFilters] = useState(false)
  const shown = useRef<Listing[] | null>(null)
  useEffect(() => {
    shown.current = results
  }, [results])

  function update(key: string, value: string) {
    const next = new URLSearchParams(params)
    if (value) next.set(key, value)
    else next.delete(key)
    setParams(next, { replace: true })
  }

  const toggle = (key: string, value: string, current: string) => update(key, current === value ? '' : value)

  // Debounce typing so each keystroke isn't a request.
  useEffect(() => {
    const t = setTimeout(() => {
      if (search !== q) update('q', search.trim())
    }, 300)
    return () => clearTimeout(t)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search])

  const fetchListings = useCallback(() => {
    const query: ListingQuery = { q, max_price: maxPrice, min_qty: minQty, sort }
    if (when === 'tonight') query.date_to = todayISO(0)
    if (when === 'week') query.date_to = todayISO(7)
    if (when === 'month') query.date_to = todayISO(31)
    return listings.search(query)
  }, [q, when, maxPrice, minQty, sort])

  // Load whenever the filters change.
  useEffect(() => {
    let cancelled = false
    setPending(null)
    fetchListings()
      .then((r) => !cancelled && (setResults(r), setError(null)))
      .catch((err: Error) => !cancelled && setError(err.message))
    return () => {
      cancelled = true
    }
  }, [fetchListings])

  // Keep the feed live. Sold listings drop out quietly; new ones wait behind
  // a "new tickets" pill so the list doesn't jump while someone is reading.
  useEffect(() => {
    const id = setInterval(() => {
      if (document.hidden) return
      fetchListings()
        .then((fresh) => {
          const seen = new Set((shown.current ?? []).map((l) => l.id))
          const added = fresh.filter((l) => !seen.has(l.id))
          if (added.length === 0) {
            setResults(fresh)
            setPending(null)
          } else {
            setPending(fresh)
          }
        })
        .catch(() => {})
    }, REFRESH_MS)
    return () => clearInterval(id)
  }, [fetchListings])

  const newCount = pending ? pending.filter((l) => !(results ?? []).some((r) => r.id === l.id)).length : 0

  function showPending() {
    if (pending) setResults(pending)
    setPending(null)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const extraFilters = (sort !== 'newest' ? 1 : 0) + (maxPrice && maxPrice !== '50' ? 1 : 0)

  return (
    <>
      <header className="feed-head">
        <span className="wordmark">
          <span className="dot" aria-hidden />
          Sloan Marketplace
        </span>
        <span className="live">
          <span className="live-dot" aria-hidden />
          Live
        </span>
      </header>

      <div className="toolbar">
        <label className="search">
          <Icon name="search" size={18} />
          <input
            type="search"
            placeholder="Search events, venues"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            aria-label="Search listings"
          />
        </label>
        <button
          type="button"
          className={`icon-btn ${showFilters || extraFilters ? 'on' : ''}`}
          onClick={() => setShowFilters((s) => !s)}
          aria-expanded={showFilters}
          aria-label={`More filters${extraFilters ? ` (${extraFilters} on)` : ''}`}
        >
          <Icon name="sliders" size={18} />
        </button>
      </div>

      <div className="chips" role="group" aria-label="Quick filters">
        <button type="button" className={`chip ${!when ? 'on' : ''}`} onClick={() => update('when', '')}>
          Any date
        </button>
        {(
          [
            ['tonight', 'Tonight'],
            ['week', 'This week'],
            ['month', 'This month'],
          ] as const
        ).map(([v, label]) => (
          <button key={v} type="button" className={`chip ${when === v ? 'on' : ''}`} onClick={() => toggle('when', v, when)}>
            {label}
          </button>
        ))}
        <button type="button" className={`chip ${maxPrice === '50' ? 'on' : ''}`} onClick={() => toggle('max_price', '50', maxPrice)}>
          Under $50
        </button>
        <button type="button" className={`chip ${minQty === '2' ? 'on' : ''}`} onClick={() => toggle('min_qty', '2', minQty)}>
          2+ tickets
        </button>
      </div>

      {showFilters && (
        <div className="filters sheet">
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
          {(when || maxPrice || minQty || sort !== 'newest') && (
            <button type="button" className="link" onClick={() => setParams(q ? { q } : {}, { replace: true })}>
              Clear filters
            </button>
          )}
        </div>
      )}

      <div className="section-head">
        <h1>{HEADINGS[sort] ?? 'Tickets'}</h1>
        {results && <span className="muted small">{results.length} available</span>}
      </div>

      {newCount > 0 && (
        <button type="button" className="new-pill" onClick={showPending}>
          <Icon name="up" size={14} stroke={3} />
          {newCount} new ticket{newCount === 1 ? '' : 's'}
        </button>
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
