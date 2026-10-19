package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/akr24/sloan-marketplace/backend/internal/domain"
)

// Bot turns WhatsApp messages from sellers into listings. It is channel
// agnostic: it takes the sender's number and message text and returns the
// reply text; the WhatsApp adapters handle delivery.
//
// The sender's WhatsApp number is how it finds the account, which is why
// users.whatsapp_phone is unique. WhatsApp itself authenticates the sender.
//
// In a direct chat every message is a command ("sell ...", "mine"). In a
// group the bot only answers messages starting with GroupPrefix, so normal
// chatter is ignored, and its replies never reveal anyone's phone number.
type Bot struct {
	users    domain.UserRepository
	listings *Listings
	appURL   string
	now      func() time.Time

	// drafts are parsed listings waiting for the seller's "yes", keyed by
	// user id. In memory on purpose: a draft lost to a restart just means
	// re-sending the message.
	mu     sync.Mutex
	drafts map[int64]draft
}

type draft struct {
	in      ListingInput
	created time.Time
}

// DraftTTL is how long a preview waits for yes / edit / cancel.
const DraftTTL = 10 * time.Minute

func NewBot(users domain.UserRepository, listings *Listings, appURL string) *Bot {
	return &Bot{users: users, listings: listings, appURL: strings.TrimRight(appURL, "/"), now: time.Now, drafts: map[int64]draft{}}
}

// GroupPrefix starts every bot command in a group chat; "/mp" is accepted
// as a shorthand.
const GroupPrefix = "/marketplace"

// chat carries what differs between a direct chat and a group.
type chat struct {
	group bool
	cmd   string // how users type a command here, e.g. `/marketplace sold 12`
}

var (
	directChat = chat{cmd: ""}
	groupChat  = chat{group: true, cmd: GroupPrefix + " "}
)

// Handle processes one direct-chat message and returns the reply.
func (b *Bot) Handle(ctx context.Context, fromPhone, text string) (string, error) {
	return b.handle(ctx, fromPhone, text, directChat)
}

// HandleGroup processes one group message. ok is false when the message
// isn't addressed to the bot, and then nothing should be sent. fromPhone is
// empty when WhatsApp hides the sender's number from the bot.
func (b *Bot) HandleGroup(ctx context.Context, fromPhone, text string) (reply string, ok bool, err error) {
	rest, ok := stripGroupPrefix(text)
	if !ok {
		return "", false, nil
	}
	if fromPhone == "" {
		return "I can't see your number in this group yet. Send me a direct message once (just \"hi\"), then try again here.", true, nil
	}
	reply, err = b.handle(ctx, fromPhone, rest, groupChat)
	return reply, true, err
}

func stripGroupPrefix(text string) (string, bool) {
	text = strings.TrimSpace(text)
	for _, p := range []string{GroupPrefix, "/mp"} {
		if len(text) >= len(p) && strings.EqualFold(text[:len(p)], p) {
			rest := text[len(p):]
			if rest == "" || rest[0] == ' ' || rest[0] == '\n' {
				return strings.TrimSpace(rest), true
			}
		}
	}
	return "", false
}

func (b *Bot) handle(ctx context.Context, fromPhone, text string, c chat) (string, error) {
	phone, err := NormalisePhone(fromPhone)
	if err != nil {
		return "", err
	}
	user, err := b.users.FindByPhone(ctx, phone)
	if errors.Is(err, domain.ErrNotFound) {
		if c.group {
			return fmt.Sprintf("Your WhatsApp number isn't linked to a Sloan Marketplace account yet. "+
				"Sign up with your MIT email at %s using this WhatsApp number, then try again.", b.appURL), nil
		}
		return fmt.Sprintf("Hi! This number isn't linked to a Sloan Marketplace account yet. "+
			"Sign up with your MIT email at %s and enter this WhatsApp number (+%s), then message me again.", b.appURL, phone), nil
	}
	if err != nil {
		return "", err
	}

	text = strings.TrimSpace(text)
	cmd, rest, bang := splitCommand(text)
	pending, hasDraft := b.draft(user.ID)
	switch cmd {
	case "sell", "post":
		if rest == "" && hasDraft {
			return b.confirm(ctx, user, pending, c)
		}
		return b.sell(ctx, user, rest, bang, c)
	case "yes", "y", "confirm", "ok", "go":
		if !hasDraft {
			return fmt.Sprintf("Nothing waiting to post. Send \"%spost ...\" with your tickets first.", c.cmd), nil
		}
		return b.confirm(ctx, user, pending, c)
	case "cancel", "no", "nevermind", "stop":
		b.dropDraft(user.ID)
		if !hasDraft {
			return "Nothing to cancel.", nil
		}
		return "Cancelled, nothing was posted.", nil
	case "mine", "my", "listings":
		return b.mine(ctx, user, c)
	case "sold", "remove", "delete", "withdraw":
		return b.close(ctx, user, cmd, rest, c)
	case "edit", "update", "change":
		// "edit price $50" with a draft pending changes the draft;
		// "edit 12 price $50" changes a posted listing.
		if hasDraft && !startsWithNumber(rest) {
			return b.editDraft(user, pending, rest, c)
		}
		return b.edit(ctx, user, rest, c)
	default:
		return b.help(user, c), nil
	}
}

