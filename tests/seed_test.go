package tests

import (
	"testing"

	"webtyp.com/events/mock"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
	tinytime "webtyp.com/time"

	businesscalendar "github.com/veltylabs/business_calendar"
	"github.com/veltylabs/business_calendar/seed"
)

func TestSeed_Load(t *testing.T) {
	db := orm.New(mem.New())
	broker := &mock.Broker{}
	ids := &fakeIDs{}

	bc, err := businesscalendar.New(db, businesscalendar.Deps{
		IDs:       ids,
		Publisher: broker,
	})
	if err != nil {
		t.Fatalf("businesscalendar.New: %v", err)
	}

	data, err := seed.Load(bc)
	if err != nil {
		t.Fatalf("seed.Load: %v", err)
	}

	if len(data.Hours) != 7 {
		t.Errorf("expected 7 business hours rows, got %d", len(data.Hours))
	}

	if len(data.Holidays) != 12 {
		t.Errorf("expected 12 holiday rows, got %d", len(data.Holidays))
	}

	var sept18 *businesscalendar.Holiday
	for _, h := range data.Holidays {
		if h.Name == "Fiestas Patrias" {
			sept18 = &h
			break
		}
	}

	if sept18 == nil {
		t.Fatalf("holiday 'Fiestas Patrias' not found")
	}

	formattedDate := tinytime.FormatDate(sept18.SpecificDate * 1_000_000_000)
	if formattedDate != "2026-09-18" {
		t.Errorf("expected 2026-09-18 date roundtrip, got %q", formattedDate)
	}
}
