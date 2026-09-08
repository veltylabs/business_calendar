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
	if err := p.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
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
	if err := p.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
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
	if err := p.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	items := p.Items()
	if len(items) != 1 || items[0].Label != "Maintenance" {
		t.Errorf("expected the seeded closure, got %+v", items)
	}
}