func (b *Bot) draft(userID int64) (draft, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	d, ok := b.drafts[userID]
	if ok && b.now().Sub(d.created) > DraftTTL {
		delete(b.drafts, userID)
		return draft{}, false
	}
	return d, ok
}

func (b *Bot) setDraft(userID int64, in ListingInput) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.drafts[userID] = draft{in: in, created: b.now()}
}

func (b *Bot) dropDraft(userID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.drafts, userID)
}

func startsWithNumber(s string) bool {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	return s != "" && s[0] >= '0' && s[0] <= '9'
}

// sell parses the message. Normally it stores a draft and shows a preview
// to confirm, so the parser's guesses are checked before anything goes
// live; "sell!" / "post!" skips the preview.
func (b *Bot) sell(ctx context.Context, u *domain.User, body string, skipConfirm bool, c chat) (string, error) {
	in, err := ParseSellMessage(body, b.now())
	if err != nil {
		return fmt.Sprintf("I couldn't read that listing: %v\n\n%s", err, sellExample(c)), nil
	}
	if skipConfirm {
		return b.post(ctx, u, in, c)
	}
	in, verr := b.listings.validate(in)
	if verr != nil {
		return "I couldn't post that: " + strings.TrimPrefix(verr.Error(), ErrInvalidInput.Error()+": "), nil
	}
	b.setDraft(u.ID, in)
	return fmt.Sprintf("Here's what I'll post:\n%s\n\nReply *yes* to post it, *edit price $50* (or date, qty, event, venue, notes) to change something, or *cancel*.",
		previewOf(in)), nil
}

func (b *Bot) confirm(ctx context.Context, u *domain.User, d draft, c chat) (string, error) {
	b.dropDraft(u.ID)
	return b.post(ctx, u, d.in, c)
}

// editDraft changes one field of the pending draft and shows it again.
func (b *Bot) editDraft(u *domain.User, d draft, rest string, c chat) (string, error) {
	fields := strings.Fields(rest)
	if len(fields) < 2 {
		return "Tell me what to change, e.g. *edit price $50* or *edit qty 3*.", nil
	}
	in, msg := applyField(d.in, fields[0], strings.TrimSpace(rest[len(fields[0]):]), b.now())
	if msg != "" {
		return msg, nil
	}
	in, err := b.listings.validate(in)
	if err != nil {
		return "I couldn't change that: " + strings.TrimPrefix(err.Error(), ErrInvalidInput.Error()+": "), nil
	}
	b.setDraft(u.ID, in)
	return fmt.Sprintf("Updated:\n%s\n\nReply *yes* to post, *edit ...* to change more, or *cancel*.", previewOf(in)), nil
}

func (b *Bot) post(ctx context.Context, u *domain.User, in ListingInput, c chat) (string, error) {
	l, err := b.listings.Create(ctx, u.ID, domain.SourceWhatsApp, in)
	if errors.Is(err, ErrInvalidInput) {
		return fmt.Sprintf("I couldn't post that: %s\n\n%s", strings.TrimPrefix(err.Error(), ErrInvalidInput.Error()+": "), sellExample(c)), nil
	}
	if err != nil {
		return "", err
	}
	summary := summaryOf(l)
	link := fmt.Sprintf("%s/listings/%d", b.appURL, l.ID)
	if c.group {
		return fmt.Sprintf("✅ Listed #%d for %s: %s\n%s", l.ID, firstName(u.Name), summary, link), nil
	}
	return fmt.Sprintf("Posted #%d: %s\n%s\n\nShare it to a group (tap, pick the group, send):\n%s\n\nSend \"%ssold %d\" when they're gone.",
		l.ID, summary, link, b.shareLink(l, link), c.cmd, l.ID), nil
}

