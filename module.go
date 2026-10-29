package businesscalendar

import (
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/orm"
	tinytime "webtyp.com/time"
)

// Deps lleva los colaboradores inyectados que un módulo nunca debe construir por sí mismo.
// IDs es obligatorio; Publisher es opcional — nil deshabilita la publicación de eventos
// de forma silenciosa (no-op).
// Ejemplo: deps := businesscalendar.Deps{IDs: ids, Publisher: pub}
type Deps struct {
	IDs       model.IDGenerator
	Publisher events.Publisher
}

// Module representa el módulo de dominio de horas y feriados.
// Ejemplo: var m *businesscalendar.Module = cal
type Module struct {
	db  *orm.DB
	ids model.IDGenerator
	pub events.Publisher
}

// New conecta el módulo a un *orm.DB ya conectado; se asume que el esquema
// ya existe — consulte el subpaquete migrate.
// Ejemplo: cal, err := businesscalendar.New(db, deps)
func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.IDs == nil {
		return nil, fmt.Err("business_calendar: Deps.IDs is required")
	}
	return &Module{db: db, ids: deps.IDs, pub: deps.Publisher}, nil
}

// ModelName es la identidad de este módulo: mcp.HarvestOps califica cada operación como
// "business_calendar.<name>" en la red — la calificación que hace que su
// propio "get_day_bounds" sea distinto de la operación de appointment_booking con el mismo
// nombre base (la colisión que este mecanismo hace
// irrepresentable). Los tres constructores NewXView de view.go pasan esta misma
// constante como view.Ops.Module.
// Ejemplo: name := cal.ModelName()
const ModelName = "business_calendar"

func (m *Module) ModelName() string { return ModelName }

// publish dispara EventCalendarChanged solo cuando hay un publisher conectado. Cada
// mutación exitosa pasa por aquí para que el campo de dirección permanezca centralizado.
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
		if orm.IsNotFound(err) {
			return BusinessHours{}, ErrNotFound
		}
		return BusinessHours{}, err
	}
	return bh, nil
}

// hoursCloseTime reporta si moverse de prev a next CIERRA tiempo — la única
// dirección que puede invalidar una reserva existente. Un día de la semana que
// abre más temprano y cierra más temprano está cerrando: cierto tiempo previamente
// reservable desapareció, por lo que Closed es true.
func hoursCloseTime(prev, next BusinessHours) bool {
	if !next.IsOpen {
		return prev.IsOpen // open → closed closes; closed → closed does not
	}
	if !prev.IsOpen {
		return false // closed → open only opens
	}

	nextOpen, _ := parseClock(next.OpenMin)
	nextClose, _ := parseClock(next.CloseMin)
	prevOpen, _ := parseClock(prev.OpenMin)
	prevClose, _ := parseClock(prev.CloseMin)

	return nextOpen > prevOpen || nextClose < prevClose
}

// UpsertBusinessHours crea o actualiza la regla semanal para un día de la semana. Una
// fila nueva nunca puede invalidar la disponibilidad existente (el día no tenía
// horario antes, por lo que ya estaba cerrado por defecto), por lo que una creación
// publica Closed: false; solo una actualización que reduce la ventana publica Closed: true.
// Ejemplo: err := cal.UpsertBusinessHours(bh)
func (m *Module) UpsertBusinessHours(h BusinessHours) error {
	if h.DayOfWeek < 0 || h.DayOfWeek > 6 {
		return &model.FieldError{Field: "day_of_week", Err: ErrInvalidDay}
	}
	if h.IsOpen {
		openMin, err1 := parseClock(h.OpenMin)
		closeMin, err2 := parseClock(h.CloseMin)
		if err1 != nil || err2 != nil || openMin >= closeMin {
			return &model.FieldError{Field: "open_min", Err: ErrInvalidMinutes}
		}
	} else {
		// Un día cerrado no lleva minutos significativos — normalizar a una
		// codificación única para que GetDayBounds nunca tenga que interpretarlos.
		h.OpenMin, h.CloseMin = "00:00", "00:00"
	}

	prev, err := m.businessHoursByDay(int(h.DayOfWeek))
	if err != nil && model.KindOf(err) != model.ErrorNotFound {
		return err
	}
	if err != nil && model.KindOf(err) == model.ErrorNotFound {
		h.Id = m.ids.NewID()
		h.UpdatedAt = tinytime.NowNano()
		if err := m.db.Create(&h); err != nil {
			return err
		}
		m.publish(CalendarKindBusinessHours, false, 0, 0)
		return nil
	}

	closed := hoursCloseTime(prev, h)
	h.Id = prev.Id
	h.UpdatedAt = tinytime.NowNano()
	if err := m.db.Update(&h, orm.Eq(BusinessHours_.Id, prev.Id)); err != nil {
		return err
	}
	m.publish(CalendarKindBusinessHours, closed, 0, 0)
	return nil
}

