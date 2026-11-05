package tests

import (
	"reflect"
	"testing"

	"github.com/veltylabs/business_calendar"
	"webtyp.com/time"
)

func TestClosedDates(t *testing.T) {
	m, _ := newModule(t, nil)
	seedWeek(t, m)

	// Add holidays and closures as per specification
	h1, _ := time.ParseDateUTC("2026-09-18")
	if err := m.AddHoliday(businesscalendar.Holiday{SpecificDate: int64(h1), Name: "Holiday 1"}); err != nil {
		t.Fatal(err)
	}

	c1, _ := time.ParseDateUTC("2026-09-25")
	if err := m.AddClosure(businesscalendar.Closure{SpecificDate: int64(c1), Reason: "Closure 1"}); err != nil {
		t.Fatal(err)
	}

	h2, _ := time.ParseDateUTC("2026-10-12")
	if err := m.AddHoliday(businesscalendar.Holiday{SpecificDate: int64(h2), Name: "Holiday 2"}); err != nil {
		t.Fatal(err)
	}

	from, _ := time.ParseDateUTC("2026-09-01")
	to, _ := time.ParseDateUTC("2026-09-30")

	holidays, closures, err := m.ClosedDates(from, to)
	if err != nil {
		t.Fatalf("ClosedDates: %v", err)
	}

	expectedHolidays := []string{"2026-09-18"}
	expectedClosures := []string{"2026-09-25"}

	if !reflect.DeepEqual(holidays, expectedHolidays) {
		t.Errorf("expected holidays %v, got %v", expectedHolidays, holidays)
	}
	if !reflect.DeepEqual(closures, expectedClosures) {
		t.Errorf("expected closures %v, got %v", expectedClosures, closures)
	}
}

func TestClosedDatesEmpty(t *testing.T) {
	m, _ := newModule(t, nil)
	seedWeek(t, m)

	from, _ := time.ParseDateUTC("2026-09-01")
	to, _ := time.ParseDateUTC("2026-09-30")

	holidays, closures, err := m.ClosedDates(from, to)
	if err != nil {
		t.Fatalf("ClosedDates: %v", err)
	}

	if len(holidays) != 0 {
		t.Errorf("expected empty holidays, got %v", holidays)
	}
	if len(closures) != 0 {
		t.Errorf("expected empty closures, got %v", closures)
	}
}
