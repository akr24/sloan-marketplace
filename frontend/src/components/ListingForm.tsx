import { useState, type FormEvent } from 'react'
import type { Listing, NewListing } from '../api/client'
import { todayISO } from '../api/format'

type Props = {
  initial?: Listing
  submitLabel: string
  busyLabel: string
  onSubmit: (body: NewListing) => Promise<void>
}

// The sell and edit pages share this form; `initial` pre-fills it.
export default function ListingForm({ initial, submitLabel, busyLabel, onSubmit: submit }: Props) {
  const [eventName, setEventName] = useState(initial?.event_name ?? '')
  const [eventDate, setEventDate] = useState(initial?.event_date.slice(0, 10) ?? '')
  const [venue, setVenue] = useState(initial?.venue ?? '')
  const [quantity, setQuantity] = useState(initial?.quantity ?? 1)
  const [price, setPrice] = useState(initial ? String(initial.price_cents / 100) : '')
  const [notes, setNotes] = useState(initial?.notes ?? '')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setBusy(true)
    try {
      await submit({
        event_name: eventName,
        event_date: eventDate,
        venue,
        quantity,
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
    <form onSubmit={onSubmit} className="card">
      <label>
        Event
        <input
          type="text"
          required
          maxLength={120}
          placeholder="e.g. Sloan Fall Ball"
          value={eventName}
          onChange={(e) => setEventName(e.target.value)}
        />
      </label>
      <label>
        Date
        <input type="date" required min={todayISO()} value={eventDate} onChange={(e) => setEventDate(e.target.value)} />
      </label>
      <label>
        <span>Venue <span className="muted small">optional</span></span>
        <input type="text" maxLength={120} value={venue} onChange={(e) => setVenue(e.target.value)} />
      </label>
      <div className="row">
        <label>
          Tickets
          <div className="stepper">
            <button type="button" onClick={() => setQuantity((q) => Math.max(1, q - 1))} aria-label="One fewer">
              −
            </button>
            <output>{quantity}</output>
            <button type="button" onClick={() => setQuantity((q) => Math.min(20, q + 1))} aria-label="One more">
              +
            </button>
          </div>
        </label>
        <label>
          Price each ($)
          <input
            type="number"
            inputMode="decimal"
            min={0}
            step="0.01"
            required
            placeholder="0 for free"
            value={price}
            onChange={(e) => setPrice(e.target.value)}
          />
        </label>
      </div>
      <label>
        <span>Notes <span className="muted small">optional</span></span>
        <textarea
          rows={3}
          maxLength={1000}
          placeholder="Seat section, transfer method, pickup details…"
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
        />
      </label>
      {error && <p className="error">{error}</p>}
      <button type="submit" className="primary" disabled={busy}>
        {busy ? busyLabel : submitLabel}
      </button>
    </form>
  )
}
