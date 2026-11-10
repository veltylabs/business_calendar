package tests

import (
	"testing"

	businesscalendar "github.com/veltylabs/business_calendar"
	"webtyp.com/crud"
	"webtyp.com/events"
	"webtyp.com/model"
	"webtyp.com/router"
)

// call runs one operation of the module as `user` through the same loopback the demo uses and answers the callback error.
func call(user string, pub events.Publisher, mod *businesscalendar.Module, entity string, action model.Action, args model.Encodable, into model.Decodable) error {
	var callErr error
	crud.LoopbackActingAs(pub, user, mod).Call(
		router.Operation{Module: businesscalendar.ModelName, Entity: entity, Action: action},
		args, into, func(err error) { callErr = err })
	return callErr
}

// failedField answers the field a refusal points at, or "".
func failedField(err error) string {
	if f, ok := err.(interface{ FailedField() string }); ok {
		return f.FailedField()
	}
	return ""
}

func createHours(m *businesscalendar.Module, rec *businesscalendar.BusinessHours) error {
	return call("u1", nil, m, "business_hours", model.Create, &crud.CreateArgs{Records: []model.Model{rec}}, nil)
}

func readHours(t *testing.T, m *businesscalendar.Module, id string) businesscalendar.BusinessHours {
	t.Helper()
	var list businesscalendar.BusinessHoursList
	if err := call("u1", nil, m, "business_hours", model.Read, &crud.ReadArgs{ID: id}, &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("one row expected, got %d", len(list))
	}
	return *list[0]
}

func TestBusinessHours_SundayAndClosedAreAllowed(t *testing.T) {
	m, _ := newModule(t, nil)
	if err := createHours(m, &businesscalendar.BusinessHours{DayOfWeek: 0, IsOpen: false}); err != nil {
		t.Fatal(err)
	}
	row := readHours(t, m, "id-1")
	if row.OpenMin != "00:00" || row.CloseMin != "00:00" {
		t.Errorf("a closed day stores 00:00–00:00, got %q–%q", row.OpenMin, row.CloseMin)
	}
}

func TestBusinessHours_OneRowPerDay(t *testing.T) {
	m, _ := newModule(t, nil)
	rec := func() *businesscalendar.BusinessHours {
		return &businesscalendar.BusinessHours{DayOfWeek: 1, OpenMin: "08:00", CloseMin: "18:00", IsOpen: true}
	}
	if err := createHours(m, rec()); err != nil {
		t.Fatal(err)
	}
	if err := createHours(m, rec()); model.KindOf(err) != model.ErrorConflict {
		t.Errorf("the second row of the same day: got %v, want a conflict", err)
	}
}

func TestBusinessHours_OpenBeforeClose(t *testing.T) {
	m, _ := newModule(t, nil)
	err := createHours(m, &businesscalendar.BusinessHours{DayOfWeek: 2, OpenMin: "18:00", CloseMin: "08:00", IsOpen: true})
	if failedField(err) != "open_min" {
		t.Errorf("got %v (field %q), want a refusal on open_min", err, failedField(err))
	}
}

func TestBusinessHours_InvalidDay(t *testing.T) {
	m, _ := newModule(t, nil)
	err := createHours(m, &businesscalendar.BusinessHours{DayOfWeek: 7, OpenMin: "08:00", CloseMin: "18:00", IsOpen: true})
	if failedField(err) != "day_of_week" {
		t.Errorf("got %v (field %q), want a refusal on day_of_week", err, failedField(err))
	}
}

