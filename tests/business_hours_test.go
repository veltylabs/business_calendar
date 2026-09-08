package tests

import (
	"testing"

	businesscalendar "github.com/veltylabs/business_calendar"
)

// TestAdminDefinesWeeklyBusinessHours — CU-01. The administrator defines the
// weekly opening window; the reader sees exactly that.
func TestAdminDefinesWeeklyBusinessHours(t *testing.T) {
	m, _ := newModule(t, nil)
	seedWeek(t, m)

	rows, err := m.ListBusinessHours()
	if err != nil {
		t.Fatalf("ListBusinessHours: %v", err)
	}
	if len(rows) != 7 {
		t.Fatalf("expected 7 weekly rows, got %d", len(rows))
	}

	// Open weekdays (Monday=1, Friday=5) carry the window.
	if !rows[1].IsOpen || rows[1].OpenMin != 480 || rows[1].CloseMin != 1080 {
		t.Errorf("Monday incorrect: %+v", rows[1])
	}
	// Closed weekend days normalize their minutes to 0.
	if rows[0].IsOpen || rows[0].OpenMin != 0 || rows[0].CloseMin != 0 {
		t.Errorf("Sunday incorrect: %+v", rows[0])
	}
	if rows[6].IsOpen || rows[6].OpenMin != 0 || rows[6].CloseMin != 0 {
		t.Errorf("Saturday incorrect: %+v", rows[6])
	}
}

// TestWideningHoursIsVisibleToReaderImmediately — CU-03. A Reader.GetDayBounds
// call made AFTER an upsert sees the new bounds with no restart and no cache
// invalidation step.
func TestWideningHoursIsVisibleToReaderImmediately(t *testing.T) {
	m, _ := newModule(t, nil)

	// Monday, 08:00–18:00.
	if err := m.UpsertBusinessHours(businesscalendar.BusinessHours{
		DayOfWeek: 1, OpenMin: 480, CloseMin: 1080, IsOpen: true,
	}); err != nil {
		t.Fatalf("upsert narrow: %v", err)
	}
	before, err := m.GetDayBounds(sep14_2026_Mon)
	if err != nil {
		t.Fatalf("GetDayBounds: %v", err)
	}
	if !before.Open || before.OpenMin != 480 || before.CloseMin != 1080 {
		t.Fatalf("expected 480..1080, got %+v", before)
	}

	// Widen to 07:00–20:00.
	if err := m.UpsertBusinessHours(businesscalendar.BusinessHours{
		DayOfWeek: 1, OpenMin: 420, CloseMin: 1200, IsOpen: true,
	}); err != nil {
		t.Fatalf("upsert widen: %v", err)
	}
	after, err := m.GetDayBounds(sep14_2026_Mon)
	if err != nil {
		t.Fatalf("GetDayBounds after: %v", err)
	}
	if !after.Open || after.OpenMin != 420 || after.CloseMin != 1200 {
		t.Errorf("expected widened 420..1200, got %+v", after)
	}
}

// TestUpsertBusinessHours_NarrowingCloses — an update that narrows the window
// (later open, earlier close) is still reflected, and hoursCloseTime treats it
// as closing.
func TestUpsertBusinessHours_NarrowingCloses(t *testing.T) {
	pub := newRecording()
	m, _ := newModule(t, pub)

	if err := m.UpsertBusinessHours(businesscalendar.BusinessHours{
		DayOfWeek: 2, OpenMin: 480, CloseMin: 1080, IsOpen: true,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := m.UpsertBusinessHours(businesscalendar.BusinessHours{
		DayOfWeek: 2, OpenMin: 540, CloseMin: 1020, IsOpen: true,
	}); err != nil {
		t.Fatalf("upsert narrow: %v", err)
	}

	if len(pub.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(pub.Events))
	}
	if !pub.Events[1].Payload.(*businesscalendar.CalendarChangedPayload).Closed {
		t.Error("narrowing the window must publish Closed: true")
	}
	if pub.Events[1].Payload.(*businesscalendar.CalendarChangedPayload).Kind != businesscalendar.CalendarKindBusinessHours {
		t.Errorf("expected kind BUSINESS_HOURS, got %q", pub.Events[1].Payload.(*businesscalendar.CalendarChangedPayload).Kind)
	}
}

func TestInvalidMinutesRejected(t *testing.T) {
	m, _ := newModule(t, nil)

	cases := []businesscalendar.BusinessHours{
		{DayOfWeek: 1, OpenMin: 1080, CloseMin: 480, IsOpen: true}, // open >= close
		{DayOfWeek: 1, OpenMin: -1, CloseMin: 480, IsOpen: true},   // negative
		{DayOfWeek: 1, OpenMin: 0, CloseMin: 1440, IsOpen: true},   // 1440 out of range
		{DayOfWeek: 1, OpenMin: 0, CloseMin: 1441, IsOpen: true},   // 1441 out of range
	}
	for i, c := range cases {
		if err := m.UpsertBusinessHours(c); err != businesscalendar.ErrInvalidMinutes {
			t.Errorf("case %d: expected ErrInvalidMinutes, got %v", i, err)
		}
	}
}

func TestInvalidDayRejected(t *testing.T) {
	m, _ := newModule(t, nil)
	err := m.UpsertBusinessHours(businesscalendar.BusinessHours{
		DayOfWeek: 7, OpenMin: 480, CloseMin: 1080, IsOpen: true,
	})
	if err != businesscalendar.ErrInvalidDay {
		t.Errorf("expected ErrInvalidDay, got %v", err)
	}
}

// TestListBusinessHours_EmptyIsNotError — an unseeded week is a valid closed
// state, not an error.
func TestListBusinessHours_EmptyIsNotError(t *testing.T) {
	m, _ := newModule(t, nil)
	rows, err := m.ListBusinessHours()
	if err != nil {
		t.Fatalf("ListBusinessHours on empty week: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}
}
