package tests

import (
	"testing"

	businesscalendar "github.com/veltylabs/business_calendar"
)

// TestWeekdayLimits_RegularHours checks that a regular open day
// returns the configured Open, OpenMin, and CloseMin.
func TestWeekdayLimits_RegularHours(t *testing.T) {
	m, _ := newModule(t, nil)
	seedWeek(t, m)

	// Monday (1) is seeded as open 08:00-18:00.
	bounds, err := m.WeekdayLimits(1)
	if err != nil {
		t.Fatalf("WeekdayLimits: %v", err)
	}
	if !bounds.Open || bounds.OpenMin != 480 || bounds.CloseMin != 1080 {
		t.Errorf("expected open 08:00-18:00, got %+v", bounds)
	}
}

// TestWeekdayLimits_ClosedDay checks that a day with IsOpen: false
// returns {Open: false} and a day with no rows returns {Open: false}.
func TestWeekdayLimits_ClosedDay(t *testing.T) {
	m, _ := newModule(t, nil)

	// No rows initialized yet. Day 0 (Sunday) should be closed.
	emptyBounds, err := m.WeekdayLimits(0)
	if err != nil {
		t.Fatalf("WeekdayLimits: %v", err)
	}
	if emptyBounds.Open {
		t.Errorf("expected empty day to be closed, got %+v", emptyBounds)
	}

	seedWeek(t, m)
	// Sunday (0) is seeded as closed in seedWeek.
	closedBounds, err := m.WeekdayLimits(0)
	if err != nil {
		t.Fatalf("WeekdayLimits: %v", err)
	}
	if closedBounds.Open {
		t.Errorf("expected explicitly closed day to be closed, got %+v", closedBounds)
	}
}

// TestWeekdayLimits_InvalidWeekday checks that out-of-range values like -1 and 7
// return ErrInvalidWeekday.
func TestWeekdayLimits_InvalidWeekday(t *testing.T) {
	m, _ := newModule(t, nil)

	_, err := m.WeekdayLimits(-1)
	if err == nil || err.Error() != businesscalendar.ErrInvalidWeekday.Error() {
		t.Errorf("expected ErrInvalidWeekday, got %v", err)
	}

	_, err = m.WeekdayLimits(7)
	if err == nil || err.Error() != businesscalendar.ErrInvalidWeekday.Error() {
		t.Errorf("expected ErrInvalidWeekday, got %v", err)
	}
}

// TestWeekdayLimits_IsolatedFromHolidaysAndClosures adding a holiday and closure
// and asserting that WeekdayLimits ignores them while DayLimits properly blocks them.
func TestWeekdayLimits_IsolatedFromHolidaysAndClosures(t *testing.T) {
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

	// WeekdayLimits should ignore both the holiday and closure and say the day is open.
	monBounds, err := m.WeekdayLimits(1)
	if err != nil {
		t.Fatalf("WeekdayLimits Monday: %v", err)
	}
	if !monBounds.Open {
		t.Errorf("expected WeekdayLimits for Monday to ignore holiday and be open, got %+v", monBounds)
	}

	tueBounds, err := m.WeekdayLimits(2)
	if err != nil {
		t.Fatalf("WeekdayLimits Tuesday: %v", err)
	}
	if !tueBounds.Open {
		t.Errorf("expected WeekdayLimits for Tuesday to ignore closure and be open, got %+v", tueBounds)
	}

	// DayLimits should reflect the closures.
	monDayLimits, err := m.DayLimits(sep14_2026_Mon)
	if err != nil {
		t.Fatalf("DayLimits Monday: %v", err)
	}
	if monDayLimits.Open {
		t.Errorf("expected DayLimits for Monday to be closed by holiday, got %+v", monDayLimits)
	}

	tueDayLimits, err := m.DayLimits(sep15_2026_Tue)
	if err != nil {
		t.Fatalf("DayLimits Tuesday: %v", err)
	}
	if tueDayLimits.Open {
		t.Errorf("expected DayLimits for Tuesday to be closed by closure, got %+v", tueDayLimits)
	}
}
