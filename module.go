package businesscalendar

import (
	"webtyp.com/ddl"
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/orm"
	tinytime "webtyp.com/time"
)

// Deps carries the injected collaborators a module may never construct itself.
// IDs is required; Publisher is optional — nil disables event publishing
// silently (no-op).
type Deps struct {
	IDs       model.IDGenerator
	Publisher events.Publisher
}

type Module struct {
	db  *orm.DB
	ids model.IDGenerator
	pub events.Publisher
}

func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.IDs == nil {
		return nil, fmt.Err("business_calendar: Deps.IDs is required")
	}
	// ddl.Compiler is an optional capability — only SQL backends (sqlt,
	// postgres) implement it; storage/mem creates tables lazily and needs no
	// DDL. The type assertion, not an unconditional call, is what keeps the
	// module backend-agnostic here.
	if ddlCompiler, ok := db.RawConn().(ddl.Compiler); ok {
		if err := ddl.New(db.RawConn(), ddlCompiler).CreateTable(&BusinessHours{}); err != nil {
			return nil, err
		}
		if err := ddl.New(db.RawConn(), ddlCompiler).CreateTable(&Holiday{}); err != nil {
			return nil, err
		}
		if err := ddl.New(db.RawConn(), ddlCompiler).CreateTable(&Closure{}); err != nil {
			return nil, err
		}
	}
	return &Module{db: db, ids: deps.IDs, pub: deps.Publisher}, nil
}

func (m *Module) ModelName() string { return "business_calendar" }

// publish fires EventCalendarChanged only when a publisher is wired in. Every
// successful mutation funnels through here so the direction field stays right
// in one place.
func (m *Module) publish(kind string, closed bool, from, to int64) {
	if m.pub == nil {
		return
	}
	m.pub.Publish(events.Event{
		Topic:   EventCalendarChanged,
		Payload: &CalendarChangedPayload{FromDate: from, ToDate: to, Kind: kind, Closed: closed},
	})
}

// --- Business hours ---------------------------------------------------------

func (m *Module) businessHoursByDay(day int) (BusinessHours, error) {
	var bh BusinessHours
	_, err := ReadOneBusinessHours(m.db.Query(&bh).Where(BusinessHours_.DayOfWeek).Eq(int64(day)), &bh)
	if err != nil {
		if err == orm.ErrNotFound {
			return BusinessHours{}, ErrNotFound
		}
		return BusinessHours{}, err
	}
	return bh, nil
}

// hoursCloseTime reports whether moving from prev to next CLOSES time — the only
// direction that can invalidate an existing reservation. A weekday that both
// opens earlier and closes earlier is closing: some previously-bookable time
// disappeared, so Closed is true.
func hoursCloseTime(prev, next BusinessHours) bool {
	if !next.IsOpen {
		return prev.IsOpen // open → closed closes; closed → closed does not
	}
	if !prev.IsOpen {
		return false // closed → open only opens
	}
	return next.OpenMin > prev.OpenMin || next.CloseMin < prev.CloseMin
}

// UpsertBusinessHours creates or updates the weekly rule for one weekday. A
// brand-new row can never invalidate existing availability (the day had no
// hours before, so it was already closed by default), so a create publishes
// Closed: false; only an update that narrows the window publishes Closed: true.
func (m *Module) UpsertBusinessHours(h BusinessHours) error {
	if h.DayOfWeek < 0 || h.DayOfWeek > 6 {
		return ErrInvalidDay
	}
	if h.IsOpen {
		if h.OpenMin < 0 || h.OpenMin > 1439 || h.CloseMin < 0 || h.CloseMin > 1439 || h.OpenMin >= h.CloseMin {
			return ErrInvalidMinutes
		}
	} else {
		// A closed day carries no meaningful minutes — normalize to a single
		// encoding so GetDayBounds never has to interpret them.
		h.OpenMin, h.CloseMin = 0, 0
	}

	prev, err := m.businessHoursByDay(int(h.DayOfWeek))
	if err != nil && err != ErrNotFound {
		return err
	}
	if err == ErrNotFound {
		h.Id = m.ids.NewID()
		h.UpdatedAt = tinytime.Now()
		if err := m.db.Create(&h); err != nil {
			return err
		}
		m.publish(CalendarKindBusinessHours, false, 0, 0)
		return nil
	}

	closed := hoursCloseTime(prev, h)
	h.Id = prev.Id
	h.UpdatedAt = tinytime.Now()
	if err := m.db.Update(&h, orm.Eq(BusinessHours_.Id, prev.Id)); err != nil {
		return err
	}
	m.publish(CalendarKindBusinessHours, closed, 0, 0)
	return nil
}

