package tests

import (
	"testing"

	businesscalendar "github.com/veltylabs/business_calendar"
	"webtyp.com/model"
	"webtyp.com/router/mock"
)

func mounted(t *testing.T, m *businesscalendar.Module) *mock.Router {
	t.Helper()
	reg := &mock.Router{}
	reg.Configure(mock.Config{
		Authorize: func(userID string, r model.Resource, a model.Action) bool { return true },
	})
	m.MountOperations(reg)
	return reg
}

// TestMountOperations_NineOpsAndRequires — the full op surface is registered
// with the correct Requires bitmasks and paths.
func TestMountOperations_NineOpsAndRequires(t *testing.T) {
	m, _ := newModule(t, nil)
	reg := mounted(t, m)

	if m.ModelName() != "business_calendar" {
		t.Errorf("expected ModelName business_calendar, got %q", m.ModelName())
	}

	routes := reg.Routes()
	if len(routes) != 9 {
		t.Fatalf("expected 9 ops, got %d", len(routes))
	}

	type want struct {
		path     string
		resource string
		action   model.Action
		args     bool
	}
	wants := []want{
		{businesscalendar.OpListBusinessHours, "business_hours", model.Read, false},
		{businesscalendar.OpUpsertBusinessHours, "business_hours", model.Create | model.Update, true},
		{businesscalendar.OpGetDayBounds, "business_hours", model.Read, true},
		{businesscalendar.OpListHolidays, "holiday", model.Read, false},
		{businesscalendar.OpAddHoliday, "holiday", model.Create, true},
		{businesscalendar.OpRemoveHoliday, "holiday", model.Delete, true},
		{businesscalendar.OpListClosures, "closure", model.Read, false},
		{businesscalendar.OpAddClosure, "closure", model.Create, true},
		{businesscalendar.OpRemoveClosure, "closure", model.Delete, true},
	}
	for _, r := range routes {
		var w *want
		for i := range wants {
			if wants[i].path == r.Path[1:] {
				w = &wants[i]
				break
			}
		}
		if w == nil {
			t.Errorf("unexpected route %q", r.Path)
			continue
		}
		if string(r.Resource) != w.resource {
			t.Errorf("%s: resource %q, want %q", r.Path, r.Resource, w.resource)
		}
		if r.Action != w.action {
			t.Errorf("%s: action %v, want %v", r.Path, r.Action, w.action)
		}
		if w.args && r.Args == nil {
			t.Errorf("%s: expected an Accepts schema", r.Path)
		}
		if !w.args && r.Args != nil {
			t.Errorf("%s: expected Accepts(nil)", r.Path)
		}
	}
}

func TestOpGetDayBounds_Success(t *testing.T) {
	m, _ := newModule(t, nil)
	seedWeek(t, m)
	reg := mounted(t, m)

	ctx := &mock.Context{InBody: []byte(`{"date":1789354800}`)} // Monday
	ctx.SetUserID("u1")
	reg.Invoke("OP", "/"+businesscalendar.OpGetDayBounds, ctx)
	if ctx.Status != 0 {
		t.Fatalf("expected success, got status %d body=%s", ctx.Status, ctx.ResponseBody())
	}
	if len(ctx.ResponseBody()) == 0 {
		t.Error("expected a non-empty encoded response")
	}
}

func TestOpListBusinessHours_Success(t *testing.T) {
	m, _ := newModule(t, nil)
	seedWeek(t, m)
	reg := mounted(t, m)

	ctx := &mock.Context{}
	ctx.SetUserID("u1")
	reg.Invoke("OP", "/"+businesscalendar.OpListBusinessHours, ctx)
	if ctx.Status != 0 {
		t.Fatalf("expected success, got status %d body=%s", ctx.Status, ctx.ResponseBody())
	}
	if len(ctx.ResponseBody()) == 0 {
		t.Error("expected a non-empty encoded response")
	}
}

func TestOpAddHoliday_DuplicateIs409(t *testing.T) {
	m, _ := newModule(t, nil)
	reg := mounted(t, m)

	body := []byte(`{"specific_date":1789354800,"name":"Holiday"}`)
	first := &mock.Context{InBody: body}
	first.SetUserID("u1")
	reg.Invoke("OP", "/"+businesscalendar.OpAddHoliday, first)
	if first.Status != 200 {
		t.Fatalf("first add: status %d body=%s", first.Status, first.ResponseBody())
	}

	second := &mock.Context{InBody: body}
	second.SetUserID("u1")
	reg.Invoke("OP", "/"+businesscalendar.OpAddHoliday, second)
	if second.Status != 409 {
		t.Errorf("duplicate add must be 409, got %d", second.Status)
	}
}

func TestOpAddHoliday_MissingNameIs400(t *testing.T) {
	m, _ := newModule(t, nil)
	reg := mounted(t, m)

	ctx := &mock.Context{InBody: []byte(`{"specific_date":1789354800}`)}
	ctx.SetUserID("u1")
	reg.Invoke("OP", "/"+businesscalendar.OpAddHoliday, ctx)
	if ctx.Status != 400 {
		t.Errorf("missing required name must be 400, got %d", ctx.Status)
	}
}

func TestOpUpsertBusinessHours_InvalidMinutesIs400(t *testing.T) {
	m, _ := newModule(t, nil)
	reg := mounted(t, m)

	ctx := &mock.Context{InBody: []byte(`{"day_of_week":1,"open_min":1080,"close_min":480,"is_open":true}`)}
	ctx.SetUserID("u1")
	reg.Invoke("OP", "/"+businesscalendar.OpUpsertBusinessHours, ctx)
	if ctx.Status != 400 {
		t.Errorf("invalid minutes must be 400, got %d", ctx.Status)
	}
}

func TestOpRemoveHoliday_NotFoundIs404(t *testing.T) {
	m, _ := newModule(t, nil)
	reg := mounted(t, m)

	ctx := &mock.Context{InBody: []byte(`{"id":"nope"}`)}
	ctx.SetUserID("u1")
	reg.Invoke("OP", "/"+businesscalendar.OpRemoveHoliday, ctx)
	if ctx.Status != 404 {
		t.Errorf("remove unknown must be 404, got %d", ctx.Status)
	}
}
