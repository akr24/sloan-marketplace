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

  const sendCode = (e: FormEvent) => {
    e.preventDefault()
    run(async () => {
      await auth.requestCode(email)
      setCode('')
      setStep('code')
    })
  }

  const resend = () =>
    run(async () => {
      await auth.requestCode(email)
      setNotice('New code sent.')
    })

  const verify = (e: FormEvent) => {
    e.preventDefault()
    run(async () => {
      const res = await auth.verifyCode(email, code)
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
      <div className="hero">
        <div className="logo" aria-hidden>
          🎟️
        </div>
        <h1>Sloan Marketplace</h1>
        <p className="muted">Buy and sell event tickets with fellow Sloanies.</p>
      </div>

      {step === 'email' && (
        <form onSubmit={sendCode} className="card">
          <label>
            MIT email
            <input
              type="email"
              inputMode="email"
              autoComplete="email"
              autoCapitalize="none"
              placeholder="you@mit.edu"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </label>
          {error && <p className="error">{error}</p>}
          <button type="submit" className="primary" disabled={busy}>
            {busy ? 'Sending…' : 'Email me a code'}
          </button>
          <p className="muted small">New or returning, it's the same step. We only accept mit.edu addresses.</p>
        </form>
      )}

      {step === 'code' && (
        <form onSubmit={verify} className="card">
          <p>
            We sent a 6-digit code to <strong>{email}</strong>.
          </p>
          <label>
            Code
            <input
              className="code"
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              pattern="[0-9]{6}"
              maxLength={6}
              required
              autoFocus
              value={code}
              onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
            />
          </label>
          {error && <p className="error">{error}</p>}
          {notice && <p className="notice">{notice}</p>}
          <button type="submit" className="primary" disabled={busy || code.length !== 6}>
            {busy ? 'Checking…' : 'Continue'}
          </button>
          <div className="row between">
            <button type="button" className="link" onClick={() => setStep('email')}>
              Use a different email
            </button>
            <button type="button" className="link" onClick={resend} disabled={busy}>
              Resend code
            </button>
          </div>
        </form>
      )}

      {step === 'profile' && (
        <form onSubmit={finish} className="card">
          <p>
            <strong>Email verified.</strong> Last step: how buyers will reach you.
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
            <span className="muted small">
              Include the country code. Buyers tap your listing to message you here, and it's how our WhatsApp bot
              knows it's you.
            </span>
          </label>
          {error && <p className="error">{error}</p>}
          <button type="submit" className="primary" disabled={busy}>
            {busy ? 'Creating account…' : 'Create account'}
          </button>
        </form>
      )}
    </div>
  )
}
