package businesscalendar

import (
	"webtyp.com/crud"
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/storage"
	"webtyp.com/time"
)

type Deps struct {
	IDs       model.IDGenerator
	Publisher events.Publisher
}

type Module struct {
	db    *orm.DB
	store crud.Store
	ids   model.IDGenerator
	pub   events.Publisher
}

func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.IDs == nil {
		return nil, fmt.Err("business_calendar: Deps.IDs is required")
	}
	m := &Module{db: db, ids: deps.IDs, pub: deps.Publisher}
	m.store = orm.GlobalStore(db, m.ids)
	return m, nil
}

func (m *Module) publish(kind string, closed bool, from, to int64) {
	if m.pub == nil { return }
	m.pub.Publish(events.Event{
		Topic:   EventCalendarChanged,
		Payload: &CalendarChangedPayload{FromDate: from, ToDate: to, Kind: kind, Closed: closed},
	})
}

func (m *Module) UpsertBusinessHours(h BusinessHours) error {
	h, err := normalizeHours(h)
	if err != nil { return err }
	closed := false

	err = func(tx crud.Tx) error {
		row, found, err := firstRow(tx, func() model.Model { return &BusinessHours{} },
			storage.Eq("day_of_week", int64(h.DayOfWeek)))
		if err != nil { return err }
		h.UpdatedAt = time.NowNano()
		if !found {
			h.Id = m.ids.NewID()
			return tx.Create(&h)
		}
		previous := row.(*BusinessHours)
		closed = hoursCloseTime(*previous, h)
		h.Id = previous.Id
		return tx.UpdateFields(&h, previous.Id, []string{
			"day_of_week", "open_min", "close_min", "is_open", "notes", "updated_at",
		})
	}(m.store)

	if err != nil { return err }
	m.publish(CalendarKindBusinessHours, closed, 0, 0)
	return nil
}

func (m *Module) AddHoliday(h Holiday) error {
	err := func(tx crud.Tx) error {
		if err := checkNewHoliday(tx, &h); err != nil { return err }
		h.Id = m.ids.NewID()
		h.UpdatedAt = time.NowNano()
		if err := model.ValidateFields(model.ActionCreate, &h); err != nil { return err }
		return tx.Create(&h)
	}(m.store)
	if err != nil { return err }
	m.publish(CalendarKindHoliday, true, h.SpecificDate, h.SpecificDate)
	return nil
}

func (m *Module) RemoveHoliday(id string) error {
	var removed Holiday
	err := func(tx crud.Tx) error {
		if err := tx.ReadOne(&removed, id); err != nil { return notFound(err) }
		return tx.Delete(&removed, id)
	}(m.store)
	if err != nil { return err }
	m.publish(CalendarKindHoliday, false, removed.SpecificDate, removed.SpecificDate)
	return nil
}

func (m *Module) AddClosure(c Closure) error {
	err := func(tx crud.Tx) error {
		if err := checkNewClosure(tx, &c); err != nil { return err }
		c.Id = m.ids.NewID()
		c.UpdatedAt = time.NowNano()
		if err := model.ValidateFields(model.ActionCreate, &c); err != nil { return err }
		return tx.Create(&c)
	}(m.store)
	if err != nil { return err }
	m.publish(CalendarKindClosure, true, c.SpecificDate, c.SpecificDate)
	return nil
}

func (m *Module) RemoveClosure(id string) error {
	var removed Closure
	err := func(tx crud.Tx) error {
		if err := tx.ReadOne(&removed, id); err != nil { return notFound(err) }
		return tx.Delete(&removed, id)
	}(m.store)
	if err != nil { return err }
	m.publish(CalendarKindClosure, false, removed.SpecificDate, removed.SpecificDate)
	return nil
}

func (m *Module) ListBusinessHours(after string, limit int64) ([]BusinessHours, error) {
	var out []BusinessHours
	err := readByID(m.store, func() model.Model { return &BusinessHours{} }, nil, after, limit,
		func(r model.Model) string { row := r.(*BusinessHours); out = append(out, *row); return row.Id })
	return out, err
}

func (m *Module) ListHolidays(from, to int64, search, after string, limit int64) ([]Holiday, error) {
	var conditions []storage.Condition
	if from != 0 { conditions = append(conditions, storage.Gte("specific_date", from)) }
	if to != 0 { conditions = append(conditions, storage.Lte("specific_date", to)) }
	if search != "" { conditions = append(conditions, storage.Like("name", "%"+search+"%")) }
	var out []Holiday
	err := readByID(m.store, func() model.Model { return &Holiday{} }, conditions, after, limit,
		func(r model.Model) string { row := r.(*Holiday); out = append(out, *row); return row.Id })
	return out, err
}

func (m *Module) ListClosures(from, to int64, search, after string, limit int64) ([]Closure, error) {
	var conditions []storage.Condition
	if from != 0 { conditions = append(conditions, storage.Gte("specific_date", from)) }
	if to != 0 { conditions = append(conditions, storage.Lte("specific_date", to)) }
	if search != "" { conditions = append(conditions, storage.Like("reason", "%"+search+"%")) }
	var out []Closure
	err := readByID(m.store, func() model.Model { return &Closure{} }, conditions, after, limit,
		func(r model.Model) string { row := r.(*Closure); out = append(out, *row); return row.Id })
	return out, err
}
