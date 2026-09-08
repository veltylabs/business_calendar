package businesscalendar

import (
	"webtyp.com/model"
	"webtyp.com/orm"
	tinytime "webtyp.com/time"
)

// ClosedReason names the origin of a closure. The string literals live ONLY in
// these constants — never inline at a call site.
type ClosedReason string

const (
	ClosedWeekly  ClosedReason = "WEEKLY"  // business_hours says closed that weekday
	ClosedHoliday ClosedReason = "HOLIDAY" // a Holiday row falls on that date
	ClosedLocal   ClosedReason = "CLOSURE" // a Closure row falls on that date
)

// DayDetail is this module's own answer for one date: the neutral bounds PLUS
// why a closed day is closed. ClosedBy is this module's vocabulary and never
// crosses to a neighbour — see Reader below.
//
// It is an output-only result model: it implements model.Encodable so the
// get_day_bounds op can respond with it, and it is never a DB table.
type DayDetail struct {
	Bounds tinytime.DayBounds
	// ClosedBy names why the day is closed. "" when Bounds.Open is true.
	ClosedBy ClosedReason
}

func (d *DayDetail) EncodeFields(w model.FieldWriter) {
	w.Bool("open", d.Bounds.Open)
	w.Int("open_min", int64(d.Bounds.OpenMin))
	w.Int("close_min", int64(d.Bounds.CloseMin))
	w.String("closed_by", string(d.ClosedBy))
}

func (d *DayDetail) IsNil() bool { return d == nil }

// Reader is the READ-ONLY port this module's own consumers depend on — a
// convenience alias, not the contract a sibling domain module names.
//
// A neighbour that only needs "is this date usable, and between which
// minutes" (e.g. veltylabs/appointment_booking) declares its OWN interface
// returning tinytime.DayBounds — a neutral leaf both sides already import —
// and this Module satisfies it structurally, with no adapter and no import in
// either direction. See webtyp/docs/AGENDA_DOMAIN_MASTER_PLAN.md §3-bis for
// why: a generic booking module must not depend on any specific institutional
// calendar.
type Reader interface {
	// GetDayBounds answers "is the establishment open on this date, and
	// between which minutes". date is midnight UTC in seconds.
	GetDayBounds(date int64) (tinytime.DayBounds, error)
}

// GetDayDetail resolves the establishment's status for one date, INCLUDING
// why a closed day is closed. This is the single resolution path — GetBounds
// wraps it and simply discards ClosedBy for a Go-level consumer that never
// asked for it.
//
// Resolution order, in this exact precedence:
//
//  1. A Closure on that date → {Open: false, ClosedBy: ClosedLocal}.
//  2. A Holiday on that date → {Open: false, ClosedBy: ClosedHoliday}.
//  3. BusinessHours for that weekday with is_open == false →
//     {Open: false, ClosedBy: ClosedWeekly}.
//  4. Otherwise {Open: true, OpenMin, CloseMin}.
//
// A local closure outranks a holiday because it is the more specific, more
// recent decision, and because a holiday the establishment chose to work
// through must not be reopened by removing the closure. A weekday with no
// business_hours row at all is closed (closed by default — the absence of a
// schedule never means open).
func (m *Module) GetDayDetail(date int64) (DayDetail, error) {
	var c Closure
	_, err := ReadOneClosure(m.db.Query(&c).Where(Closure_.SpecificDate).Eq(date), &c)
	if err == nil {
		return DayDetail{ClosedBy: ClosedLocal}, nil
	}
	if err != orm.ErrNotFound {
		return DayDetail{}, err
	}

	var h Holiday
	_, err = ReadOneHoliday(m.db.Query(&h).Where(Holiday_.SpecificDate).Eq(date), &h)
	if err == nil {
		return DayDetail{ClosedBy: ClosedHoliday}, nil
	}
	if err != orm.ErrNotFound {
		return DayDetail{}, err
	}

	bh, err := m.businessHoursByDay(tinytime.Weekday(date))
	if err != nil {
		if err == ErrNotFound {
			return DayDetail{ClosedBy: ClosedWeekly}, nil
		}
		return DayDetail{}, err
	}
	if !bh.IsOpen {
		return DayDetail{ClosedBy: ClosedWeekly}, nil
	}
	return DayDetail{Bounds: tinytime.DayBounds{Open: true, OpenMin: int(bh.OpenMin), CloseMin: int(bh.CloseMin)}}, nil
}

// GetDayBounds is the Reader port: the neutral bounds only, no reason
// attached. Satisfies any sibling's own BoundsReader-shaped interface
// structurally.
func (m *Module) GetDayBounds(date int64) (tinytime.DayBounds, error) {
	d, err := m.GetDayDetail(date)
	return d.Bounds, err
}

var _ Reader = (*Module)(nil)
