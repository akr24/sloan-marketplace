package usecase

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/akr24/sloan-marketplace/backend/internal/domain"
)

// Bot turns WhatsApp messages from sellers into listings. It is channel
// agnostic: it takes the sender's number and message text and returns the
// reply text; the WhatsApp adapter handles delivery.
//
// The sender's WhatsApp number is how it finds the account, which is why
// users.whatsapp_phone is unique. WhatsApp itself authenticates the sender.
type Bot struct {
	users    domain.UserRepository
	listings *Listings
	appURL   string
	now      func() time.Time
}

func NewBot(users domain.UserRepository, listings *Listings, appURL string) *Bot {
	return &Bot{users: users, listings: listings, appURL: strings.TrimRight(appURL, "/"), now: time.Now}
}

// Handle processes one inbound message and returns the reply.
func (b *Bot) Handle(ctx context.Context, fromPhone, text string) (string, error) {
	phone, err := NormalisePhone(fromPhone)
	if err != nil {
		return "", err
	}
	user, err := b.users.FindByPhone(ctx, phone)
	if errors.Is(err, domain.ErrNotFound) {
		return fmt.Sprintf("Hi! This number isn't linked to a Sloan Marketplace account yet. "+
			"Sign up with your MIT email at %s and enter this WhatsApp number (+%s), then message me again.", b.appURL, phone), nil
	}
	if err != nil {
		return "", err
	}

	text = strings.TrimSpace(text)
	cmd, rest := splitCommand(text)
	switch cmd {
	case "sell", "post":
		return b.sell(ctx, user, rest)
	case "mine", "my", "listings":
		return b.mine(ctx, user)
	case "sold", "remove", "delete", "withdraw":
		return b.close(ctx, user, cmd, rest)
	default:
		return b.help(user), nil
	}
}

func (b *Bot) sell(ctx context.Context, u *domain.User, body string) (string, error) {
	in, err := ParseSellMessage(body, b.now())
	if err != nil {
		return fmt.Sprintf("I couldn't read that listing: %v\n\n%s", err, sellExample), nil
	}
	l, err := b.listings.Create(ctx, u.ID, domain.SourceWhatsApp, in)
	if errors.Is(err, ErrInvalidInput) {
		return fmt.Sprintf("I couldn't post that: %s\n\n%s", strings.TrimPrefix(err.Error(), ErrInvalidInput.Error()+": "), sellExample), nil
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Posted #%d: %d × %s on %s at %s each.\n%s/listings/%d\n\nReply \"sold %d\" when they're gone.",
		l.ID, l.Quantity, l.EventName, l.EventDate.Format("Mon Jan 2"), formatPrice(l.PriceCents), b.appURL, l.ID, l.ID), nil
}

func (b *Bot) mine(ctx context.Context, u *domain.User) (string, error) {
	all, err := b.listings.Mine(ctx, u.ID)
	if err != nil {
		return "", err
	}
	var lines []string
	for _, l := range all {
		if l.Status == domain.StatusActive {
			lines = append(lines, fmt.Sprintf("#%d %d × %s, %s, %s", l.ID, l.Quantity, l.EventName, l.EventDate.Format("Jan 2"), formatPrice(l.PriceCents)))
		}
	}
	if len(lines) == 0 {
		return "You have no active listings. Send \"sell\" to see how to post one.", nil
	}
	return "Your active listings:\n" + strings.Join(lines, "\n") + "\n\nReply \"sold <number>\" to close one.", nil
}