// ListBusinessHours devuelve las siete reglas de días de la semana ordenadas por day_of_week. Una
// semana vacía es un estado válido (cerrada por defecto), no un error.
// Ejemplo: rows, err := cal.ListBusinessHours()
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

// AddHoliday registra un cierre fechado legal/administrativo. Una segunda fila en
// la misma fecha es rechazada (ErrDuplicateDate) — un feriado duplicado
// cerraría dos veces un día y haría que RemoveHoliday parezca roto.
// Ejemplo: err := cal.AddHoliday(h)
func (m *Module) AddHoliday(h Holiday) error {
	var ex Holiday
	_, err := ReadOneHoliday(m.db.Query(&ex).Where(Holiday_.SpecificDate).Eq(h.SpecificDate), &ex)
	if err == nil {
		return ErrDuplicateDate
	}
	if !orm.IsNotFound(err) {
		return err
	}
	h.Id = m.ids.NewID()
	h.UpdatedAt = tinytime.NowNano()
	if err := h.Validate(model.ActionCreate); err != nil { return err }
	if err := m.db.Create(&h); err != nil {
		return err
	}
	m.publish(CalendarKindHoliday, true, h.SpecificDate, h.SpecificDate)
	return nil
}

// RemoveHoliday elimina un feriado por id y reabre su día para reservas.
// Ejemplo: err := cal.RemoveHoliday(id)
func (m *Module) RemoveHoliday(id string) error {
	var h Holiday
	_, err := ReadOneHoliday(m.db.Query(&h).Where(Holiday_.Id).Eq(id), &h)
	if err != nil {
		if orm.IsNotFound(err) {
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

// ListHolidays devuelve feriados en el rango [from, to] (segundos UTC a medianoche,
// inclusivo). Un from/to en cero significa sin límite en esa dirección.
// Ejemplo: rows, err := cal.ListHolidays(0, 0)
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

// AddClosure registra un cierre fechado que el propio establecimiento decidió
// (mantenimiento, un evento). Las fechas del mismo tipo son únicas, por lo que
// un segundo cierre en una fecha ya cerrada por cierre es rechazado con ErrDuplicateDate.
// Ejemplo: err := cal.AddClosure(c)
func (m *Module) AddClosure(c Closure) error {
	var ex Closure
	_, err := ReadOneClosure(m.db.Query(&ex).Where(Closure_.SpecificDate).Eq(c.SpecificDate), &ex)
	if err == nil {
		return ErrDuplicateDate
	}
	if !orm.IsNotFound(err) {
		return err
	}
	c.Id = m.ids.NewID()
	c.UpdatedAt = tinytime.NowNano()
	if err := c.Validate(model.ActionCreate); err != nil { return err }
	if err := m.db.Create(&c); err != nil {
		return err
	}
	m.publish(CalendarKindClosure, true, c.SpecificDate, c.SpecificDate)
	return nil
}

// RemoveClosure elimina un cierre por id.
// Ejemplo: err := cal.RemoveClosure(id)
func (m *Module) RemoveClosure(id string) error {
	var c Closure
	_, err := ReadOneClosure(m.db.Query(&c).Where(Closure_.Id).Eq(id), &c)
	if err != nil {
		if orm.IsNotFound(err) {
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

// ListClosures devuelve cierres en el rango [from, to] (segundos UTC a medianoche,
// inclusivo). Un from/to en cero significa sin límite en esa dirección.
// Ejemplo: rows, err := cal.ListClosures(0, 0)
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
