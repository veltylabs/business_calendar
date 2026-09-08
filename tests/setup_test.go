package tests

import (
	"testing"

	businesscalendar "github.com/veltylabs/business_calendar"
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
)

// Midnight-UTC epoch seconds for the fixed dates the suite uses. Computed for
// 00:00:00 UTC; the weekday (0=Sunday … 6=Saturday) is annotated for clarity.
const (
	sep08_2026_Tue = 1788836400 // Tuesday
	sep09_2026_Wed = 1788922800 // Wednesday
	sep12_2026_Sat = 1789182000 // Saturday
	sep13_2026_Sun = 1789268400 // Sunday
	sep14_2026_Mon = 1789354800 // Monday
	sep15_2026_Tue = 1789441200 // Tuesday
	sep16_2026_Wed = 1789527600 // Wednesday
	oct01_2026_Thu = 1790823600 // Thursday
	dec25_2026_Fri = 1798167600 // Friday
)

// fakeIDs mints stable, deterministic ids.
type fakeIDs struct{ n int }

func (f *fakeIDs) NewID() string {
	f.n++
	return "id-" + fmt.Convert(f.n).String()
}

// recordingPublisher captures every published event so a test can assert both
// the topic and the payload direction.
type recordingPublisher struct {
	Events []events.Event
}

func (r *recordingPublisher) Publish(e events.Event) {
	r.Events = append(r.Events, e)
}

var _ events.Publisher = (*recordingPublisher)(nil)

// newModule builds a Module over storage/mem with a fresh fake id generator and
// an optional publisher. dep for mem is a no-op (no ddl.Compiler), which is
// exactly the seam this module must keep backend-agnostic.
func newModule(t *testing.T, pub events.Publisher) (*businesscalendar.Module, *fakeIDs) {
	t.Helper()
	db := orm.New(mem.New())
	ids := &fakeIDs{}
	m, err := businesscalendar.New(db, businesscalendar.Deps{IDs: ids, Publisher: pub})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m, ids
}

func newRecording() *recordingPublisher {
	return &recordingPublisher{}
}

// seedWeek defines a conventional week through the real service: open Monday..
// Friday 08:00–18:00, closed Saturday and Sunday. It exercises the write path
// (UpsertBusinessHours) rather than raw db access.
func seedWeek(t *testing.T, m *businesscalendar.Module) {
	t.Helper()
	for day := int64(0); day < 7; day++ {
		err := m.UpsertBusinessHours(businesscalendar.BusinessHours{
			DayOfWeek: day,
			OpenMin:   480,
			CloseMin:  1080,
			IsOpen:    day != 0 && day != 6,
		})
		if err != nil {
			t.Fatalf("seed weekday %d: %v", day, err)
		}
	}
}
