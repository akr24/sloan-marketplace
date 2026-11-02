import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import { listings, type Listing, type ListingStatus } from '../api/client'
import {
  contactText,
  firstName,
  formatDate,
  formatPrice,
  groupShareLink,
  initials,
  shortName,
  sourceLabel,
  timeAgo,
  whatsappLink,
} from '../api/format'
import { useAuth } from '../auth/context'

export default function ListingPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const navigate = useNavigate()
  // Opened straight from a shared link there's no page to go back to.
  const canGoBack = useLocation().key !== 'default'
  const [listing, setListing] = useState<Listing | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    listings
      .get(Number(id))
      .then(setListing)
      .catch((err: Error) => setError(err.message))
  }, [id])

  async function setStatus(status: ListingStatus) {
    if (!listing) return
    try {
      setListing(await listings.setStatus(listing.id, status))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not update listing')
    }
  }

  const back = (
    <header className="page-head">
      <button type="button" className="link" style={{ padding: 0, textDecoration: 'none' }} onClick={() => (canGoBack ? navigate(-1) : navigate('/'))}>
        ‹ Back
      </button>
      {listing && (
        <span className={`aside live-dot ${listing.status === 'active' ? '' : 'off'}`}>
          {listing.status === 'active' ? 'Available' : listing.status === 'sold' ? 'Sold' : 'Removed'} · posted{' '}
          {timeAgo(listing.created_at)}
        </span>
      )}
    </header>
  )

  if (error) {
    return (
      <>
        {back}
        <p className="error">{error === 'not found' ? 'That listing no longer exists.' : error}</p>
      </>
    )
  }
  if (!listing) return <p className="center muted">Loading…</p>

  const l = listing
  const isMine = user?.id === l.seller_id
  const prefill = contactText(l)

  return (
    <article>
      {back}
      <h1 className="detail-title">{l.event_name}</h1>
      <p className="detail-price">
        {formatPrice(l.price_cents)} {l.price_cents > 0 && <span>per ticket · </span>}
        <span>{l.quantity} available</span>
      </p>

      <dl className="facts">
        <div>
          <dt>Date</dt>
          <dd>{formatDate(l.event_date)}</dd>
        </div>
        <div>
          <dt>Tickets</dt>
          <dd>{l.quantity}</dd>
        </div>
        <div>
          <dt>Venue</dt>
          <dd>{l.venue || '—'}</dd>
        </div>
        <div>
          <dt>Posted</dt>
          <dd>{sourceLabel(l).replace('via ', 'Via ')}</dd>
        </div>
      </dl>

      {l.notes && (
        <div className="inset quote">
          <p>Note from {isMine ? 'you' : 'the seller'}</p>
          <p>{l.notes}</p>
        </div>
      )}

      <div className="box seller">
        <span className="avatar" aria-hidden>
          {initials(l.seller_name)}
        </span>
        <div>
          <strong>{isMine ? 'You' : shortName(l.seller_name)}</strong>
          <div className="meta">MIT email verified</div>
        </div>
      </div>

      {isMine ? (
        <div className="sticky-cta">
          {l.status === 'active' ? (
            <>
              <div className="owner-actions">
                <Link to={`/listings/${l.id}/edit`} className="btn outline">
                  Edit
                </Link>
                <button type="button" className="btn outline" onClick={() => setStatus('sold')}>
                  Mark sold
                </button>
              </div>
              <a className="btn primary" href={groupShareLink(l, true)} target="_blank" rel="noreferrer">
                Share to a WhatsApp group
              </a>
            </>
          ) : (
            <button type="button" className="btn primary" onClick={() => setStatus('active')}>
              Relist
            </button>
          )}
        </div>
      ) : (
        l.status === 'active' && (
          <div className="sticky-cta">
            <a className="btn primary" href={whatsappLink(l.seller_whatsapp, prefill)} target="_blank" rel="noreferrer">
              Message {firstName(l.seller_name)} on WhatsApp
            </a>
            <p className="mono">Prefilled: “{prefill}”</p>
          </div>
        )
      )}
    </article>
  )
}
