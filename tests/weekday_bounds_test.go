package tests

import (
	"testing"

	businesscalendar "github.com/veltylabs/business_calendar"
)

// TestGetWeekdayBounds_RegularHours checks that a regular open day
// returns the configured Open, OpenMin, and CloseMin.
func TestGetWeekdayBounds_RegularHours(t *testing.T) {
	m, _ := newModule(t, nil)
	seedWeek(t, m)

	// Monday (1) is seeded as open 480-1080.
	bounds, err := m.GetWeekdayBounds(1)
	if err != nil {
		t.Fatalf("GetWeekdayBounds: %v", err)
	}
	if !bounds.Open || bounds.OpenMin != 480 || bounds.CloseMin != 1080 {
		t.Errorf("expected open 480-1080, got %+v", bounds)
	}
}

// TestGetWeekdayBounds_ClosedDay checks that a day with IsOpen: false
// returns {Open: false} and a day with no rows returns {Open: false}.
func TestGetWeekdayBounds_ClosedDay(t *testing.T) {
	m, _ := newModule(t, nil)

	// No rows initialized yet. Day 0 (Sunday) should be closed.
	emptyBounds, err := m.GetWeekdayBounds(0)
	if err != nil {
		t.Fatalf("GetWeekdayBounds: %v", err)
	}
	if emptyBounds.Open {
		t.Errorf("expected empty day to be closed, got %+v", emptyBounds)
	}

	seedWeek(t, m)
	// Sunday (0) is seeded as closed in seedWeek.
	closedBounds, err := m.GetWeekdayBounds(0)
	if err != nil {
		t.Fatalf("GetWeekdayBounds: %v", err)
	}
	if closedBounds.Open {
		t.Errorf("expected explicitly closed day to be closed, got %+v", closedBounds)
	}
}

// TestGetWeekdayBounds_InvalidWeekday checks that out-of-range values like -1 and 7
// return ErrInvalidWeekday.
func TestGetWeekdayBounds_InvalidWeekday(t *testing.T) {
	m, _ := newModule(t, nil)

	_, err := m.GetWeekdayBounds(-1)
	if err == nil || err.Error() != businesscalendar.ErrInvalidWeekday.Error() {
		t.Errorf("expected ErrInvalidWeekday, got %v", err)
	}

	_, err = m.GetWeekdayBounds(7)
	if err == nil || err.Error() != businesscalendar.ErrInvalidWeekday.Error() {
		t.Errorf("expected ErrInvalidWeekday, got %v", err)
	}
}

// TestGetWeekdayBounds_IsolatedFromHolidaysAndClosures adding a holiday and closure
// and asserting that GetWeekdayBounds ignores them while GetDayBounds properly blocks them.
func TestGetWeekdayBounds_IsolatedFromHolidaysAndClosures(t *testing.T) {
	m, _ := newModule(t, nil)
	seedWeek(t, m)

	// sep14_2026_Mon is a Monday. Monday is open.
	// Add a holiday for that specific Monday.
	if err := m.AddHoliday(businesscalendar.Holiday{
		SpecificDate: sep14_2026_Mon,
		Name:         "Monday Holiday",
	}); err != nil {
		t.Fatalf("AddHoliday: %v", err)
	}

	// Add a closure for Tuesday.
	if err := m.AddClosure(businesscalendar.Closure{
		SpecificDate: sep15_2026_Tue,
		Reason:       "Tuesday Closure",
	}); err != nil {
		t.Fatalf("AddClosure: %v", err)
	}

	// GetWeekdayBounds should ignore both the holiday and closure and say the day is open.
	monBounds, err := m.GetWeekdayBounds(1)
	if err != nil {
		t.Fatalf("GetWeekdayBounds Monday: %v", err)
	}
	if !monBounds.Open {
		t.Errorf("expected GetWeekdayBounds for Monday to ignore holiday and be open, got %+v", monBounds)
	}

	tueBounds, err := m.GetWeekdayBounds(2)
	if err != nil {
		t.Fatalf("GetWeekdayBounds Tuesday: %v", err)
	}
	if !tueBounds.Open {
		t.Errorf("expected GetWeekdayBounds for Tuesday to ignore closure and be open, got %+v", tueBounds)
	}

	// GetDayBounds should reflect the closures.
	monDayBounds, err := m.GetDayBounds(sep14_2026_Mon)
	if err != nil {
		t.Fatalf("GetDayBounds Monday: %v", err)
	}
	if monDayBounds.Open {
		t.Errorf("expected GetDayBounds for Monday to be closed by holiday, got %+v", monDayBounds)
	}

	tueDayBounds, err := m.GetDayBounds(sep15_2026_Tue)
	if err != nil {
		t.Fatalf("GetDayBounds Tuesday: %v", err)
	}
	if tueDayBounds.Open {
		t.Errorf("expected GetDayBounds for Tuesday to be closed by closure, got %+v", tueDayBounds)
	}
}
