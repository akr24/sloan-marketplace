import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { listings } from '../api/client'
import { whatsappLink } from '../api/format'
import Icon from '../components/Icon'
import ListingForm from '../components/ListingForm'

const BOT_NUMBER = import.meta.env.VITE_BOT_NUMBER ?? ''

// Texting the bot is the fast way to sell; the form stays one tap away.
export default function SellPage() {
  const navigate = useNavigate()
  const [showForm, setShowForm] = useState(!BOT_NUMBER)

  return (
    <>
      <h2>
        Selling?
        <br />
        <span className="accent">Just text it.</span>
      </h2>
      <p className="lede">Send our WhatsApp bot one line and your tickets go live in the feed.</p>

      <div className="chat-sample" aria-label="Example message to the bot">
        <p className="bubble mine">sell Sloan Fall Ball | Nov 14 | $40 | 2</p>
        <p className="muted small">
          That's event | date | price each | number of tickets. Add | venue | notes if you like. Send <code>sold</code>{' '}
          with the listing number when they're gone.
        </p>
      </div>

      {BOT_NUMBER && (
        <a className="button primary block big" href={whatsappLink(BOT_NUMBER, 'help')} target="_blank" rel="noreferrer">
          <Icon name="chat" />
          Open the bot in WhatsApp
        </a>
      )}

      {showForm ? (
        <>
          <h3 className="form-title">List a ticket</h3>
          <ListingForm
            submitLabel="Post it — goes live now"
            busyLabel="Posting…"
            onSubmit={async (body) => {
              const l = await listings.create(body)
              navigate(`/listings/${l.id}?posted`)
            }}
          />
        </>
      ) : (
        <button type="button" className="secondary block" onClick={() => setShowForm(true)}>
          Fill in a form instead
        </button>
      )}
    </>
  )
}
