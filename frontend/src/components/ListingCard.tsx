import { Link } from 'react-router-dom'
import type { Listing } from '../api/client'
import { contactText, formatDate, formatPrice, whatsappLink } from '../api/format'

export default function ListingCard({ listing: l, mine = false }: { listing: Listing; mine?: boolean }) {
  return (
    <article className="listing">
      <Link to={`/listings/${l.id}`} className="listing-main">
        <div className="date-badge" aria-hidden>
          <span>{formatDate(l.event_date, { weekday: undefined, day: undefined })}</span>
          <strong>{formatDate(l.event_date, { weekday: undefined, month: undefined })}</strong>
        </div>
        <div className="listing-body">
          <h3>{l.event_name}</h3>
          <p className="muted small">
            {formatDate(l.event_date)}
            {l.venue && ` · ${l.venue}`}
          </p>
          <p className="price">
            {formatPrice(l.price_cents)}
            {l.price_cents > 0 && <span className="muted small"> each</span>}
            <span className="qty">
              {l.quantity} ticket{l.quantity === 1 ? '' : 's'}
            </span>
          </p>
          {!mine && <p className="muted small">Sold by {l.seller_name}</p>}
        </div>
      </Link>
      {!mine && (
        <a className="whatsapp" href={whatsappLink(l.seller_whatsapp, contactText(l))} target="_blank" rel="noreferrer">
          Message on WhatsApp
        </a>
      )}
    </article>
  )
}
