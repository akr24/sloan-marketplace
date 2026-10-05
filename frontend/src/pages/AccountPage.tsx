import { formatPhone, whatsappLink } from '../api/format'
import { useAuth } from '../auth/context'

const BOT_NUMBER = import.meta.env.VITE_BOT_NUMBER ?? ''

export default function AccountPage() {
  const { user, logout } = useAuth()
  if (!user) return null

  return (
    <>
      <h2>Account</h2>
      <div className="card">
        <p>
          <strong>{user.name}</strong>
        </p>
        <p className="muted">{user.email}</p>
        <p className="muted">WhatsApp {formatPhone(user.whatsapp_phone)}</p>
      </div>

      <h3>Post from WhatsApp</h3>
      <div className="card bot-help">
        <p>
          Message our bot from <strong>{formatPhone(user.whatsapp_phone)}</strong> and it posts the listing for you.
          Send:
        </p>
        <pre>sell Sloan Fall Ball | Nov 14 | $40 | 2</pre>
        <p className="muted small">
          That's event | date | price each | number of tickets, optionally followed by | venue | notes. Send{' '}
          <code>help</code> for the long form, <code>mine</code> to see your listings, and <code>sold 12</code> to close
          one.
        </p>
        {BOT_NUMBER ? (
          <a className="whatsapp block" href={whatsappLink(BOT_NUMBER, 'help')} target="_blank" rel="noreferrer">
            Open the bot in WhatsApp
          </a>
        ) : (
          <p className="muted small">The bot's number will appear here once it's set up.</p>
        )}
      </div>

      <button type="button" className="secondary block" onClick={logout}>
        Sign out
      </button>
    </>
  )
}
