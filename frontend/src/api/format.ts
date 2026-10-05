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