func TestBusinessHours_UpdateOnlyListedFields(t *testing.T) {
	m, _ := newModule(t, nil)
	if err := createHours(m, &businesscalendar.BusinessHours{DayOfWeek: 1, OpenMin: "08:00", CloseMin: "18:00", IsOpen: true}); err != nil {
		t.Fatal(err)
	}
	err := call("u1", nil, m, "business_hours", model.Update,
		&crud.UpdateArgs{IDs: []string{"id-1"}, Fields: []string{"close_min"}, Record: &businesscalendar.BusinessHours{CloseMin: "17:00"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if row := readHours(t, m, "id-1"); row.OpenMin != "08:00" || row.CloseMin != "17:00" {
		t.Errorf("got %q–%q, want 08:00–17:00", row.OpenMin, row.CloseMin)
	}
}

func TestBusinessHours_UpdateRejectsOtherFields(t *testing.T) {
	m, _ := newModule(t, nil)
	if err := createHours(m, &businesscalendar.BusinessHours{DayOfWeek: 1, OpenMin: "08:00", CloseMin: "18:00", IsOpen: true}); err != nil {
		t.Fatal(err)
	}
	err := call("u1", nil, m, "business_hours", model.Update,
		&crud.UpdateArgs{IDs: []string{"id-1"}, Fields: []string{"id"}, Record: &businesscalendar.BusinessHours{}}, nil)
	if model.KindOf(err) != model.ErrorInvalid {
		t.Errorf("got %v, want crud's refusal of an immutable field", err)
	}
}

func TestBusinessHours_CloseDayNormalizesMinutes(t *testing.T) {
	m, _ := newModule(t, nil)
	if err := createHours(m, &businesscalendar.BusinessHours{DayOfWeek: 3, OpenMin: "08:00", CloseMin: "18:00", IsOpen: true}); err != nil {
		t.Fatal(err)
	}
	err := call("u1", nil, m, "business_hours", model.Update,
		&crud.UpdateArgs{IDs: []string{"id-1"}, Fields: []string{"is_open"}, Record: &businesscalendar.BusinessHours{IsOpen: false}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if row := readHours(t, m, "id-1"); row.IsOpen || row.OpenMin != "00:00" || row.CloseMin != "00:00" {
		t.Errorf("a closed day stores 00:00–00:00, got open=%v %q–%q", row.IsOpen, row.OpenMin, row.CloseMin)
	}
}

func createHoliday(m *businesscalendar.Module, pub events.Publisher, date int64) error {
	return call("u1", pub, m, "holiday", model.Create,
		&crud.CreateArgs{Records: []model.Model{&businesscalendar.Holiday{SpecificDate: date, Name: "Navidad"}}}, nil)
}

func TestHoliday_DuplicateDateRejected(t *testing.T) {
	m, _ := newModule(t, nil)
	if err := createHoliday(m, nil, dec25_2026_Fri); err != nil {
		t.Fatal(err)
	}
	err := createHoliday(m, nil, dec25_2026_Fri)
	if model.KindOf(err) != model.ErrorConflict {
		t.Errorf("got %v, want a conflict", err)
	}
}

func TestHoliday_NoUpdateOperation(t *testing.T) {
	m, _ := newModule(t, nil)
	err := call("u1", nil, m, "holiday", model.Update,
		&crud.UpdateArgs{IDs: []string{"x"}, Fields: []string{"name"}, Record: &businesscalendar.Holiday{Name: "x"}}, nil)
	if err == nil || !contains(err.Error(), "unknown operation") {
		t.Errorf("got %v, want an unknown operation", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestHoliday_DeleteReopensAndPublishes(t *testing.T) {
	pub := newRecording()
	m, _ := newModule(t, pub)
	if err := createHoliday(m, pub, dec25_2026_Fri); err != nil {
		t.Fatal(err)
	}
	if err := call("u1", pub, m, "holiday", model.Delete, &crud.DeleteArgs{IDs: []string{"id-1"}}, nil); err != nil {
		t.Fatal(err)
	}
	var closed []bool
	for _, e := range pub.Events {
		if e.Topic == businesscalendar.EventCalendarChanged {
			closed = append(closed, e.Payload.(*businesscalendar.CalendarChangedPayload).Closed)
		}
	}
	if len(closed) != 2 || !closed[0] || closed[1] {
		t.Errorf("EventCalendarChanged Closed values = %v, want [true false]", closed)
	}
}

func TestClosure_DuplicateDateRejected(t *testing.T) {
	m, _ := newModule(t, nil)
	create := func() error {
		return call("u1", nil, m, "closure", model.Create,
			&crud.CreateArgs{Records: []model.Model{&businesscalendar.Closure{SpecificDate: sep12_2026_Sat, Reason: "Maintenance"}}}, nil)
	}
	if err := create(); err != nil {
		t.Fatal(err)
	}
	if err := create(); model.KindOf(err) != model.ErrorConflict {
		t.Errorf("got %v, want a conflict", err)
	}
}

func TestDayDetail_ClosedOnHoliday(t *testing.T) {
	m, _ := newModule(t, nil)
	if err := createHoliday(m, nil, sep12_2026_Sat); err != nil {
		t.Fatal(err)
	}
	var list businesscalendar.DayStatusList
	err := call("u1", nil, m, "day_detail", model.Read, &crud.ReadArgs{Filter: &businesscalendar.DayStatus{Date: sep12_2026_Sat}}, &list)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Open || list[0].ClosedBy != "HOLIDAY" {
		t.Errorf("got %+v, want one closed row closed by HOLIDAY", list)
	}
}

func TestDayDetail_NeedsDate(t *testing.T) {
	m, _ := newModule(t, nil)
	var list businesscalendar.DayStatusList
	err := call("u1", nil, m, "day_detail", model.Read, &crud.ReadArgs{Filter: &businesscalendar.DayStatus{}}, &list)
	if failedField(err) != "date" {
		t.Errorf("got %v (field %q), want a refusal on date", err, failedField(err))
	}
}

func TestMount_MatchesDefinition(t *testing.T) {
	m, _ := newModule(t, nil)
	if err := crud.Mount(nil, nil, nil, m); err != nil {
		t.Errorf("Mount: %v", err)
	}
}

func TestCreate_PublishesChange(t *testing.T) {
	pub := newRecording()
	m, _ := newModule(t, nil)
	if err := createHoliday(m, pub, dec25_2026_Fri); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range pub.Events {
		if _, ok := e.Payload.(*crud.Change); ok {
			found = true
		}
	}
	if !found {
		t.Errorf("no *crud.Change event was published: %+v", pub.Events)
	}
}
