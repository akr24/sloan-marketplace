import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { listings, type Listing } from '../api/client'
import { formatDate, formatPhone, formatPrice, groupShareLink, ticketCount } from '../api/format'

// Shown once, straight after a seller publishes from the in-app form.
export default function ListingLivePage() {
  const { id } = useParams()
  const [listing, setListing] = useState<Listing | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    listings
      .get(Number(id))
      .then(setListing)
      .catch((err: Error) => setError(err.message))
  }, [id])

  if (error) return <p className="error">{error}</p>
  if (!listing) return <p className="center muted">Loading…</p>

  const l = listing
  return (
    <>
      <header className="page-head">
        <h1>Listing live</h1>
      </header>

      <p style={{ fontWeight: 600, margin: 0 }} className="check">
        Your tickets are now listed
      </p>
      <p className="muted small">Your listing is live in the marketplace.</p>

      <div className="summary">
        <div>
          <p style={{ fontWeight: 600, fontSize: '0.95rem' }}>{l.event_name}</p>
          <p className="muted">
            {formatDate(l.event_date)}
            {l.venue && ` · ${l.venue}`}
          </p>
          <p>
            {formatPrice(l.price_cents)}
            {l.price_cents > 0 && ' each'} · {ticketCount(l.quantity)}
          </p>
          <p className="muted" style={{ fontSize: '0.75rem' }}>
            Live · just now · via app
          </p>
        </div>
        {l.notes && (
          <div>
            <p className="muted" style={{ fontSize: '0.75rem' }}>
              Note for buyers
            </p>
            <p>{l.notes}</p>
          </div>
        )}
      </div>

      <div className="inset">
        <p style={{ fontWeight: 600, fontSize: '0.9rem', margin: 0 }}>Buyers contact you on WhatsApp</p>
        <p className="small" style={{ margin: 0 }}>
          {formatPhone(l.seller_whatsapp)}
        </p>
      </div>

      <h3 style={{ marginTop: '1.5rem' }}>Manage it from Sell or the bot</h3>
      <p className="small">
        Edit or mark sold from the Sell tab, or message the WhatsApp bot. Want more eyes on it? Share it to a Sloan
        group.
      </p>
      <p className="feed-note" style={{ paddingTop: 0 }}>
        Same listing in the app and bot, so changes stay in sync.
      </p>

      <div className="btn-stack">
        <Link to="/sell" className="btn primary">
          Back to my listings
        </Link>
        <a className="btn outline" href={groupShareLink(l, true)} target="_blank" rel="noreferrer">
          Share to a WhatsApp group
        </a>
        <Link to={`/listings/${l.id}`} className="btn outline">
          View listing
        </Link>
      </div>
    </>
  )
}
