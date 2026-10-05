// Thin fetch wrapper for the Sloan Marketplace API. Attaches the bearer token
// when one is set and turns non-2xx responses into thrown ApiErrors carrying
// the server's message, so forms can show it directly.

const API_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8081'

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

let token: string | null = null

export function setToken(t: string | null) {
  token = t
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body) headers.set('Content-Type', 'application/json')
  if (token) headers.set('Authorization', `Bearer ${token}`)

  const res = await fetch(`${API_URL}${path}`, { ...init, headers })
  if (!res.ok) {
    let message = res.statusText
    try {
      const body = await res.json()
      if (typeof body.message === 'string') message = body.message
    } catch {
      // not JSON; keep statusText
    }
    throw new ApiError(res.status, message)
  }
  return res.json() as Promise<T>
}

export type User = {
  id: number
  email: string
  name: string
  whatsapp_phone: string
  created_at: string
}

export type Session = { user: User; token: string }

export type VerifyResult = {
  email: string
  session?: Session
  signup_token?: string
}

export type ListingStatus = 'active' | 'sold' | 'withdrawn'

export type Listing = {
  id: number
  seller_id: number
  seller_name: string
  seller_whatsapp: string
  event_name: string
  event_date: string // RFC 3339 at midnight UTC; only the date part matters
  venue: string
  quantity: number
  price_cents: number
  notes: string
  source: 'web' | 'whatsapp'
  status: ListingStatus
  created_at: string
}

export type ListingSort = 'soonest' | 'newest' | 'price_low' | 'price_high'

export type ListingQuery = {
  q?: string
  date_from?: string
  date_to?: string
  max_price?: string
  min_qty?: string
  sort?: ListingSort
}

export type NewListing = {
  event_name: string
  event_date: string
  venue: string
  quantity: number
  price: number
  notes: string
}

const post = (body: unknown) => ({ method: 'POST', body: JSON.stringify(body) })

export const auth = {
  requestCode: (email: string) => api<{ status: string }>('/api/auth/request-code', post({ email })),
  verifyCode: (email: string, code: string) => api<VerifyResult>('/api/auth/verify-code', post({ email, code })),
  completeSignup: (body: { signup_token: string; name: string; whatsapp_phone: string }) =>
    api<Session>('/api/auth/complete-signup', post(body)),
  me: () => api<User>('/api/auth/me'),
}

export const listings = {
  search: (q: ListingQuery) => {
    const params = new URLSearchParams()
    for (const [k, v] of Object.entries(q)) if (v) params.set(k, v)
    const qs = params.toString()
    return api<Listing[]>(`/api/listings${qs ? `?${qs}` : ''}`)
  },
  get: (id: number) => api<Listing>(`/api/listings/${id}`),
  mine: () => api<Listing[]>('/api/listings/mine'),
  create: (body: NewListing) => api<Listing>('/api/listings', post(body)),
  setStatus: (id: number, status: ListingStatus) =>
    api<Listing>(`/api/listings/${id}/status`, { method: 'PATCH', body: JSON.stringify({ status }) }),
}
