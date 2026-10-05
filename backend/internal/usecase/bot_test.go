package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/akr24/sloan-marketplace/backend/internal/domain"
)

// Monday 5 Oct 2026, mid-afternoon in Cambridge.
var testNow = time.Date(2026, 10, 5, 19, 0, 0, 0, time.UTC)

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestParseEventDate(t *testing.T) {
	cases := map[string]time.Time{
		"2026-11-14":        date(2026, 11, 14),
		"11/14":             date(2026, 11, 14),
		"11/14/26":          date(2026, 11, 14),
		"Nov 14":            date(2026, 11, 14),
		"nov 14th":          date(2026, 11, 14),
		"Sat, Nov 14":       date(2026, 11, 14),
		"November 14, 2026": date(2026, 11, 14),
		"14 Nov":            date(2026, 11, 14),
		"Jan 10":            date(2027, 1, 10), // already passed this year
		"today":             date(2026, 10, 5),
		"tomorrow":          date(2026, 10, 6),
	}
	for in, want := range cases {
		got, err := ParseEventDate(in, testNow)
		if err != nil || !got.Equal(want) {
			t.Errorf("ParseEventDate(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "soon", "13/45"} {
		if _, err := ParseEventDate(bad, testNow); err == nil {
			t.Errorf("ParseEventDate(%q) should fail", bad)
		}
	}
}

func TestParsePriceCents(t *testing.T) {
	cases := map[string]int{"$40": 4000, "40": 4000, "40.50": 4050, "$1,200": 120000, "free": 0, "$35 each": 3500}
	for in, want := range cases {
		got, err := ParsePriceCents(in)
		if err != nil || got != want {
			t.Errorf("ParsePriceCents(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	if _, err := ParsePriceCents("cheap"); err == nil {
		t.Error("ParsePriceCents(cheap) should fail")
	}
}

func TestParseSellMessageForms(t *testing.T) {
	keyed := "\nEvent: Sloan Fall Ball\nDate: Nov 14\nPrice: $40\nQty: 2\nVenue: Boston Harbor Hotel\nNotes: doors 7:30, black tie"
	oneLine := "Sloan Fall Ball | Nov 14 | $40 | 2 | Boston Harbor Hotel | doors 7:30, black tie"

	for name, body := range map[string]string{"keyed": keyed, "one line": oneLine} {
		in, err := ParseSellMessage(body, testNow)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want := ListingInput{
			EventName: "Sloan Fall Ball", EventDate: date(2026, 11, 14), Venue: "Boston Harbor Hotel",
			Quantity: 2, PriceCents: 4000, Notes: "doors 7:30, black tie",
		}
		if in != want {
			t.Errorf("%s:\n got %+v\nwant %+v", name, in, want)
		}
	}
}

func TestParseSellMessageDefaultsAndErrors(t *testing.T) {
	in, err := ParseSellMessage("Gala | Nov 1 | 25", testNow)
	if err != nil || in.Quantity != 1 {
		t.Fatalf("quantity should default to 1: %+v, %v", in, err)
	}
	if _, err := ParseSellMessage("Date: Nov 1\nPrice: 10", testNow); err == nil || !strings.Contains(err.Error(), "event") {
		t.Errorf("missing event: got %v", err)
	}
	if _, err := ParseSellMessage("Gala | Nov 1", testNow); err == nil || !strings.Contains(err.Error(), "price") {
		t.Errorf("missing price: got %v", err)
	}
}

func newTestBot(t *testing.T) (*Bot, *fakeListings) {
	t.Helper()
	users := &fakeUsers{}
	_ = users.Create(context.Background(), &domain.User{Email: "tim@mit.edu", Name: "Tim Beaver", WhatsAppPhone: "16175550123"})
	repo := &fakeListings{users: users}
	l := NewListings(repo)
	l.now = func() time.Time { return testNow }
	b := NewBot(users, l, "https://market.example/")
	b.now = func() time.Time { return testNow }
	return b, repo
}

func TestBotPostsAndClosesListing(t *testing.T) {
	ctx := context.Background()
	b, repo := newTestBot(t)

	reply, err := b.Handle(ctx, "16175550123", "SELL Sloan Fall Ball | Nov 14 | $40 | 2")
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.rows) != 1 || repo.rows[0].Source != domain.SourceWhatsApp {
		t.Fatalf("listing not stored: %+v", repo.rows)
	}
	if !strings.Contains(reply, "Posted #1") || !strings.Contains(reply, "https://market.example/listings/1") {
		t.Errorf("unexpected reply: %s", reply)
	}

	reply, _ = b.Handle(ctx, "16175550123", "mine")
	if !strings.Contains(reply, "#1 2 × Sloan Fall Ball") {
		t.Errorf("mine reply: %s", reply)
	}

	reply, _ = b.Handle(ctx, "16175550123", "sold 1")
	if repo.rows[0].Status != domain.StatusSold {
		t.Errorf("status = %s, reply %s", repo.rows[0].Status, reply)
	}
}

func TestBotUnknownSenderAndBadInput(t *testing.T) {
	ctx := context.Background()
	b, repo := newTestBot(t)

	reply, _ := b.Handle(ctx, "447911123456", "sell x | nov 1 | 5")
	if !strings.Contains(reply, "isn't linked") || len(repo.rows) != 0 {
		t.Errorf("unknown sender: %s", reply)
	}

	reply, _ = b.Handle(ctx, "16175550123", "sell Gala | yesterday-ish | $5")
	if !strings.Contains(reply, "couldn't read") {
		t.Errorf("bad date: %s", reply)
	}

	reply, _ = b.Handle(ctx, "16175550123", "sell Gala | 2026-10-01 | $5")
	if !strings.Contains(reply, "in the past") {
		t.Errorf("past date: %s", reply)
	}

	reply, _ = b.Handle(ctx, "16175550123", "hello")
	if !strings.Contains(reply, "Hi Tim") {
		t.Errorf("help: %s", reply)
	}
}

func TestSellerCannotCloseOthersListing(t *testing.T) {
	ctx := context.Background()
	b, repo := newTestBot(t)
	_ = b.users.(*fakeUsers).Create(ctx, &domain.User{Email: "o@mit.edu", Name: "Other", WhatsAppPhone: "16175550999"})
	_, _ = b.Handle(ctx, "16175550123", "sell Gala | Nov 1 | 5")

	reply, _ := b.Handle(ctx, "16175550999", "sold 1")
	if repo.rows[0].Status != domain.StatusActive || !strings.Contains(reply, "don't have") {
		t.Errorf("other seller closed listing: %s", reply)
	}
}

func TestBotInGroup(t *testing.T) {
	ctx := context.Background()
	b, repo := newTestBot(t)

	if _, ok, _ := b.HandleGroup(ctx, "16175550123", "who has bruins tix?"); ok {
		t.Fatal("bot answered ordinary group chatter")
	}

	reply, ok, err := b.HandleGroup(ctx, "16175550123", "/marketplace post BRUINS TICKETS 80 DOLLARS fri")
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	if len(repo.rows) != 1 || !strings.Contains(reply, "Listed #1 for Tim") || !strings.Contains(reply, "/listings/1") {
		t.Errorf("group post: %s", reply)
	}

	reply, _, _ = b.HandleGroup(ctx, "16175550123", "/marketplace mine")
	if !strings.Contains(reply, `"/marketplace sold <number>"`) {
		t.Errorf("group commands should use the prefix: %s", reply)
	}

	reply, _, _ = b.HandleGroup(ctx, "447911123456", "/mp post Gala fri $5")
	if strings.Contains(reply, "447911123456") || !strings.Contains(reply, "isn't linked") {
		t.Errorf("group reply must not reveal the number: %s", reply)
	}

	reply, _, _ = b.HandleGroup(ctx, "", "/marketplace post Gala fri $5")
	if !strings.Contains(reply, "direct message") {
		t.Errorf("hidden sender: %s", reply)
	}
}

func TestBotDirectFreeText(t *testing.T) {
	b, repo := newTestBot(t)
	reply, _ := b.Handle(context.Background(), "16175550123", "sell BRUINS TICKETS 80 DOLLARS fri")
	if len(repo.rows) != 1 || repo.rows[0].EventName != "Bruins Tickets" {
		t.Errorf("free text DM: %s", reply)
	}
}
