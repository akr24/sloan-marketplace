package usecase

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Patterns for pulling listing fields out of a casual message such as
// "BRUINS TICKETS 80 DOLLARS fri" or "selling 2 tix to the Sloan Gala 11/14 $40 each".
// Each match is cut out of the text; whatever is left becomes the event name.
var (
	reDollarSign = regexp.MustCompile(`(?i)\$\s?(\d{1,5}(?:\.\d{1,2})?)(?:\s*(?:each|ea|apiece|per ticket|a ticket|/ticket|/ea))?\b`)
	reDollarWord = regexp.MustCompile(`(?i)\b(\d{1,5}(?:\.\d{1,2})?)\s*(?:dollars?|bucks|usd)\b(?:\s*(?:each|ea|apiece|per ticket|a ticket))?`)
	reFree       = regexp.MustCompile(`(?i)\b(?:for )?free\b`)
	reQuantity   = regexp.MustCompile(`(?i)\b(\d{1,2})\s*(?:x\s*)?(?:tickets?|tix|seats?|passes)\b`)
	reQuantityX  = regexp.MustCompile(`(?i)(?:^|\s)[x×]\s?(\d{1,2})\b|\b(\d{1,2})\s?[x×](?:\s|$)`)
	rePair       = regexp.MustCompile(`(?i)\b(?:a )?pair of\b`)
	reTime       = regexp.MustCompile(`(?i)\b(?:at |@ ?)?\d{1,2}(?::\d{2})?\s?(?:am|pm)\b`)
	reBareNumber = regexp.MustCompile(`(?:^|\s)(\d{1,5}(?:\.\d{1,2})?)(?:\s|$)`)

	month   = `(?:jan|feb|mar|apr|may|jun|jul|aug|sep|sept|oct|nov|dec)[a-z]*\.?`
	weekday = `(?:mon|tue|tues|wed|thu|thur|thurs|fri|sat|sun)(?:day|sday|nesday|rsday|urday)?`
	reDates = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:today|tonight|tomorrow)\b`),
		regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`),
		regexp.MustCompile(`(?i)\b(?:` + weekday + `,?\s+)?\d{1,2}/\d{1,2}(?:/\d{2,4})?\b`),
		regexp.MustCompile(`(?i)\b(?:` + weekday + `,?\s+)?` + month + `\s+\d{1,2}(?:st|nd|rd|th)?(?:,?\s+\d{4})?\b`),
		regexp.MustCompile(`(?i)\b(?:` + weekday + `,?\s+)?\d{1,2}(?:st|nd|rd|th)?\s+` + month + `(?:\s+\d{4})?`),
		regexp.MustCompile(`(?i)\b(?:this |next )?` + weekday + `\b`),
	}

	// Words that frame a listing rather than name the event.
	reFiller = regexp.MustCompile(`(?i)\b(?:selling|for sale|available|obo|or best offer)\b`)
	edgeJunk = map[string]bool{"for": true, "on": true, "at": true, "each": true, "ea": true, "of": true, "-": true, "–": true, "—": true, "@": true, "to": true}
)

// ParseFreeText reads a listing from a casual one-line message. It needs at
// least an event and a price; quantity defaults to 1. A date is required too,
// because listings expire after their event, and the error says so plainly.
func ParseFreeText(text string, now time.Time) (ListingInput, error) {
	in := ListingInput{Quantity: 1}
	s := " " + strings.Join(strings.Fields(text), " ") + " "

	cut := func(loc []int) {
		s = s[:loc[0]] + " " + s[loc[1]:]
	}

	priceFound := false
	if m := reDollarSign.FindStringSubmatchIndex(s); m != nil {
		in.PriceCents, _ = ParsePriceCents(s[m[2]:m[3]])
		priceFound = true
		cut(m[:2])
	} else if m := reDollarWord.FindStringSubmatchIndex(s); m != nil {
		in.PriceCents, _ = ParsePriceCents(s[m[2]:m[3]])
		priceFound = true
		cut(m[:2])
	} else if m := reFree.FindStringIndex(s); m != nil {
		priceFound = true
		cut(m)
	}

	if m := reQuantity.FindStringSubmatchIndex(s); m != nil {
		in.Quantity, _ = strconv.Atoi(s[m[2]:m[3]])
		cut(m[:2])
	} else if m := reQuantityX.FindStringSubmatchIndex(s); m != nil {
		g := 2
		if m[2] < 0 {
			g = 4
		}
		in.Quantity, _ = strconv.Atoi(s[m[g]:m[g+1]])
		cut(m[:2])
	} else if m := rePair.FindStringIndex(s); m != nil {
		in.Quantity = 2
		cut(m)
	}

	if m := reTime.FindStringIndex(s); m != nil {
		in.Notes = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(s[m[0]:m[1]]), "at "), "@"))
		cut(m)
	}

	dateFound := false
	for _, re := range reDates {
		m := re.FindStringIndex(s)
		if m == nil {
			continue
		}
		d, err := ParseEventDate(s[m[0]:m[1]], now)
		if err != nil {
			continue
		}
		in.EventDate = d
		dateFound = true
		cut(m)
		break
	}

	// "Bruins tickets 80 fri": a lone leftover number is the price.
	if !priceFound {
		if all := reBareNumber.FindAllStringSubmatchIndex(s, -1); len(all) == 1 {
			m := all[0]
			in.PriceCents, _ = ParsePriceCents(s[m[2]:m[3]])
			priceFound = true
			cut([]int{m[2], m[3]})
		}
	}

	in.EventName = cleanEventName(reFiller.ReplaceAllString(s, " "))

	var missing []string
	if in.EventName == "" {
		missing = append(missing, "the event")
	}
	if !dateFound {
		missing = append(missing, "the date (e.g. Sat, Oct 10 or 10/10)")
	}
	if !priceFound {
		missing = append(missing, "the price (e.g. $80)")
	}
	if len(missing) > 0 {
		return in, errors.New("missing " + strings.Join(missing, " and "))
	}
	return in, nil
}

// cleanEventName trims connective words and punctuation left at either end
// once the other fields are cut out, and calms down ALL-CAPS names.
func cleanEventName(s string) string {
	words := strings.Fields(strings.NewReplacer(",", " ", ";", " ", "!", " ").Replace(s))
	for len(words) > 0 && edgeJunk[strings.ToLower(words[0])] {
		words = words[1:]
	}
	for len(words) > 0 && edgeJunk[strings.ToLower(words[len(words)-1])] {
		words = words[:len(words)-1]
	}
	name := strings.Trim(strings.Join(words, " "), " .:-–—")
	if name != "" && name == strings.ToUpper(name) && strings.ToLower(name) != name {
		parts := strings.Fields(strings.ToLower(name))
		for i, p := range parts {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
		name = strings.Join(parts, " ")
	}
	return name
}
