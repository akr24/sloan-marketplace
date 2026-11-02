import { Link } from 'react-router-dom'
import type { Listing } from '../api/client'
import {
  contactText,
  formatDate,
  formatPrice,
  isNew,
  shortName,
  sourceLabel,
  ticketCount,
  timeAgo,
  whatsappLink,
} from '../api/format'
import { useAuth } from '../auth/context'

// One compact row in the feed: event and price, date · count · seller, then
// a mono meta line, with a Chat button straight to the seller's WhatsApp.
export default function ListingCard({ listing: l }: { listing: Listing }) {
  const { user } = useAuth()
  const mine = user?.id === l.seller_id

  return (
    <article className="feed-row">
      <Link to={`/listings/${l.id}`}>
        <div className="title">
          <span>{l.event_name}</span>
          <span>{formatPrice(l.price_cents)}</span>
        </div>
        <p className="sub">
          {formatDate(l.event_date)} · {ticketCount(l.quantity)} · {mine ? 'You' : shortName(l.seller_name)}
        </p>
        <p className="meta">
          {isNew(l.created_at) && 'NEW · '}
          {timeAgo(l.created_at)} · {sourceLabel(l)}
          {l.venue && ` · ${l.venue}`}
        </p>
      </Link>
      {mine ? (
        <Link to={`/listings/${l.id}/edit`} className="pill">
          Edit
        </Link>
      ) : (
        <a className="pill" href={whatsappLink(l.seller_whatsapp, contactText(l))} target="_blank" rel="noreferrer">
          Chat
        </a>
      )}
    </article>
  )
}
