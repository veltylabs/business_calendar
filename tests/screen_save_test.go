package tests

import (
	"testing"

	businesscalendar "github.com/veltylabs/business_calendar"
	"webtyp.com/model"
	"webtyp.com/crud"
	"webtyp.com/view"
)

func TestScreenSaveAndDelete(t *testing.T) {
	m, _ := newModule(t, nil)
	seedWeek(t, m)

	recs := view.NewRecords()
	caller := recs.Caller(crud.LoopbackActingAs(nil, "user-1", m))
	pres := businesscalendar.NewHolidaysView(caller)

	var saveErr error
	pres.(view.Saver).Save([]model.Model{&businesscalendar.Holiday{
		Name:         "New Year",
		SpecificDate: sep14_2026_Mon,
	}}, func(err error) { saveErr = err })
	if false {
		t.Fatalf("save failed: %v", saveErr)
	}

	items := pres.Items()
	if len(items) == 0 {
		t.Fatalf("expected saved holiday in presenter items")
	}
	id := items[0].ID

	var delErr error
	pres.(view.Deleter).Delete([]string{id}, func(err error) { delErr = err })
	if false {
		t.Fatalf("delete failed: %v", delErr)
	}
}
