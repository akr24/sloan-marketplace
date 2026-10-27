// One flow for sign-up and sign-in: email → emailed code → (first time
// only) name and WhatsApp number. Owning an MIT inbox is the membership check.

import { useState, type FormEvent } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { auth } from '../api/client'
import { useAuth } from '../auth/context'

type Step = 'email' | 'code' | 'profile'

export default function SignInPage() {
  const { accept } = useAuth()
  const navigate = useNavigate()
  const from = (useLocation().state as { from?: string } | null)?.from ?? '/'

  const [step, setStep] = useState<Step>('email')
  const [email, setEmail] = useState('')
  const [code, setCode] = useState('')
  const [signupToken, setSignupToken] = useState('')
  const [name, setName] = useState('')
  const [phone, setPhone] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function run(fn: () => Promise<void>) {
    setError(null)
    setNotice(null)
    setBusy(true)
    try {
      await fn()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setBusy(false)
    }
  }

  // Most people type just their Kerberos name; a full address (say
  // @sloan.mit.edu) is passed through as typed.
  const fullEmail = email.includes('@') ? email.trim() : `${email.trim()}@mit.edu`

  const sendCode = (e: FormEvent) => {
    e.preventDefault()
    run(async () => {
      await auth.requestCode(fullEmail)
      setCode('')
      setStep('code')
    })
  }

  const resend = () =>
    run(async () => {
      await auth.requestCode(fullEmail)
      setNotice('New code sent.')
    })

  const verify = (e: FormEvent) => {
    e.preventDefault()
    run(async () => {
      const res = await auth.verifyCode(fullEmail, code)
      if (res.session) {
        accept(res.session)
        navigate(from, { replace: true })
      } else if (res.signup_token) {
        setSignupToken(res.signup_token)
        setStep('profile')
      }
    })
  }

  const finish = (e: FormEvent) => {
    e.preventDefault()
    run(async () => {
      const s = await auth.completeSignup({ signup_token: signupToken, name, whatsapp_phone: phone })
      accept(s)
      navigate(from, { replace: true })
    })
  }

  return (
    <div className="signin">
      <div className="logo" aria-hidden>
        SM
      </div>
      <h1>Sloan Marketplace</h1>
      <p className="tagline">Every ticket Sloanies are selling, live in one place.</p>

      {step !== 'profile' && (
        <form onSubmit={sendCode}>
          <label>
            MIT email
            <span className="suffixed">
              <input
                type="text"
                inputMode="email"
                autoComplete="username"
                autoCapitalize="none"
                autoCorrect="off"
                spellCheck={false}
                placeholder="kerberos"
                required
                readOnly={step === 'code'}
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
              {!email.includes('@') && <span>@mit.edu</span>}
            </span>
          </label>
          {step === 'email' ? (
            <>
              {error && <p className="error">{error}</p>}
              <button type="submit" className="btn primary" disabled={busy}>
                {busy ? 'Sending…' : 'Send code'}
              </button>
            </>
          ) : (
            <button type="button" className="link" onClick={() => setStep('email')} style={{ alignSelf: 'flex-start' }}>
              Use a different email
            </button>
          )}
        </form>
      )}

      {step === 'code' && (
        <form onSubmit={verify} style={{ marginTop: '1rem' }}>
          <label>
            6-digit code
            <span className="code-boxes">
              {Array.from({ length: 6 }, (_, i) => (
                <span key={i} className={i === code.length ? 'cursor' : ''} aria-hidden>
                  {code[i] ?? ''}
                </span>
              ))}
              <input
                type="text"
                inputMode="numeric"
                autoComplete="one-time-code"
                pattern="[0-9]{6}"
                maxLength={6}
                required
                autoFocus
                aria-label={`Code sent to ${fullEmail}`}
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
              />
            </span>
            <span className="hint">Sent to {fullEmail}.</span>
          </label>
          {error && <p className="error">{error}</p>}
          {notice && <p className="notice">{notice}</p>}
          <button type="submit" className="btn primary" disabled={busy || code.length !== 6}>
            {busy ? 'Checking…' : 'Start browsing'}
          </button>
          <button type="button" className="link" onClick={resend} disabled={busy} style={{ alignSelf: 'center' }}>
            Resend code
          </button>
        </form>
      )}

      {step === 'profile' && (
        <form onSubmit={finish}>
          <p className="small">
            <strong className="check">Email verified.</strong> Last step: how buyers reach you.
          </p>
          <label>
            Your name
            <input type="text" autoComplete="name" required value={name} onChange={(e) => setName(e.target.value)} />
          </label>
          <label>
            WhatsApp number
            <input
              type="tel"
              inputMode="tel"
              autoComplete="tel"
              placeholder="+1 617 555 0123"
              required
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
            />
            <span className="hint">
              Include the country code. Buyers tap your listing to message you here, and it's how the WhatsApp bot
              knows it's you.
            </span>
          </label>
          {error && <p className="error">{error}</p>}
          <button type="submit" className="btn primary" disabled={busy}>
            {busy ? 'Creating account…' : 'Create account'}
          </button>
        </form>
      )}

      <div className="dashed" style={{ marginTop: '2rem' }}>
        <p>
          <strong>Selling?</strong>
        </p>
        <p className="muted" style={{ margin: 0 }}>
          Sign in once, then text the Sloan Marketplace bot on WhatsApp and your listing appears here.
        </p>
      </div>
      <p className="footnote mono">Only mit.edu addresses can join. Payment and handoff happen between buyer and seller.</p>
    </div>
  )
}
