package tests

import (
	"testing"

	businesscalendar "github.com/veltylabs/business_calendar"
)

// TestEventCalendarChanged_EveryMutation — each successful write publishes
// EventCalendarChanged with a fully populated CalendarChangedPayload; the
// Closed direction is the one a subscriber trusts (CU-03/CU-04/CU-29).
func TestEventCalendarChanged_EveryMutation(t *testing.T) {
	pub := newRecording()
	m, _ := newModule(t, pub)

	// A brand-new open weekday only opens time: Closed false.
	if err := m.UpsertBusinessHours(businesscalendar.BusinessHours{
		DayOfWeek: 1, OpenMin: 480, CloseMin: 1080, IsOpen: true,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	// Narrowing the same day closes time: Closed true.
	if err := m.UpsertBusinessHours(businesscalendar.BusinessHours{
		DayOfWeek: 1, OpenMin: 540, CloseMin: 960, IsOpen: true,
	}); err != nil {
		t.Fatalf("upsert narrow: %v", err)
	}

	// A holiday closes its day.
	if err := m.AddHoliday(businesscalendar.Holiday{SpecificDate: oct01_2026_Thu, Name: "H"}); err != nil {
		t.Fatalf("AddHoliday: %v", err)
	}
	// A closure closes its day.
	if err := m.AddClosure(businesscalendar.Closure{SpecificDate: dec25_2026_Fri, Reason: "R"}); err != nil {
		t.Fatalf("AddClosure: %v", err)
	}

	if len(pub.Events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(pub.Events))
	}

	assert := func(i int, topic, kind string, closed bool, from, to int64) {
		t.Helper()
		e := pub.Events[i]
		if e.Topic != businesscalendar.EventCalendarChanged {
			t.Errorf("event %d: expected topic %q, got %q", i, businesscalendar.EventCalendarChanged, e.Topic)
		}
		p, ok := e.Payload.(*businesscalendar.CalendarChangedPayload)
		if !ok {
			t.Fatalf("event %d: payload type %T", i, e.Payload)
		}
		if p.Kind != kind || p.Closed != closed || p.FromDate != from || p.ToDate != to {
			t.Errorf("event %d: got %+v (want kind=%q closed=%v from=%d to=%d)", i, p, kind, closed, from, to)
		}
	}

	assert(0, businesscalendar.EventCalendarChanged, businesscalendar.CalendarKindBusinessHours, false, 0, 0)
	assert(1, businesscalendar.EventCalendarChanged, businesscalendar.CalendarKindBusinessHours, true, 0, 0)
	assert(2, businesscalendar.EventCalendarChanged, businesscalendar.CalendarKindHoliday, true, oct01_2026_Thu, oct01_2026_Thu)
	assert(3, businesscalendar.EventCalendarChanged, businesscalendar.CalendarKindClosure, true, dec25_2026_Fri, dec25_2026_Fri)
}

// TestEventCalendarChanged_RemovalOpens — removing a dated closure opens time.
func TestEventCalendarChanged_RemovalOpens(t *testing.T) {
	pub := newRecording()
	m, _ := newModule(t, pub)

	if err := m.AddHoliday(businesscalendar.Holiday{SpecificDate: oct01_2026_Thu, Name: "H"}); err != nil {
		t.Fatalf("AddHoliday: %v", err)
	}
	rows, err := m.ListHolidays(0, 0)
	if err != nil || len(rows) != 1 {
		t.Fatalf("ListHolidays: n=%d err=%v", len(rows), err)
	}
	if err := m.RemoveHoliday(rows[0].Id); err != nil {
		t.Fatalf("RemoveHoliday: %v", err)
	}

	if len(pub.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(pub.Events))
	}
	p := pub.Events[1].Payload.(*businesscalendar.CalendarChangedPayload)
	if p.Kind != businesscalendar.CalendarKindHoliday || p.Closed {
		t.Errorf("removal must open time (Closed false), got %+v", p)
	}
}
