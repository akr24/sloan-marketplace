import { useEffect, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { listings, type Listing } from '../api/client'
import {
  contactText,
  eventColor,
  formatDate,
  formatPhone,
  formatPrice,
  groupShareLink,
  initials,
  timeAgo,
  whatsappLink,
} from '../api/format'
import { useAuth } from '../auth/context'
import Icon from '../components/Icon'

export default function ListingPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const justPosted = useSearchParams()[0].has('posted')
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
  const active = l.status === 'active'
  const color = eventColor(l.event_name)
  const firstName = l.seller_name.split(' ')[0]

  return (
    <article className="detail">
      <section className="event-hero" style={{ background: color.bg, color: color.ink }}>
        <div className="event-hero-top">
          <Link to="/" className="round-btn dark" aria-label="All tickets">
            <Icon name="back" />
          </Link>
          <span className="age-chip">● Listed {timeAgo(l.created_at)}</span>
          {active ? (
            <a
              className="round-btn ghost"
              href={groupShareLink(l, isMine)}
              target="_blank"
              rel="noreferrer"
              aria-label="Share to a WhatsApp group"
            >
              <Icon name="share" />
            </a>
          ) : (
            <span className="round-btn-spacer" />
          )}
        </div>
        <p className="event-when">
          {formatDate(l.event_date, { weekday: 'short', month: 'short', day: 'numeric' })}
          {l.venue && ` · ${l.venue}`}
        </p>
        <h1>{l.event_name}</h1>
        <div className="event-facts">
          <div>
            <strong>{formatPrice(l.price_cents)}</strong>
            {l.price_cents > 0 && <span> /ticket</span>}
          </div>
          <div>
            <strong>{l.quantity}</strong>
            <span> {active ? 'left' : `ticket${l.quantity === 1 ? '' : 's'}`}</span>
          </div>
        </div>
      </section>

      {!active && <p className="banner">This listing is {l.status === 'withdrawn' ? 'no longer available' : 'sold'}.</p>}
      {justPosted && isMine && (
        <p className="banner success">
          <Icon name="check" />
          You're live. Share it to a Sloan WhatsApp group so more people see it.
        </p>
      )}

      <div className="seller">
        <div className="avatar" aria-hidden>
          {initials(l.seller_name)}
        </div>
        <div>
          <strong>{l.seller_name}</strong>
          <p className="muted small">
            mit.edu verified{l.source === 'whatsapp' && ' · posted via WhatsApp'}
          </p>
        </div>
      </div>

      {l.notes && <p className="notes">{l.notes}</p>}

      {isMine ? (
        <div className="cta">
          {active && (
            <a className="button primary block" href={groupShareLink(l, true)} target="_blank" rel="noreferrer">
              <Icon name="share" />
              Share to a WhatsApp group
            </a>
          )}
          <Link to={`/listings/${l.id}/edit`} className="button secondary block">
            Edit listing
          </Link>
          <Link to="/mine" className="share">
            Manage in My listings
          </Link>
        </div>
      ) : (
        active && (
          <div className="cta">
            <p className="muted small">We'll open WhatsApp with this message ready to send</p>
            <p className="bubble">{contactText(l)}</p>
            <a
              className="button primary block big"
              href={whatsappLink(l.seller_whatsapp, contactText(l))}
              target="_blank"
              rel="noreferrer"
            >
              <Icon name="chat" />
              Message {firstName}
            </a>
            <p className="muted small center">
              {formatPhone(l.seller_whatsapp)} · You pay the seller directly.
            </p>
          </div>
        )
      )}
    </article>
  )
}
