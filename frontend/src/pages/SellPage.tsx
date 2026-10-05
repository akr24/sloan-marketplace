import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { listings } from '../api/client'
import { todayISO } from '../api/format'

export default function SellPage() {
  const navigate = useNavigate()
  const [eventName, setEventName] = useState('')
  const [eventDate, setEventDate] = useState('')
  const [venue, setVenue] = useState('')
  const [quantity, setQuantity] = useState(1)
  const [price, setPrice] = useState('')
  const [notes, setNotes] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setBusy(true)
    try {
      const l = await listings.create({
        event_name: eventName,
        event_date: eventDate,
        venue,
        quantity,
        price: Number(price),
        notes,
      })
      navigate(`/listings/${l.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not post listing')
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <h2>Sell tickets</h2>
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
          {busy ? 'Posting…' : 'Post listing'}
        </button>
      </form>
      <p className="muted small center">
        Rather post from WhatsApp? <Link to="/account">Here's how</Link>.
      </p>
    </>
  )
}