func (b *Bot) close(ctx context.Context, u *domain.User, cmd, rest string) (string, error) {
	id, err := strconv.ParseInt(strings.TrimPrefix(strings.TrimSpace(rest), "#"), 10, 64)
	if err != nil {
		return "Which listing? Reply e.g. \"sold 12\". Send \"mine\" to see your listing numbers.", nil
	}
	status := domain.StatusWithdrawn
	if cmd == "sold" {
		status = domain.StatusSold
	}
	l, err := b.listings.SetStatus(ctx, u.ID, id, status)
	if errors.Is(err, domain.ErrNotFound) {
		return fmt.Sprintf("You don't have a listing #%d. Send \"mine\" to see yours.", id), nil
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Marked #%d (%s) as %s.", l.ID, l.EventName, status), nil
}

func (b *Bot) help(u *domain.User) string {
	return fmt.Sprintf("Hi %s! I post ticket listings to Sloan Marketplace for you.\n\n%s\n\n"+
		"Other commands:\n• mine: list your active listings\n• sold 12: mark listing #12 sold\n• remove 12: take it down\n\nBrowse: %s",
		firstName(u.Name), sellExample, b.appURL)
}

const sellExample = `To post tickets, send:
sell
Event: Sloan Fall Ball
Date: Nov 14
Price: $40
Qty: 2
Venue: Boston Harbor Hotel (optional)
Notes: anything else (optional)

Or in one line:
sell Sloan Fall Ball | Nov 14 | $40 | 2`

// ParseSellMessage reads a listing from the text after "sell". It accepts
// "Key: value" lines (event, date, price, qty, venue, notes, any order) or a
// single "event | date | price | qty | venue | notes" line.
func ParseSellMessage(body string, now time.Time) (ListingInput, error) {
	fields := map[string]string{}
	body = strings.TrimSpace(body)

	if isPipeLine(body) {
		parts := strings.Split(body, "|")
		keys := []string{"event", "date", "price", "qty", "venue", "notes"}
		for i, p := range parts {
			if i >= len(keys) {
				// Extra pipes belong to the notes.
				fields["notes"] += " | " + strings.TrimSpace(p)
				continue
			}
			fields[keys[i]] = strings.TrimSpace(p)
		}
	} else {
		for _, line := range strings.Split(body, "\n") {
			k, v, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			if key := fieldKey(k); key != "" {
				fields[key] = strings.TrimSpace(v)
			}
		}
	}

	var in ListingInput
	in.EventName = fields["event"]
	in.Venue = fields["venue"]
	in.Notes = strings.TrimSpace(fields["notes"])
	if in.EventName == "" {
		return in, errors.New("missing the event name")
	}

	date, err := ParseEventDate(fields["date"], now)
	if err != nil {
		return in, err
	}
	in.EventDate = date

	price, err := ParsePriceCents(fields["price"])
	if err != nil {
		return in, err
	}
	in.PriceCents = price

	in.Quantity = 1
	if q := strings.TrimSpace(fields["qty"]); q != "" {
		n, err := strconv.Atoi(strings.Fields(q)[0])
		if err != nil {
			return in, fmt.Errorf("quantity %q isn't a number", q)
		}
		in.Quantity = n
	}
	return in, nil
}

// isPipeLine reports whether the message uses the one-line form. Only the
// first line counts, so a "|" inside multi-line notes doesn't flip the mode.
func isPipeLine(body string) bool {
	first, _, _ := strings.Cut(body, "\n")
	return strings.Contains(first, "|")
}

func fieldKey(k string) string {
	switch strings.ToLower(strings.TrimSpace(k)) {
	case "event", "event name", "name", "what", "title":
		return "event"
	case "date", "when", "day":
		return "date"
	case "price", "cost", "$", "price each", "price per ticket":
		return "price"
	case "qty", "quantity", "tickets", "count", "number":
		return "qty"
	case "venue", "where", "location", "place":
		return "venue"
	case "notes", "note", "details", "info":
		return "notes"
	}
	return ""
}

var dateLayouts = []string{
	"2006-01-02",
	"1/2/2006", "1/2/06",
	"Jan 2 2006", "January 2 2006", "2 Jan 2006", "2 January 2006",
}

var yearlessLayouts = []string{
	"1/2", "Jan 2", "January 2", "2 Jan", "2 January",
}

// ParseEventDate accepts common ways people write a date. Without a year it
// picks the next occurrence, so "Jan 10" sent in November means next January.
func ParseEventDate(s string, now time.Time) (time.Time, error) {
	orig := s
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return time.Time{}, errors.New("missing the date")
	}
	today := campusDate(now)
	switch s {
	case "today", "tonight":
		return today, nil
	case "tomorrow":
		return today.AddDate(0, 0, 1), nil
	}

	s = strings.NewReplacer(",", " ", ".", " ").Replace(s)
	words := strings.Fields(s)
	// Drop a leading weekday ("sat nov 14") and ordinal suffixes ("14th").
	if len(words) > 1 && isWeekday(words[0]) {
		words = words[1:]
	}
	for i, w := range words {
		for _, suf := range []string{"st", "nd", "rd", "th"} {
			if n := strings.TrimSuffix(w, suf); n != w && n != "" && n[0] >= '0' && n[0] <= '9' {
				words[i] = n
			}
		}
	}
	s = strings.Join(words, " ")

	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, titleMonth(s)); err == nil {
			return t, nil
		}
	}
	for _, layout := range yearlessLayouts {
		if t, err := time.Parse(layout, titleMonth(s)); err == nil {
			t = time.Date(today.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			if t.Before(today) {
				t = t.AddDate(1, 0, 0)
			}
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("I don't understand the date %q; try e.g. Nov 14 or 11/14", strings.TrimSpace(orig))
}

func isWeekday(w string) bool {
	for d := time.Sunday; d <= time.Saturday; d++ {
		name := strings.ToLower(d.String())
		if w == name || w == name[:3] {
			return true
		}
	}
	return false
}

// titleMonth capitalises words so time.Parse matches "Jan" and "January".
func titleMonth(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// ParsePriceCents reads "$40", "40", "40.50", "$40 each" or "free".
func ParsePriceCents(s string) (int, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, errors.New("missing the price")
	}
	if s == "free" || s == "0" {
		return 0, nil
	}
	s = strings.TrimPrefix(strings.Fields(s)[0], "$")
	s = strings.TrimSuffix(strings.ReplaceAll(s, ",", ""), "usd")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f < 0 {
		return 0, fmt.Errorf("price %q isn't a dollar amount", s)
	}
	return int(f*100 + 0.5), nil
}

func formatPrice(cents int) string {
	if cents == 0 {
		return "free"
	}
	if cents%100 == 0 {
		return fmt.Sprintf("$%d", cents/100)
	}
	return fmt.Sprintf("$%d.%02d", cents/100, cents%100)
}

func splitCommand(text string) (cmd, rest string) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", ""
	}
	cmd = strings.ToLower(strings.Trim(fields[0], "!/.:"))
	rest = strings.TrimSpace(text[strings.Index(text, fields[0])+len(fields[0]):])
	rest = strings.TrimPrefix(rest, ":")
	return cmd, rest
}

func firstName(name string) string {
	if f := strings.Fields(name); len(f) > 0 {
		return f[0]
	}
	return "there"
}
