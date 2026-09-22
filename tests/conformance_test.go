package tests

import (
	"testing"

	businesscalendar "github.com/veltylabs/business_calendar"
	"webtyp.com/router/loopback"
	"webtyp.com/view"
)

// TestNewBusinessHoursView_ListsWeek drives the weekly-hours presenter through
// the real op stack (router/loopback) and checks it is save-capable but not
// deletable. English weekday labels are the canonical, untranslated form.
func TestNewBusinessHoursView_ListsWeek(t *testing.T) {
	m, _ := newModule(t, nil)
	seedWeek(t, m)

	p := businesscalendar.NewBusinessHoursView(loopback.New(m))
	var rerr error
	p.Reload(func(err error) { rerr = err })
	if rerr != nil {
		t.Fatalf("Reload: %v", rerr)
	}
	items := p.Items()
	if len(items) != 7 {
		t.Fatalf("expected 7 items, got %d", len(items))
	}
	if items[1].Label != "Monday" {
		t.Errorf("expected Monday label, got %q", items[1].Label)
	}
	if items[1].Description != "08:00–18:00" {
		t.Errorf("expected 08:00-18:00 description, got %q", items[1].Description)
	}
	if _, ok := p.(view.Saver); !ok {
		t.Error("business hours presenter must be save-capable (upsert)")
	}
	if _, ok := p.(view.Deleter); ok {
		t.Error("business hours presenter must not be delete-capable")
	}
}

// TestNewHolidaysView_CRUD — the holidays presenter is full CRUD (list + save +
// delete) and lists a seeded holiday through the real ops.
func TestNewHolidaysView_CRUD(t *testing.T) {
	m, _ := newModule(t, nil)
	if err := m.AddHoliday(businesscalendar.Holiday{SpecificDate: oct01_2026_Thu, Name: "Independence Day"}); err != nil {
		t.Fatalf("AddHoliday: %v", err)
	}

	p := businesscalendar.NewHolidaysView(loopback.New(m))
	if _, ok := p.(view.Saver); !ok {
		t.Error("holidays presenter must be save-capable")
	}
	if _, ok := p.(view.Deleter); !ok {
		t.Error("holidays presenter must be delete-capable")
	}
	var rerr error
	p.Reload(func(err error) { rerr = err })
	if rerr != nil {
		t.Fatalf("Reload: %v", rerr)
	}
	items := p.Items()
	if len(items) != 1 || items[0].Label != "Independence Day" {
		t.Errorf("expected the seeded holiday, got %+v", items)
	}
}

// TestNewClosuresView_CRUD — the closures presenter is full CRUD and lists a
// seeded closure through the real ops.
func TestNewClosuresView_CRUD(t *testing.T) {
	m, _ := newModule(t, nil)
	if err := m.AddClosure(businesscalendar.Closure{SpecificDate: sep15_2026_Tue, Reason: "Maintenance"}); err != nil {
		t.Fatalf("AddClosure: %v", err)
	}

	p := businesscalendar.NewClosuresView(loopback.New(m))
	if _, ok := p.(view.Saver); !ok {
		t.Error("closures presenter must be save-capable")
	}
	if _, ok := p.(view.Deleter); !ok {
		t.Error("closures presenter must be delete-capable")
	}
	var rerr error
	p.Reload(func(err error) { rerr = err })
	if rerr != nil {
		t.Fatalf("Reload: %v", rerr)
	}
	items := p.Items()
	if len(items) != 1 || items[0].Label != "Maintenance" {
		t.Errorf("expected the seeded closure, got %+v", items)
	}
}

// TestHolidayItem_DescriptionIsTheRealDate reproduces a real bug found while
// manually testing the "Feriados" screen: every seeded holiday showed
// "1969-12-31" instead of its real date, regardless of which date was
// actually stored.
//
// Root cause: Holiday.SpecificDate is stored in SECONDS (see model.go's own
// comment "midnight UTC, seconds", and AddHoliday's callers, which divide
// tinytime.ParseDate's nanosecond result by 1e9 before writing). But
// view.go's Item() calls tinytime.FormatDate(h.SpecificDate) directly —
// FormatDate takes UnixNANO (webtyp.com/time's own convention: Now() and
// ParseDate both return nanoseconds; FormatDate's own implementation does
// time.Unix(0, v), the nanosecond form of time.Unix). Passing seconds where
// nanoseconds are expected is off by 1e9: a real 2026 date (~1.7 billion
// seconds since epoch) is ~1.7 seconds in nanosecond terms — a few seconds
// after the Unix epoch — which a negative UTC offset (e.g. Chile, UTC-3/-4)
// pushes back across midnight into "1969-12-31" for every single row,
// independent of the real stored date.
func TestHolidayItem_DescriptionIsTheRealDate(t *testing.T) {
	m, _ := newModule(t, nil)

	// 2026-06-20 at midnight UTC, in SECONDS — exactly how AddHoliday's own
	// real callers compute it (tinytime.ParseDate(...) / 1e9).
	const specificDateSeconds = 1781913600 // 2026-06-20T00:00:00Z

	if err := m.AddHoliday(businesscalendar.Holiday{
		SpecificDate: specificDateSeconds,
		Name:         "Día Nacional de los Pueblos Indígenas",
	}); err != nil {
		t.Fatalf("AddHoliday: %v", err)
	}

	p := businesscalendar.NewHolidaysView(loopback.New(m))
	var rerr error
	p.Reload(func(err error) { rerr = err })
	if rerr != nil {
		t.Fatalf("Reload: %v", rerr)
	}
	items := p.Items()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Description != "2026-06-20" {
		t.Fatalf("holiday Description = %q, want %q — the date shown to an admin must be the "+
			"date they entered, not the Unix epoch misread by a factor of 1e9",
			items[0].Description, "2026-06-20")
	}
}
