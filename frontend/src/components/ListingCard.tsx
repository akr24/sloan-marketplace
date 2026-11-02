import { Link } from 'react-router-dom'
import type { Listing } from '../api/client'
import { contactText, eventColor, formatDate, formatPrice, isFresh, timeAgo, whatsappLink } from '../api/format'
import Icon from './Icon'

// One row in the feed: a colored date tile, the event, price and age, and a
// one-tap button to WhatsApp the seller.
export default function ListingCard({ listing: l, mine = false }: { listing: Listing; mine?: boolean }) {
  const color = eventColor(l.event_name)
  const active = l.status === 'active'
  const fresh = active && isFresh(l.created_at)
  return (
    <article className={`listing${fresh ? ' fresh' : ''}${active ? '' : ' inactive'}`}>
      <Link to={`/listings/${l.id}`} className="listing-main">
        <div className="date-tile" style={{ background: color.bg, color: color.ink }} aria-hidden>
          <span>{formatDate(l.event_date, { weekday: undefined, day: undefined })}</span>
          <strong>{formatDate(l.event_date, { weekday: undefined, month: undefined })}</strong>
        </div>
        <div className="listing-body">
          <div className="listing-title">
            <h3>{l.event_name}</h3>
            {fresh && <span className="tag-new">NEW</span>}
          </div>
          <p className="meta">
            <span className="sr-only">{formatDate(l.event_date)} · </span>
            {l.venue && `${l.venue} · `}
            {l.quantity} ticket{l.quantity === 1 ? '' : 's'}
          </p>
        </div>
        <div className="listing-price">
          <strong>{formatPrice(l.price_cents)}</strong>
          <span>{timeAgo(l.created_at)}</span>
        </div>
      </Link>
      {!mine && active && (
        <a
          className="wa-btn"
          href={whatsappLink(l.seller_whatsapp, contactText(l))}
          target="_blank"
          rel="noreferrer"
          aria-label={`Message ${l.seller_name.split(' ')[0]} on WhatsApp about ${l.event_name}`}
        >
          <Icon name="chat" />
        </a>
      )}
      {!active && <span className="tag-sold">{l.status === 'sold' ? 'SOLD' : 'OFF'}</span>}
    </article>
  )
}
