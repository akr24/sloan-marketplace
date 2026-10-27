import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listings, type Listing, type ListingStatus } from '../api/client'
import { formatPrice, sourceLabel, whatsappLink } from '../api/format'

const BOT_NUMBER = import.meta.env.VITE_BOT_NUMBER ?? ''

// The Sell tab: how to post through the bot, with the in-app form as the
// fallback, and the seller's own listings underneath. The bot and this page
// manage the same listings.
export default function SellPage() {
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

  const live = items?.filter((l) => l.status === 'active') ?? []
  const past = items?.filter((l) => l.status !== 'active') ?? []

  return (
    <>
      <header className="page-head">
        <h1>Sell</h1>
      </header>

      <section className="section" style={{ marginTop: '-1rem' }}>
        <h3>Sell by texting the bot</h3>
        <ol className="steps">
          <li>
            <span>1</span>
            <span>
              Message the bot like you'd post in a group:
              <code>sell 2 Follies tix, $20 each, Nov 13</code>
            </span>
          </li>
          <li>
            <span>2</span>
            <span>Reply yes to its preview and it goes live here.</span>
          </li>
          <li>
            <span>3</span>
            <span>Buyers WhatsApp you directly. Mark it sold here or in the bot when they're gone.</span>
          </li>
        </ol>
        {BOT_NUMBER ? (
          <a className="btn primary" href={whatsappLink(BOT_NUMBER, 'sell ')} target="_blank" rel="noreferrer">
            Open WhatsApp bot
          </a>
        ) : (
          <p className="hint">The bot's number will appear here once it's set up.</p>
        )}
        <p className="center" style={{ margin: '0.25rem 0 0' }}>
          <Link to="/sell/new" className="link">
            or fill out a form in the app instead
          </Link>
        </p>
      </section>

      <section style={{ paddingTop: '1rem' }}>
        <p className="section-label">Your live listings ({live.length})</p>
        {error && <p className="error">{error}</p>}
        {!items && !error && <p className="muted small">Loading…</p>}
        {items && live.length === 0 && <p className="muted small">Nothing live right now.</p>}
        {live.map((l) => (
          <MineRow key={l.id} listing={l}>
            <Link to={`/listings/${l.id}/edit`} className="pill">
              Edit
            </Link>
            <button type="button" className="pill" onClick={() => setStatus(l.id, 'sold')}>
              Sold
            </button>
          </MineRow>
        ))}

        {past.length > 0 && (
          <>
            <p className="section-label" style={{ marginTop: '1.5rem' }}>
              Sold or removed ({past.length})
            </p>
            {past.map((l) => (
              <MineRow key={l.id} listing={l} faded>
                <button type="button" className="pill" onClick={() => setStatus(l.id, 'active')}>
                  Relist
                </button>
              </MineRow>
            ))}
          </>
        )}
        <p className="feed-note">Same listings the bot manages. Editing here or replying in WhatsApp both work.</p>
      </section>
    </>
  )
}

function MineRow({ listing: l, faded, children }: { listing: Listing; faded?: boolean; children: React.ReactNode }) {
  return (
    <div className={`mine-row ${faded ? 'faded' : ''}`}>
      <Link to={`/listings/${l.id}`}>
        <strong>{l.event_name}</strong>
        <span className="sub">
          {l.quantity} × {formatPrice(l.price_cents)} · {faded ? l.status : sourceLabel(l)}
        </span>
      </Link>
      {children}
    </div>
  )
}
