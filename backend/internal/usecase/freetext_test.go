package usecase

import (
	"strings"
	"testing"
)

func TestParseFreeText(t *testing.T) {
	// testNow is Monday 5 Oct 2026.
	cases := map[string]ListingInput{
		"BRUINS TICKETS 80 DOLLARS fri":                  {EventName: "Bruins Tickets", EventDate: date(2026, 10, 9), PriceCents: 8000, Quantity: 1},
		"Bruins tickets Sat 10/10 $80 each, 2 tickets":   {EventName: "Bruins tickets", EventDate: date(2026, 10, 10), PriceCents: 8000, Quantity: 2},
		"selling 2 tix to the Sloan Gala 11/14 $40 each": {EventName: "the Sloan Gala", EventDate: date(2026, 11, 14), PriceCents: 4000, Quantity: 2},
		"Red Sox vs Yankees tomorrow 7pm x4 $60":         {EventName: "Red Sox vs Yankees", EventDate: date(2026, 10, 6), PriceCents: 6000, Quantity: 4, Notes: "7pm"},
		"pair of Celtics tix Nov 3rd 150 bucks":          {EventName: "Celtics tix", EventDate: date(2026, 11, 3), PriceCents: 15000, Quantity: 2},
		"Sloan Fall Ball tickets Sat, Nov 14 for free":   {EventName: "Sloan Fall Ball tickets", EventDate: date(2026, 11, 14), PriceCents: 0, Quantity: 1},
		"Bruins tickets 80 fri":                          {EventName: "Bruins tickets", EventDate: date(2026, 10, 9), PriceCents: 8000, Quantity: 1},
		"C-Function Gala Sept 5 $25":                     {EventName: "C-Function Gala", EventDate: date(2027, 9, 5), PriceCents: 2500, Quantity: 1},
	}
	for text, want := range cases {
		got, err := ParseFreeText(text, testNow)
		if err != nil {
			t.Errorf("%q: %v", text, err)
			continue
		}
		if got != want {
			t.Errorf("%q:\n got %+v\nwant %+v", text, got, want)
		}
	}
}

func TestParseFreeTextNamesWhatsMissing(t *testing.T) {
	_, err := ParseFreeText("BRUINS TICKETS 80 DOLLARS", testNow)
	if err == nil || !strings.Contains(err.Error(), "date") || strings.Contains(err.Error(), "price") {
		t.Errorf("want only the date reported missing, got %v", err)
	}
	_, err = ParseFreeText("Bruins tickets fri", testNow)
	if err == nil || !strings.Contains(err.Error(), "price") {
		t.Errorf("want price reported missing, got %v", err)
	}
}

func TestStripGroupPrefix(t *testing.T) {
	for in, want := range map[string]string{
		"/marketplace post Bruins": "post Bruins",
		"/Marketplace":             "",
		"  /mp mine":               "mine",
	} {
		got, ok := stripGroupPrefix(in)
		if !ok || got != want {
			t.Errorf("stripGroupPrefix(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"anyone selling bruins tix?", "/marketplaces", "/mpx post", "hey /marketplace post"} {
		if _, ok := stripGroupPrefix(in); ok {
			t.Errorf("stripGroupPrefix(%q) should not match", in)
		}
	}
}