// ListBusinessHours returns all seven weekday rules ordered by day_of_week. An
// empty week is a valid state (closed by default), not an error.
func (m *Module) ListBusinessHours() ([]BusinessHours, error) {
	var bh BusinessHours
	rows, err := ReadAllBusinessHours(m.db.Query(&bh).OrderBy(BusinessHours_.DayOfWeek).Asc())
	if err != nil {
		return nil, err
	}
	out := make([]BusinessHours, len(rows))
	for i, r := range rows {
		out[i] = *r
	}
	return out, nil
}

// --- Holidays ---------------------------------------------------------------

// AddHoliday registers a dated legal/administrative closure. A second row on
// the same date is rejected (ErrDuplicateDate) — a duplicated holiday would
// double-close a day and make RemoveHoliday look broken.
func (m *Module) AddHoliday(h Holiday) error {
	var ex Holiday
	_, err := ReadOneHoliday(m.db.Query(&ex).Where(Holiday_.SpecificDate).Eq(h.SpecificDate), &ex)
	if err == nil {
		return ErrDuplicateDate
	}
	if err != orm.ErrNotFound {
		return err
	}
	h.Id = m.ids.NewID()
	h.UpdatedAt = tinytime.Now()
	if err := m.db.Create(&h); err != nil {
		return err
	}
	m.publish(CalendarKindHoliday, true, h.SpecificDate, h.SpecificDate)
	return nil
}

// RemoveHoliday deletes a holiday by id and reopens its day for booking.
func (m *Module) RemoveHoliday(id string) error {
	var h Holiday
	_, err := ReadOneHoliday(m.db.Query(&h).Where(Holiday_.Id).Eq(id), &h)
	if err != nil {
		if err == orm.ErrNotFound {
			return ErrNotFound
		}
		return err
	}
	if err := m.db.Delete(&h, orm.Eq(Holiday_.Id, id)); err != nil {
		return err
	}
	m.publish(CalendarKindHoliday, false, h.SpecificDate, h.SpecificDate)
	return nil
}

// ListHolidays returns holidays in the range [from, to] (midnight UTC seconds,
// inclusive). A zero from/to means unbounded in that direction.
func (m *Module) ListHolidays(from, to int64) ([]Holiday, error) {
	var h Holiday
	qb := m.db.Query(&h)
	if from != 0 {
		qb = qb.Where(Holiday_.SpecificDate).Gte(from)
	}
	if to != 0 {
		qb = qb.Where(Holiday_.SpecificDate).Lte(to)
	}
	qb = qb.OrderBy(Holiday_.SpecificDate).Asc()
	rows, err := ReadAllHoliday(qb)
	if err != nil {
		return nil, err
	}
	out := make([]Holiday, len(rows))
	for i, r := range rows {
		out[i] = *r
	}
	return out, nil
}

// --- Closures ---------------------------------------------------------------

// AddClosure registers a dated closure the establishment itself decided
// (maintenance, an event). Same-kind dates are unique, so a second closure on
// an already-closed-by-closure date is rejected with ErrDuplicateDate.
func (m *Module) AddClosure(c Closure) error {
	var ex Closure
	_, err := ReadOneClosure(m.db.Query(&ex).Where(Closure_.SpecificDate).Eq(c.SpecificDate), &ex)
	if err == nil {
		return ErrDuplicateDate
	}
	if err != orm.ErrNotFound {
		return err
	}
	c.Id = m.ids.NewID()
	c.UpdatedAt = tinytime.Now()
	if err := m.db.Create(&c); err != nil {
		return err
	}
	m.publish(CalendarKindClosure, true, c.SpecificDate, c.SpecificDate)
	return nil
}

// RemoveClosure deletes a closure by id.
func (m *Module) RemoveClosure(id string) error {
	var c Closure
	_, err := ReadOneClosure(m.db.Query(&c).Where(Closure_.Id).Eq(id), &c)
	if err != nil {
		if err == orm.ErrNotFound {
			return ErrNotFound
		}
		return err
	}
	if err := m.db.Delete(&c, orm.Eq(Closure_.Id, id)); err != nil {
		return err
	}
	m.publish(CalendarKindClosure, false, c.SpecificDate, c.SpecificDate)
	return nil
}

// ListClosures returns closures in the range [from, to] (midnight UTC seconds,
// inclusive). A zero from/to means unbounded in that direction.
func (m *Module) ListClosures(from, to int64) ([]Closure, error) {
	var c Closure
	qb := m.db.Query(&c)
	if from != 0 {
		qb = qb.Where(Closure_.SpecificDate).Gte(from)
	}
	if to != 0 {
		qb = qb.Where(Closure_.SpecificDate).Lte(to)
	}
	qb = qb.OrderBy(Closure_.SpecificDate).Asc()
	rows, err := ReadAllClosure(qb)
	if err != nil {
		return nil, err
	}
	out := make([]Closure, len(rows))
	for i, r := range rows {
		out[i] = *r
	}
	return out, nil
}
