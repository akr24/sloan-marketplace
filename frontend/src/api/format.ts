import type { Listing } from './client'

// Display helpers shared by the listing views.

// Event dates arrive as midnight UTC; format in UTC so they never shift a day.
export function formatDate(iso: string, opts: Intl.DateTimeFormatOptions = {}) {
  return new Date(iso).toLocaleDateString('en-US', {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
    timeZone: 'UTC',
    ...opts,
  })
}

export function formatPrice(cents: number) {
  if (cents === 0) return 'Free'
  const dollars = cents / 100
  return dollars.toLocaleString('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: cents % 100 === 0 ? 0 : 2,
  })
}

export function formatPhone(digits: string) {
  if (digits.length === 11 && digits.startsWith('1')) {
    return `+1 (${digits.slice(1, 4)}) ${digits.slice(4, 7)}-${digits.slice(7)}`
  }
  return `+${digits}`
}

// wa.me opens a chat with the seller in WhatsApp, prefilled so they know
// which listing the buyer means.
export function whatsappLink(phone: string, text: string) {
  return `https://wa.me/${phone}?text=${encodeURIComponent(text)}`
}

// Today in Cambridge as YYYY-MM-DD, matching how the API stores event dates.
export function todayISO(offsetDays = 0) {
  const d = new Date()
  d.setDate(d.getDate() + offsetDays)
  return d.toLocaleDateString('en-CA', { timeZone: 'America/New_York' })
}

// The message prefilled when a buyer taps through to a seller.
export function contactText(l: Listing) {
  return `Hi ${l.seller_name.split(' ')[0]}! I saw your Sloan Marketplace listing for ${l.event_name} (${formatDate(l.event_date)}). Are the tickets still available?`
}

// The post a seller drops into a Sloan WhatsApp group. wa.me/?text with no
// number opens WhatsApp's chat picker, so the seller chooses the group.
export function groupShareLink(l: Listing, mine: boolean) {
  const lines = [
    `🎟️ ${mine ? 'Selling' : 'Tickets for'}: ${l.event_name}`,
    `${formatDate(l.event_date)}${l.venue ? ` · ${l.venue}` : ''}`,
    `${l.quantity} ticket${l.quantity === 1 ? '' : 's'} · ${formatPrice(l.price_cents)}${l.price_cents > 0 ? ' each' : ''}`,
    `${window.location.origin}/listings/${l.id}`,
  ]
  return `https://wa.me/?text=${encodeURIComponent(lines.join('\n'))}`
}

// "just now", "4m ago", "3h ago", "2d ago" for when a listing went up.
export function timeAgo(iso: string, now = Date.now()) {
  const mins = Math.floor((now - new Date(iso).getTime()) / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

// Listings from the last hour get a NEW tag in the feed.
export function isFresh(iso: string, now = Date.now()) {
  return now - new Date(iso).getTime() < 60 * 60 * 1000
}

// Each event gets a stable color from the palette, so the same event looks
// the same everywhere it appears. `ink` is the readable text color on it.
const EVENT_COLORS = [
  { bg: '#6d4aff', ink: '#ffffff' },
  { bg: '#5ce1e6', ink: '#0c0c0e' },
  { bg: '#ff5c8a', ink: '#0c0c0e' },
  { bg: '#3d7bff', ink: '#0c0c0e' },
  { bg: '#c6f432', ink: '#0c0c0e' },
]

export function eventColor(name: string) {
  let h = 0
  for (const ch of name.trim().toLowerCase()) h = (h * 31 + ch.charCodeAt(0)) >>> 0
  return EVENT_COLORS[h % EVENT_COLORS.length]
}

export function initials(name: string) {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  return ((parts[0]?.[0] ?? '') + (parts.length > 1 ? parts[parts.length - 1][0] : '')).toUpperCase()
}