// shareLink opens WhatsApp's chat picker with the group post pre-written.
func (b *Bot) shareLink(l *domain.Listing, link string) string {
	text := fmt.Sprintf("🎟️ Selling: %s\n%s%s\n%d ticket%s · %s%s\n%s",
		l.EventName, l.EventDate.Format("Mon Jan 2"), venueSuffix(l.Venue),
		l.Quantity, plural(l.Quantity), formatPrice(l.PriceCents), eachSuffix(l.PriceCents), link)
	return "https://wa.me/?text=" + url.QueryEscape(text)
}

func previewOf(in ListingInput) string {
	s := fmt.Sprintf("• Event: %s\n• Date: %s\n• Price: %s%s\n• Tickets: %d",
		in.EventName, in.EventDate.Format("Mon Jan 2 2006"), formatPrice(in.PriceCents), eachSuffix(in.PriceCents), in.Quantity)
	if in.Venue != "" {
		s += "\n• Venue: " + in.Venue
	}
	if in.Notes != "" {
		s += "\n• Notes: " + in.Notes
	}
	return s
}

func summaryOf(l *domain.Listing) string {
	return fmt.Sprintf("%d × %s · %s · %s%s%s", l.Quantity, l.EventName, l.EventDate.Format("Mon Jan 2"),
		formatPrice(l.PriceCents), eachSuffix(l.PriceCents), venueSuffix(l.Venue))
}

func eachSuffix(cents int) string {
	if cents > 0 {
		return " each"
	}
	return ""
}

func venueSuffix(v string) string {
	if v != "" {
		return " · " + v
	}
	return ""
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func (b *Bot) mine(ctx context.Context, u *domain.User, c chat) (string, error) {
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
		return fmt.Sprintf("You have no active listings. Send \"%shelp\" to see how to post one.", c.cmd), nil
	}
	return "Your active listings:\n" + strings.Join(lines, "\n") + fmt.Sprintf("\n\nSend \"%ssold <number>\" to close one.", c.cmd), nil
}

