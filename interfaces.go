package businesscalendar

import (
	"webtyp.com/model"
	"webtyp.com/orm"
	tinytime "webtyp.com/time"
)

// DayBounds is the open window of the establishment for one concrete date.
// Minutes from midnight. Open == false means closed, and the minute fields
// carry no meaning.
//
// It is an output-only result model: it implements model.Encodable so the
// get_day_bounds op can respond with it, and it is never a DB table.
type DayBounds struct {
	Open     bool
	OpenMin  int
	CloseMin int
	// ClosedBy names why a closed day is closed, for the UI to say so.
	// "" when Open is true.
	ClosedBy ClosedReason
}

func (d *DayBounds) EncodeFields(w model.FieldWriter) {
	w.Bool("open", d.Open)
	w.Int("open_min", int64(d.OpenMin))
	w.Int("close_min", int64(d.CloseMin))
	w.String("closed_by", string(d.ClosedBy))
}

func (d *DayBounds) IsNil() bool { return d == nil }

// ClosedReason names the origin of a closure. The string literals live ONLY in
// these constants — never inline at a call site.
type ClosedReason string

const (
	ClosedWeekly  ClosedReason = "WEEKLY"  // business_hours says closed that weekday
	ClosedHoliday ClosedReason = "HOLIDAY" // a Holiday row falls on that date
	ClosedLocal   ClosedReason = "CLOSURE" // a Closure row falls on that date
)

// Reader is the READ-ONLY port a neighbour module depends on. It is declared
// HERE, by the owner of the concern — a consumer that declared its own local
// interface would fork this contract and the copy could never be reused.
type Reader interface {
	// GetDayBounds answers "is the establishment open on this date, and
	// between which minutes". date is midnight UTC in seconds.
	GetDayBounds(date int64) (DayBounds, error)
}

// GetDayBounds resolves the establishment's open window for one date.
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
func (m *Module) GetDayBounds(date int64) (DayBounds, error) {
	var c Closure
	_, err := ReadOneClosure(m.db.Query(&c).Where(Closure_.SpecificDate).Eq(date), &c)
	if err == nil {
		return DayBounds{Open: false, ClosedBy: ClosedLocal}, nil
	}
	if err != orm.ErrNotFound {
		return DayBounds{}, err
	}

	var h Holiday
	_, err = ReadOneHoliday(m.db.Query(&h).Where(Holiday_.SpecificDate).Eq(date), &h)
	if err == nil {
		return DayBounds{Open: false, ClosedBy: ClosedHoliday}, nil
	}
	if err != orm.ErrNotFound {
		return DayBounds{}, err
	}

	bh, err := m.businessHoursByDay(tinytime.Weekday(date))
	if err != nil {
		if err == ErrNotFound {
			return DayBounds{Open: false, ClosedBy: ClosedWeekly}, nil
		}
		return DayBounds{}, err
	}
	if !bh.IsOpen {
		return DayBounds{Open: false, ClosedBy: ClosedWeekly}, nil
	}
	return DayBounds{Open: true, OpenMin: int(bh.OpenMin), CloseMin: int(bh.CloseMin)}, nil
}

var _ Reader = (*Module)(nil)
