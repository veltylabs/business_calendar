package tests

import (
	"testing"

	businesscalendar "github.com/veltylabs/business_calendar"
)

// TestNewHolidayClosesTheDayWithoutRecompile — CU-04. A holiday registered at
// runtime stops that date being bookable.
func TestNewHolidayClosesTheDayWithoutRecompile(t *testing.T) {
	m, _ := newModule(t, nil)
	seedWeek(t, m)

	// Wednesday is normally open.
	before, err := m.GetDayBounds(sep09_2026_Wed)
	if err != nil {
		t.Fatalf("GetDayBounds: %v", err)
	}
	if !before.Open {
		t.Fatalf("expected Wednesday open before the holiday, got %+v", before)
	}

	if err := m.AddHoliday(businesscalendar.Holiday{
		SpecificDate: sep09_2026_Wed,
		Name:         "Independence Day",
	}); err != nil {
		t.Fatalf("AddHoliday: %v", err)
	}

	// GetDayDetail, not GetDayBounds: ClosedBy is this module's own vocabulary
	// and only GetDayDetail carries it (see interfaces.go).
	after, err := m.GetDayDetail(sep09_2026_Wed)
	if err != nil {
		t.Fatalf("GetDayDetail after: %v", err)
	}
	if after.Bounds.Open || after.ClosedBy != businesscalendar.ClosedHoliday {
		t.Errorf("expected the day closed by holiday, got %+v", after)
	}
}

// TestRemovingHolidayReopensTheDay — CU-05. Removing a holiday flips the day
// back to open.
func TestRemovingHolidayReopensTheDay(t *testing.T) {
	m, ids := newModule(t, nil)
	seedWeek(t, m)

	h := businesscalendar.Holiday{SpecificDate: oct01_2026_Thu, Name: "Dia"}
	if err := m.AddHoliday(h); err != nil {
		t.Fatalf("AddHoliday: %v", err)
	}
	rows, err := m.ListHolidays(0, 0)
	if err != nil || len(rows) != 1 {
		t.Fatalf("ListHolidays: n=%d err=%v", len(rows), err)
	}
	_ = ids // the fake ids also drive holiday ids; the listed row carries them

	holidayID := rows[0].Id
	if err := m.RemoveHoliday(holidayID); err != nil {
		t.Fatalf("RemoveHoliday: %v", err)
	}
	after, err := m.GetDayBounds(oct01_2026_Thu)
	if err != nil {
		t.Fatalf("GetDayBounds: %v", err)
	}
	if !after.Open {
		t.Errorf("expected Thursday open again, got %+v", after)
	}
}

// TestMovingHolidayFlipsBothDays — CU-05. Moving a holiday (remove one date,
// add another) flips the old day back to open and closes the new one.
func TestMovingHolidayFlipsBothDays(t *testing.T) {
	m, _ := newModule(t, nil)
	seedWeek(t, m)

	if err := m.AddHoliday(businesscalendar.Holiday{
		SpecificDate: sep15_2026_Tue, Name: "Moveable holiday",
	}); err != nil {
		t.Fatalf("AddHoliday: %v", err)
	}
	rows, err := m.ListHolidays(0, 0)
	if err != nil || len(rows) != 1 {
		t.Fatalf("ListHolidays: n=%d err=%v", len(rows), err)
	}
	if err := m.RemoveHoliday(rows[0].Id); err != nil {
		t.Fatalf("RemoveHoliday: %v", err)
	}
	if err := m.AddHoliday(businesscalendar.Holiday{
		SpecificDate: sep16_2026_Wed, Name: "Moveable holiday",
	}); err != nil {
		t.Fatalf("AddHoliday moved: %v", err)
	}

	old, err := m.GetDayBounds(sep15_2026_Tue)
	if err != nil {
		t.Fatalf("GetDayBounds old: %v", err)
	}
	if !old.Open {
		t.Errorf("old date should be open after moving, got %+v", old)
	}
	nw, err := m.GetDayDetail(sep16_2026_Wed)
	if err != nil {
		t.Fatalf("GetDayDetail new: %v", err)
	}
	if nw.Bounds.Open || nw.ClosedBy != businesscalendar.ClosedHoliday {
		t.Errorf("new date should be closed by holiday, got %+v", nw)
	}
}

// TestDuplicateHolidayOnSameDateIsRejected — the anti-footgun from the plan: a
// second holiday on the same date would double-close a day and make
// RemoveHoliday look broken.
func TestDuplicateHolidayOnSameDateIsRejected(t *testing.T) {
	m, _ := newModule(t, nil)
	if err := m.AddHoliday(businesscalendar.Holiday{SpecificDate: oct01_2026_Thu, Name: "First"}); err != nil {
		t.Fatalf("AddHoliday: %v", err)
	}
	err := m.AddHoliday(businesscalendar.Holiday{SpecificDate: oct01_2026_Thu, Name: "Second"})
	if err != businesscalendar.ErrDuplicateDate {
		t.Errorf("expected ErrDuplicateDate, got %v", err)
	}
}

func TestRemoveHoliday_NotFound(t *testing.T) {
	m, _ := newModule(t, nil)
	if err := m.RemoveHoliday("does-not-exist"); err != businesscalendar.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
