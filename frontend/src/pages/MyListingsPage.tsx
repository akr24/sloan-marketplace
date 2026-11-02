import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listings, type Listing, type ListingStatus } from '../api/client'
import { groupShareLink } from '../api/format'
import ListingCard from '../components/ListingCard'

export default function MyListingsPage() {
  const [items, setItems] = useState<Listing[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    listings
      .mine()
      .then(setItems)
      .catch((err: Error) => setError(err.message))
  }, [])

  async function setStatus(id: number, status: ListingStatus) {
    try {
      const updated = await listings.setStatus(id, status)
      setItems((prev) => prev?.map((l) => (l.id === id ? { ...l, status: updated.status } : l)) ?? null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not update listing')
    }
  }

  if (error) return <p className="error">{error}</p>
  if (!items) return <p className="center muted">Loading…</p>

  const live = items.filter((l) => l.status === 'active')
  const done = items.filter((l) => l.status !== 'active')

  return (
    <>
      <h2>Your tickets</h2>
      {items.length === 0 && (
        <div className="empty">
          <p>You haven't listed anything yet.</p>
          <Link to="/sell" className="button primary">
            Sell tickets
          </Link>
        </div>
      )}

      {live.length > 0 && <p className="group-label">Live · {live.length}</p>}
      <div className="list">
        {live.map((l) => (
          <div key={l.id} className="mine-card">
            <ListingCard listing={l} mine />
            <div className="mine-actions">
              <button type="button" className="primary" onClick={() => setStatus(l.id, 'sold')}>
                Mark sold
              </button>
              <Link to={`/listings/${l.id}/edit`} className="button outline">
                Edit
              </Link>
              <a className="button outline" href={groupShareLink(l, true)} target="_blank" rel="noreferrer">
                Share
              </a>
            </div>
            <button type="button" className="link small" onClick={() => setStatus(l.id, 'withdrawn')}>
              Remove listing
            </button>
          </div>
        ))}
      </div>

      {done.length > 0 && <p className="group-label">Sold &amp; removed · {done.length}</p>}
      <div className="list">
        {done.map((l) => (
          <div key={l.id} className="done-row">
            <ListingCard listing={l} mine />
            <button type="button" className="link small" onClick={() => setStatus(l.id, 'active')}>
              Relist
            </button>
          </div>
        ))}
      </div>

      {items.length > 0 && (
        <p className="tip muted small">
          You can also manage listings from WhatsApp: send <code>mine</code> to the bot to see them, and{' '}
          <code>sold</code> with a listing number to close one.
        </p>
      )}
    </>
  )
}
