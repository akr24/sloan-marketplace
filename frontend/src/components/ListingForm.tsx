import { useEffect, useState, type FormEvent } from 'react'
import { listings, type Listing, type NewListing } from '../api/client'
import { formatPhone, todayISO } from '../api/format'
import { useAuth } from '../auth/context'

type Props = {
  initial?: Listing
  submitLabel: string
  busyLabel: string
  onSubmit: (body: NewListing) => Promise<void>
}

// The sell and edit pages share this form; `initial` pre-fills it.
export default function ListingForm({ initial, submitLabel, busyLabel, onSubmit: submit }: Props) {
  const { user } = useAuth()
  const [quantityText, setQuantityText] = useState(String(initial?.quantity ?? 1))
  const [eventName, setEventName] = useState(initial?.event_name ?? '')
  const [eventDate, setEventDate] = useState(initial?.event_date.slice(0, 10) ?? '')
  const [venue, setVenue] = useState(initial?.venue ?? '')
  const [price, setPrice] = useState(initial ? String(initial.price_cents / 100) : '')
  const [notes, setNotes] = useState(initial?.notes ?? '')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [events, setEvents] = useState<string[]>([])

  // Suggest events already on the marketplace so the same event gets the
  // same name, which keeps search and the feed tidy.
  useEffect(() => {
    listings
      .search({ sort: 'soonest' })
      .then((r) => setEvents([...new Set(r.map((l) => l.event_name))]))
      .catch(() => {})
  }, [])

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setBusy(true)
    try {
      await submit({
        event_name: eventName,
        event_date: eventDate,
        venue,
        quantity: Number(quantityText),
        price: Number(price),
        notes,
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={onSubmit}>
      <label>
        Event
        <input
          type="text"
          required
          maxLength={120}
          list="known-events"
          placeholder="e.g. Sloan Follies"
          value={eventName}
          onChange={(e) => setEventName(e.target.value)}
        />
        <datalist id="known-events">
          {events.map((name) => (
            <option key={name} value={name} />
          ))}
        </datalist>
      </label>
      <div className="row">
        <label>
          Date
          <input type="date" required min={todayISO()} value={eventDate} onChange={(e) => setEventDate(e.target.value)} />
        </label>
        <label>
          <span>
            Venue <span className="muted">(optional)</span>
          </span>
          <input type="text" maxLength={120} placeholder="Kresge" value={venue} onChange={(e) => setVenue(e.target.value)} />
        </label>
      </div>
      <div className="row">
        <label>
          Price per ticket
          <input
            type="number"
            inputMode="decimal"
            min={0}
            step="0.01"
            required
            placeholder="$20 (0 for free)"
            value={price}
            onChange={(e) => setPrice(e.target.value)}
          />
        </label>
        <label>
          Quantity
          <input
            type="number"
            inputMode="numeric"
            min={1}
            max={20}
            required
            value={quantityText}
            onChange={(e) => setQuantityText(e.target.value)}
          />
        </label>
      </div>
      <label>
        <span>
          Note for buyers <span className="muted">(optional)</span>
        </span>
        <textarea
          rows={2}
          maxLength={1000}
          placeholder="Can transfer tickets before the show."
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
        />
      </label>
      {user && (
        <label>
          <span>
            Seller WhatsApp · <span className="check">linked</span>
          </span>
          <input type="tel" readOnly value={formatPhone(user.whatsapp_phone)} />
          <span className="hint">The number you signed up with. Buyers message you here.</span>
        </label>
      )}
      {error && <p className="error">{error}</p>}
      <div className="sticky-cta">
        <button type="submit" className="btn primary" disabled={busy}>
          {busy ? busyLabel : submitLabel}
        </button>
      </div>
    </form>
  )
}
