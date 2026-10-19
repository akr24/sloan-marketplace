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

  return (
    <>
      <h2>My listings</h2>
      {items.length === 0 && (
        <div className="empty">
          <p>You haven't listed anything yet.</p>
          <Link to="/sell" className="button primary">
            Sell tickets
          </Link>
        </div>
      )}
      <div className="list">
        {items.map((l) => (
          <div key={l.id} className={l.status === 'active' ? '' : 'faded'}>
            <ListingCard listing={l} mine />
            <div className="actions">
              <span className={`status ${l.status}`}>{l.status}</span>
              {l.status === 'active' ? (
                <>
                  <Link to={`/listings/${l.id}/edit`} className="share">
                    Edit
                  </Link>
                  <a className="share" href={groupShareLink(l, true)} target="_blank" rel="noreferrer">
                    Share
                  </a>
                  <button type="button" className="secondary" onClick={() => setStatus(l.id, 'sold')}>
                    Mark sold
                  </button>
                  <button type="button" className="link" onClick={() => setStatus(l.id, 'withdrawn')}>
                    Remove
                  </button>
                </>
              ) : (
                <button type="button" className="link" onClick={() => setStatus(l.id, 'active')}>
                  Relist
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
    </>
  )
}
