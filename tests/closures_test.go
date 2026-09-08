package tests

import (
	"testing"

	businesscalendar "github.com/veltylabs/business_calendar"
)

// TestLocalClosureIsDistinguishableFromHoliday — CU-06. A one-off closure is
// not a national holiday; the origin is distinguishable.
func TestLocalClosureIsDistinguishableFromHoliday(t *testing.T) {
	m, _ := newModule(t, nil)
	seedWeek(t, m)

	// Thursday: a local closure (maintenance). Friday: a legal holiday.
	if err := m.AddClosure(businesscalendar.Closure{
		SpecificDate: oct01_2026_Thu, Reason: "Facility maintenance",
	}); err != nil {
		t.Fatalf("AddClosure: %v", err)
	}
	if err := m.AddHoliday(businesscalendar.Holiday{
		SpecificDate: dec25_2026_Fri, Name: "Christmas",
	}); err != nil {
		t.Fatalf("AddHoliday: %v", err)
	}

	local, err := m.GetDayBounds(oct01_2026_Thu)
	if err != nil {
		t.Fatalf("GetDayBounds local: %v", err)
	}
	if local.Open || local.ClosedBy != businesscalendar.ClosedLocal {
		t.Errorf("expected a local closure, got %+v", local)
	}

	holiday, err := m.GetDayBounds(dec25_2026_Fri)
	if err != nil {
		t.Fatalf("GetDayBounds holiday: %v", err)
	}
	if holiday.Open || holiday.ClosedBy != businesscalendar.ClosedHoliday {
		t.Errorf("expected a holiday, got %+v", holiday)
	}
}

// TestGetDayBoundsPrecedenceClosureOverHolidayOverWeekly — CU-06. Resolution is
// Closure > Holiday > Weekly.
func TestGetDayBoundsPrecedenceClosureOverHolidayOverWeekly(t *testing.T) {
	m, _ := newModule(t, nil)
	seedWeek(t, m)

	// Saturday is closed weekly AND has both a holiday and a closure: the local
	// closure wins.
	if err := m.AddHoliday(businesscalendar.Holiday{
		SpecificDate: sep12_2026_Sat, Name: "A holiday on a weekend",
	}); err != nil {
		t.Fatalf("AddHoliday: %v", err)
	}
	if err := m.AddClosure(businesscalendar.Closure{
		SpecificDate: sep12_2026_Sat, Reason: "Painting",
	}); err != nil {
		t.Fatalf("AddClosure: %v", err)
	}
	bounds, err := m.GetDayBounds(sep12_2026_Sat)
	if err != nil {
		t.Fatalf("GetDayBounds: %v", err)
	}
	if bounds.Open || bounds.ClosedBy != businesscalendar.ClosedLocal {
		t.Errorf("closure must outrank holiday and weekly, got %+v", bounds)
	}

	// A plain weekly-closed day with no dated rows reports ClosedWeekly.
	sun, err := m.GetDayBounds(sep13_2026_Sun)
	if err != nil {
		t.Fatalf("GetDayBounds: %v", err)
	}
	if sun.Open || sun.ClosedBy != businesscalendar.ClosedWeekly {
		t.Errorf("expected weekly closure, got %+v", sun)
	}

	// A weekday with no business_hours row at all is also closed by default.
	empty, err := m.GetDayBounds(sep13_2026_Sun)
	if err != nil {
		t.Fatalf("GetDayBounds: %v", err)
	}
	if empty.Open || empty.ClosedBy != businesscalendar.ClosedWeekly {
		t.Errorf("expected closed by default, got %+v", empty)
	}
}

// TestDuplicateClosureOnSameDateIsRejected — same-kind date uniqueness, mirror
// of the holiday anti-footgun.
func TestDuplicateClosureOnSameDateIsRejected(t *testing.T) {
	m, _ := newModule(t, nil)
	if err := m.AddClosure(businesscalendar.Closure{SpecificDate: sep15_2026_Tue, Reason: "First"}); err != nil {
		t.Fatalf("AddClosure: %v", err)
	}
	err := m.AddClosure(businesscalendar.Closure{SpecificDate: sep15_2026_Tue, Reason: "Second"})
	if err != businesscalendar.ErrDuplicateDate {
		t.Errorf("expected ErrDuplicateDate, got %v", err)
	}
}

func TestRemoveClosure_NotFound(t *testing.T) {
	m, _ := newModule(t, nil)
	if err := m.RemoveClosure("does-not-exist"); err != businesscalendar.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// TestListClosures_RangeFilters — closures are listable within a bounded range.
func TestListClosures_RangeFilters(t *testing.T) {
	m, _ := newModule(t, nil)
	for _, d := range []int64{sep12_2026_Sat, sep13_2026_Sun, sep14_2026_Mon} {
		if err := m.AddClosure(businesscalendar.Closure{SpecificDate: d, Reason: "Maintenance"}); err != nil {
			t.Fatalf("AddClosure %d: %v", d, err)
		}
	}
	rows, err := m.ListClosures(sep13_2026_Sun, sep14_2026_Mon)
	if err != nil {
		t.Fatalf("ListClosures: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 closures in range, got %d", len(rows))
	}
}
