// Package whatsappweb runs the bot as a linked device on an ordinary WhatsApp
// account, the way WhatsApp Web does, using the whatsmeow library. Unlike
// Meta's Cloud API (package whatsapp) this lets the bot sit in the existing
// Sloan group chats and answer "/marketplace ..." commands there.
//
// It is unofficial and against WhatsApp's terms of service, so the account
// can be banned: use a dedicated number, and keep the Cloud API bot as the
// fallback. The session (keys, contacts, LID mappings) is stored in Postgres
// in whatsmeow's own tables, so only one process may run it at a time.
package whatsappweb

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// Bot is the slice of usecase.Bot this adapter drives.
type Bot interface {
	Handle(ctx context.Context, fromPhone, text string) (string, error)
	HandleGroup(ctx context.Context, fromPhone, text string) (reply string, ok bool, err error)
}

// maxAge drops messages older than this, which whatsmeow replays after a
// reconnect, so the bot doesn't answer commands from hours ago.
const maxAge = 10 * time.Minute

// Open loads (or creates) the stored device and returns a client for it.
// The client is not connected yet.
func Open(ctx context.Context, db *sql.DB) (*whatsmeow.Client, error) {
	container := sqlstore.NewWithDB(db, "postgres", waLog.Stdout("whatsapp-web/store", "WARN", false))
	if err := container.Upgrade(ctx); err != nil {
		return nil, fmt.Errorf("whatsapp-web: migrate store: %w", err)
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("whatsapp-web: load device: %w", err)
	}
	return whatsmeow.NewClient(device, waLog.Stdout("whatsapp-web", "WARN", false)), nil
}

// Linked reports whether the stored device has been paired with a phone.
func Linked(c *whatsmeow.Client) bool {
	return c.Store.ID != nil
}

// Link pairs the client with the WhatsApp account on phone using an 8-character
// code entered on that phone. onCode is called with the code; Link then
// waits for the phone to confirm. WhatsApp allows about 160 seconds.
func Link(ctx context.Context, c *whatsmeow.Client, phone string, onCode func(code string)) error {
	qr, err := c.GetQRChannel(ctx)
	if err != nil {
		return err
	}
	if err := c.Connect(); err != nil {
		return err
	}
	// The first QR event means the login socket is ready for a pairing code.
	first := <-qr
	if first.Event != whatsmeow.QRChannelEventCode {
		return fmt.Errorf("whatsapp-web: unexpected pairing event %q: %v", first.Event, first.Error)
	}
	code, err := c.PairPhone(ctx, phone, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		return fmt.Errorf("whatsapp-web: request pairing code: %w", err)
	}
	onCode(code)

	for item := range qr {
		switch {
		case item == whatsmeow.QRChannelSuccess:
			return nil
		case item.Event == whatsmeow.QRChannelEventCode:
			// New QR codes keep arriving while we wait; the pairing code stays valid.
		default:
			return fmt.Errorf("whatsapp-web: pairing ended: %s %v", item.Event, item.Error)
		}
	}
	return fmt.Errorf("whatsapp-web: pairing timed out")
}

// Listener answers bot commands arriving on a linked client.
type Listener struct {
	client *whatsmeow.Client
	bot    Bot
}

// Listen registers the message handler and connects. Call Close on shutdown.
func Listen(c *whatsmeow.Client, bot Bot) (*Listener, error) {
	l := &Listener{client: c, bot: bot}
	c.AddEventHandler(l.onEvent)
	if err := c.Connect(); err != nil {
		return nil, fmt.Errorf("whatsapp-web: connect: %w", err)
	}
	return l, nil
}

func (l *Listener) Close() {
	l.client.Disconnect()
}

func (l *Listener) onEvent(evt any) {
	switch v := evt.(type) {
	case *events.Connected:
		log.Printf("whatsapp-web: connected as %s", l.client.Store.ID.User)
	case *events.LoggedOut:
		log.Printf("whatsapp-web: logged out by the phone (%v); run `make wa-link` to link again", v.Reason)
	case *events.Message:
		if v.Info.IsFromMe || time.Since(v.Info.Timestamp) > maxAge {
			return
		}
		text := v.Message.GetConversation()
		if text == "" {
			text = v.Message.GetExtendedTextMessage().GetText()
		}
		if text == "" {
			return
		}
		go l.handle(v, text)
	}
}

func (l *Listener) handle(m *events.Message, text string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	phone := l.senderPhone(ctx, m.Info)
	var reply string
	var err error
	if m.Info.IsGroup {
		var ok bool
		reply, ok, err = l.bot.HandleGroup(ctx, phone, text)
		if !ok {
			return
		}
	} else {
		if phone == "" {
			log.Printf("whatsapp-web: can't resolve the number behind %s", m.Info.Sender)
			return
		}
		reply, err = l.bot.Handle(ctx, phone, text)
	}
	if err != nil {
		log.Printf("whatsapp-web: handling message from %s: %v", m.Info.Sender, err)
		reply = "Sorry, something went wrong on our side. Please try again in a minute."
	}

	// Quote the command so it's clear in a busy group what the reply answers.
	_, err = l.client.SendMessage(ctx, m.Info.Chat, &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(reply),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      proto.String(m.Info.ID),
				Participant:   proto.String(m.Info.Sender.ToNonAD().String()),
				QuotedMessage: m.Message,
			},
		},
	})
	if err != nil {
		log.Printf("whatsapp-web: reply to %s: %v", m.Info.Chat, err)
	}
}

// senderPhone finds the sender's phone number. Groups increasingly address
// members by LID, an opaque id that hides the number, so fall back to the
// alternate address on the message and then to the LID↔number mappings
// whatsmeow learns (for example from a direct message).
func (l *Listener) senderPhone(ctx context.Context, info types.MessageInfo) string {
	for _, j := range []types.JID{info.Sender, info.SenderAlt} {
		if j.Server == types.DefaultUserServer && j.User != "" {
			return j.User
		}
	}
	if info.Sender.Server == types.HiddenUserServer {
		pn, err := l.client.Store.LIDs.GetPNForLID(ctx, info.Sender.ToNonAD())
		if err == nil && !pn.IsEmpty() {
			return pn.User
		}
	}
	return ""
}
