import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { listings, type Listing } from '../api/client'
import { contactText, formatDate, formatPhone, formatPrice, whatsappLink } from '../api/format'
import { useAuth } from '../auth/context'

export default function ListingPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const [listing, setListing] = useState<Listing | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    listings
      .get(Number(id))
      .then(setListing)
      .catch((err: Error) => setError(err.message))
  }, [id])

  if (error) return <p className="error">{error === 'not found' ? 'That listing no longer exists.' : error}</p>
  if (!listing) return <p className="center muted">Loading…</p>

  const l = listing
  const isMine = user?.id === l.seller_id

  return (
    <article className="detail">
      <Link to="/" className="back">
        ‹ All tickets
      </Link>
      {l.status !== 'active' && <p className="banner">This listing is {l.status}.</p>}
      <h1>{l.event_name}</h1>
      <p className="muted">
        {formatDate(l.event_date, { weekday: 'long', month: 'long', year: 'numeric' })}
        {l.venue && ` · ${l.venue}`}
      </p>

      <div className="facts card">
        <div>
          <span className="muted small">Price</span>
          <strong>{formatPrice(l.price_cents)}</strong>
          {l.price_cents > 0 && <span className="muted small">per ticket</span>}
        </div>
        <div>
          <span className="muted small">Available</span>
          <strong>{l.quantity}</strong>
          <span className="muted small">ticket{l.quantity === 1 ? '' : 's'}</span>
        </div>
      </div>

      {l.notes && <p className="notes">{l.notes}</p>}

      <p className="muted small">
        Listed by {l.seller_name} {l.source === 'whatsapp' && 'via WhatsApp'} on{' '}
        {new Date(l.created_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}
      </p>

      {isMine ? (
        <Link to="/mine" className="button primary block">
          Manage in My listings
        </Link>
      ) : (
        l.status === 'active' && (
          <a
            className="whatsapp block"
            href={whatsappLink(l.seller_whatsapp, contactText(l))}
            target="_blank"
            rel="noreferrer"
          >
            Message {l.seller_name.split(' ')[0]} on WhatsApp
            <span className="small">{formatPhone(l.seller_whatsapp)}</span>
          </a>
        )
      )}
      <p className="muted small center">Payment and handoff happen between you and the seller.</p>
    </article>
  )
}