func (b *Bot) close(ctx context.Context, u *domain.User, cmd, rest string, c chat) (string, error) {
	id, err := strconv.ParseInt(strings.TrimPrefix(strings.TrimSpace(rest), "#"), 10, 64)
	if err != nil {
		return fmt.Sprintf("Which listing? Send e.g. \"%ssold 12\". Send \"%smine\" to see your listing numbers.", c.cmd, c.cmd), nil
	}
	status := domain.StatusWithdrawn
	if cmd == "sold" {
		status = domain.StatusSold
	}
	l, err := b.listings.SetStatus(ctx, u.ID, id, status)
	if errors.Is(err, domain.ErrNotFound) {
		return fmt.Sprintf("You don't have a listing #%d. Send \"%smine\" to see yours.", id, c.cmd), nil
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Marked #%d (%s) as %s.", l.ID, l.EventName, status), nil
}

// edit changes one field of a listing: "edit 12 price $50", "edit 12 qty 3",
// "edit 12 date Nov 20", "edit 12 venue TD Garden", "edit 12 notes ...",
// "edit 12 event Bruins vs Rangers".
func (b *Bot) edit(ctx context.Context, u *domain.User, rest string, c chat) (string, error) {
	usage := fmt.Sprintf("To change a listing, send \"%sedit <number> <field> <new value>\", e.g.\n"+
		"%[1]sedit 12 price $50\n%[1]sedit 12 qty 3\n%[1]sedit 12 date Nov 20\n"+
		"Fields: event, date, price, qty, venue, notes. Send \"%[1]smine\" for your listing numbers.", c.cmd)

	fields := strings.Fields(rest)
	if len(fields) < 3 {
		return usage, nil
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(fields[0], "#"), 10, 64)
	if err != nil {
		return usage, nil
	}
	field := fields[1]
	value := strings.TrimSpace(rest[strings.Index(rest, fields[1])+len(fields[1]):])
	if fieldKey(field) == "" {
		return fmt.Sprintf("I don't know the field %q.\n\n%s", field, usage), nil
	}

	l, err := b.listings.Get(ctx, id)
	if err == nil && l.SellerID != u.ID {
		err = domain.ErrNotFound
	}
	if errors.Is(err, domain.ErrNotFound) {
		return fmt.Sprintf("You don't have a listing #%d. Send \"%smine\" to see yours.", id, c.cmd), nil
	}
	if err != nil {
		return "", err
	}

	in, msg := applyField(InputOf(l), field, value, b.now())
	if msg != "" {
		return msg, nil
	}

	l, err = b.listings.Update(ctx, u.ID, id, in)
	if errors.Is(err, ErrInvalidInput) {
		return "I couldn't change that: " + strings.TrimPrefix(err.Error(), ErrInvalidInput.Error()+": "), nil
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Updated #%d: %s", l.ID, summaryOf(l)), nil
}

// applyField sets one named field of in from the user's text. A non-empty
// msg is the reply to send instead when the value can't be read.
func applyField(in ListingInput, field, value string, now time.Time) (ListingInput, string) {
	switch fieldKey(field) {
	case "event":
		in.EventName = value
	case "venue":
		in.Venue = value
	case "notes":
		in.Notes = value
	case "date":
		d, err := ParseEventDate(value, now)
		if err != nil {
			return in, fmt.Sprintf("I couldn't read that: %v", err)
		}
		in.EventDate = d
	case "price":
		p, err := ParsePriceCents(value)
		if err != nil {
			return in, fmt.Sprintf("I couldn't read that: %v", err)
		}
		in.PriceCents = p
	case "qty":
		n, err := strconv.Atoi(strings.Fields(value + " x")[0])
		if err != nil {
			return in, fmt.Sprintf("Quantity %q isn't a number.", value)
		}
		in.Quantity = n
	default:
		return in, fmt.Sprintf("I don't know the field %q. Fields: event, date, price, qty, venue, notes.", field)
	}
	return in, ""
}

func (b *Bot) help(u *domain.User, c chat) string {
	return fmt.Sprintf("Hi %s! I post ticket listings to Sloan Marketplace.\n\n%s\n\n"+
		"Other commands:\n• %[3]smine: your active listings\n• %[3]sedit 12 price $50: change a field (event, date, price, qty, venue, notes)\n• %[3]ssold 12: mark #12 sold\n• %[3]sremove 12: take #12 down\n\nBrowse: %[4]s",
		firstName(u.Name), sellExample(c), c.cmd, b.appURL)
}

func sellExample(c chat) string {
	post := "sell"
	if c.group {
		post = c.cmd + "post"
	}
	return fmt.Sprintf(`To list tickets, just write it out with a date and price:
%[1]s Bruins tickets Sat 10/10 $80 each, 2 tickets

Or use fields:
%[1]s Bruins vs Rangers | Oct 10 | $80 | 2 | TD Garden | notes

I'll show a preview; reply yes to post it. (%[1]s! skips the preview.)`, post)
}

// ParseSellMessage reads a listing from the text after "sell". It accepts
// "Key: value" lines (event, date, price, qty, venue, notes, any order), a
// single "event | date | price | qty | venue | notes" line, or failing both,
// free text like "Bruins tickets Sat $80" (see ParseFreeText).
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
		if len(fields) == 0 {
			return ParseFreeText(body, now)
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
	if len(words) == 1 || (len(words) == 2 && (words[0] == "this" || words[0] == "next")) {
		if wd, ok := parseWeekday(words[len(words)-1]); ok {
			days := (int(wd) - int(today.Weekday()) + 7) % 7
			if words[0] == "next" && days == 0 {
				days = 7
			}
			return today.AddDate(0, 0, days), nil
		}
	}
	// Drop a leading weekday ("sat nov 14") and ordinal suffixes ("14th").
	if len(words) > 1 && isWeekday(words[0]) {
		words = words[1:]
	}
	for i, w := range words {
		if w == "sept" {
			words[i] = "sep"
		}
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
	_, ok := parseWeekday(w)
	return ok
}

// parseWeekday accepts full names and common abbreviations ("thu", "thurs").
func parseWeekday(w string) (time.Weekday, bool) {
	for d := time.Sunday; d <= time.Saturday; d++ {
		name := strings.ToLower(d.String())
		if w == name || (len(w) >= 3 && strings.HasPrefix(name, w)) {
			return d, true
		}
	}
	return 0, false
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

// splitCommand separates the first word from the rest. bang reports a
// trailing "!" on the command ("post!"), which skips the confirmation step.
func splitCommand(text string) (cmd, rest string, bang bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", "", false
	}
	bang = strings.HasSuffix(fields[0], "!")
	cmd = strings.ToLower(strings.Trim(fields[0], "!/.:"))
	rest = strings.TrimSpace(text[strings.Index(text, fields[0])+len(fields[0]):])
	rest = strings.TrimPrefix(rest, ":")
	return cmd, rest, bang
}

func firstName(name string) string {
	if f := strings.Fields(name); len(f) > 0 {
		return f[0]
	}
	return "there"
}
