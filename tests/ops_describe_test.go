package tests

import (
	"testing"

	businesscalendar "github.com/veltylabs/business_calendar"
)

func TestMountOperations_Descriptions(t *testing.T) {
	m, _ := newModule(t, nil)
	reg := mounted(t, m)

	routes := reg.Routes()
	if len(routes) != 9 {
		t.Fatalf("expected 9 ops, got %d", len(routes))
	}

	wants := map[string]string{
		businesscalendar.OpListBusinessHours:   businesscalendar.DescListBusinessHours,
		businesscalendar.OpUpsertBusinessHours: businesscalendar.DescUpsertBusinessHours,
		businesscalendar.OpGetDayBounds:        businesscalendar.DescGetDayBounds,
		businesscalendar.OpListHolidays:        businesscalendar.DescListHolidays,
		businesscalendar.OpAddHoliday:          businesscalendar.DescAddHoliday,
		businesscalendar.OpRemoveHoliday:       businesscalendar.DescRemoveHoliday,
		businesscalendar.OpListClosures:        businesscalendar.DescListClosures,
		businesscalendar.OpAddClosure:          businesscalendar.DescAddClosure,
		businesscalendar.OpRemoveClosure:       businesscalendar.DescRemoveClosure,
	}

	for _, r := range routes {
		path := r.Path[1:] // strip leading slash
		wantDesc, ok := wants[path]
		if !ok {
			t.Errorf("unexpected route %q", r.Path)
			continue
		}
		if r.Description == "" {
			t.Errorf("route %q has empty description", path)
		}
		if r.Description != wantDesc {
			t.Errorf("route %q description = %q, want %q", path, r.Description, wantDesc)
		}
	}
}
