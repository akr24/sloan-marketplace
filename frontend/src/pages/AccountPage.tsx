import { formatPhone, whatsappLink } from '../api/format'
import { useAuth } from '../auth/context'

const BOT_NUMBER = import.meta.env.VITE_BOT_NUMBER ?? ''

export default function AccountPage() {
  const { user, logout } = useAuth()
  if (!user) return null

  return (
    <div className="account">
      <header className="page-head">
        <h1>Account</h1>
      </header>

      <dl>
        <dt>Name</dt>
        <dd>{user.name}</dd>
        <dt>MIT email</dt>
        <dd>
          {user.email} <span className="check muted small">verified</span>
        </dd>
        <dt>WhatsApp</dt>
        <dd>{formatPhone(user.whatsapp_phone)}</dd>
      </dl>

      <section className="section" style={{ marginTop: '1rem', borderTop: '1px solid var(--border)' }}>
        <h3>Bot commands</h3>
        <p className="small muted">
          Message the bot from {formatPhone(user.whatsapp_phone)}. In a Sloan group, start with <code>/mp</code>.
        </p>
        <div className="inset mono" style={{ lineHeight: 1.7 }}>
          sell 2 Follies tix, $20 each, Nov 13
          <br />
          mine
          <br />
          edit 12 price $25
          <br />
          sold 12
          <br />
          help
        </div>
        {BOT_NUMBER && (
          <a
            className="btn outline"
            style={{ marginTop: '0.75rem' }}
            href={whatsappLink(BOT_NUMBER, 'help')}
            target="_blank"
            rel="noreferrer"
          >
            Open the bot in WhatsApp
          </a>
        )}
      </section>

      <button type="button" className="btn outline" style={{ marginTop: '1.5rem' }} onClick={logout}>
        Sign out
      </button>
    </div>
  )
}
