package usecase

import (
	"time"
	_ "time/tzdata" // the runtime image has no zoneinfo
)

// Campus is the time zone event dates are meant in.
var Campus = func() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}
	return loc
}()

// campusDate returns t's calendar date in Cambridge as midnight UTC.
func campusDate(t time.Time) time.Time {
	y, m, d := t.In(Campus).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
