← [Stage 1](PLAN_STAGE_1_TOOLS_DEFINITION.md) | Next → [Stage 3](PLAN_STAGE_3_RULES.md)

# Stage 2 — `module.go` on the store, helpers

## 2.1 Delete

`ops.go` (all of it: the `Op*`/`Desc*` constants, `MountOperations`, every `op*` handler, `DayLimitsArgs` and friends), `hours_view.go` (the `businessHoursView` types only `ops.go` used), and the files `model_orm.go` / `definition_crud.go` if they exist. Verify: `grep -rn "reg.Operation\|view.MountSave\|view.MountDelete\|router.Context\|rpc.WriteError" --include=*.go . | grep -v _test` → empty (the tests of the deleted handlers go in Stage 6).

## 2.2 `module.go` — replace the whole file with exactly (nobody outside this repo calls the removed `ModelName` const/method: `definitionc` writes them in Stage 3)

```go
package businesscalendar

import (
	"webtyp.com/crud"
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/time"
)

// Deps lleva los colaboradores inyectados que un módulo nunca debe construir por sí mismo.
// IDs es obligatorio; Publisher es opcional — nil deshabilita la publicación de eventos.
// Ejemplo: deps := businesscalendar.Deps{IDs: ids, Publisher: pub}
type Deps struct {
	IDs       model.IDGenerator
	Publisher events.Publisher
}

// Module representa el módulo de dominio de horas y feriados.
// Ejemplo: var m *businesscalendar.Module = cal
type Module struct {
	db    *orm.DB
	store *orm.Store // las tres tablas no tienen tenant_id: almacén global
	ids   model.IDGenerator
	pub   events.Publisher
}

// New conecta el módulo a un *orm.DB ya conectado; el esquema lo crea el subpaquete migrate.
// Ejemplo: cal, err := businesscalendar.New(db, deps)
func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.IDs == nil {
		return nil, fmt.Err("business_calendar: Deps.IDs is required")
	}
	return &Module{db: db, store: orm.GlobalStore(db, deps.IDs), ids: deps.IDs, pub: deps.Publisher}, nil
}

// publish dispara EventCalendarChanged solo cuando hay un publisher conectado.
func (m *Module) publish(kind string, closed bool, from, to int64) {
	if m.pub == nil {
		return
	}
	m.pub.Publish(events.Event{
		Topic:   EventCalendarChanged,
		Payload: &CalendarChangedPayload{FromDate: from, ToDate: to, Kind: kind, Closed: closed},
	})
}

// UpsertBusinessHours crea o actualiza la regla semanal de un día. Una fila nueva nunca cierra tiempo (el día ya estaba
// cerrado por defecto) y publica Closed: false; una actualización que reduce la ventana publica Closed: true.
// Ejemplo: err := cal.UpsertBusinessHours(bh)
func (m *Module) UpsertBusinessHours(h BusinessHours) error {
	h, err := normalizeHours(h)
	if err != nil {
		return err
	}
	closed := false
	err = m.store.Atomic(func(tx crud.Tx) error {
		row, found, err := firstRow(tx, func() model.Model { return &BusinessHours{} },
			storage.Eq(BusinessHours_.DayOfWeek, int64(h.DayOfWeek)))
		if err != nil {
			return err
		}
		h.UpdatedAt = time.NowNano()
		if !found {
			h.Id = m.ids.NewID()
			return tx.Create(&h)
		}
		previous := row.(*BusinessHours)
		closed = hoursCloseTime(*previous, h)
		h.Id = previous.Id
		return tx.UpdateFields(&h, previous.Id, []string{
			BusinessHours_.DayOfWeek, BusinessHours_.OpenMin, BusinessHours_.CloseMin,
			BusinessHours_.IsOpen, BusinessHours_.Notes, BusinessHours_.UpdatedAt,
		})
	})
	if err != nil {
		return err
	}
	m.publish(CalendarKindBusinessHours, closed, 0, 0)
	return nil
}

// AddHoliday registra un feriado; una segunda fila en la misma fecha se rechaza con ErrDuplicateDate.
// Ejemplo: err := cal.AddHoliday(h)
func (m *Module) AddHoliday(h Holiday) error {
	err := m.store.Atomic(func(tx crud.Tx) error {
		if err := checkNewHoliday(tx, &h); err != nil {
			return err
		}
		h.Id = m.ids.NewID()
		h.UpdatedAt = time.NowNano()
		if err := model.ValidateFields(model.ActionCreate, &h); err != nil {
			return err
		}
		return tx.Create(&h)
	})
	if err != nil {
		return err
	}
	m.publish(CalendarKindHoliday, true, h.SpecificDate, h.SpecificDate)
	return nil
}

// RemoveHoliday elimina un feriado por id y reabre su día para reservas.
// Ejemplo: err := cal.RemoveHoliday(id)
func (m *Module) RemoveHoliday(id string) error {
	var removed Holiday
	err := m.store.Atomic(func(tx crud.Tx) error {
		if err := tx.ReadOne(&removed, id); err != nil {
			return notFound(err)
		}
		return tx.Delete(&removed, id)
	})
	if err != nil {
		return err
	}
	m.publish(CalendarKindHoliday, false, removed.SpecificDate, removed.SpecificDate)
	return nil
}

// AddClosure registra un cierre que el propio establecimiento decidió; una segunda fila en la misma fecha se
// rechaza con ErrDuplicateDate.
// Ejemplo: err := cal.AddClosure(c)
func (m *Module) AddClosure(c Closure) error {
	err := m.store.Atomic(func(tx crud.Tx) error {
		if err := checkNewClosure(tx, &c); err != nil {
			return err
		}
		c.Id = m.ids.NewID()
		c.UpdatedAt = time.NowNano()
		if err := model.ValidateFields(model.ActionCreate, &c); err != nil {
			return err
		}
		return tx.Create(&c)
	})
	if err != nil {
		return err
	}
	m.publish(CalendarKindClosure, true, c.SpecificDate, c.SpecificDate)
	return nil
}

// RemoveClosure elimina un cierre por id.
// Ejemplo: err := cal.RemoveClosure(id)
func (m *Module) RemoveClosure(id string) error {
	var removed Closure
	err := m.store.Atomic(func(tx crud.Tx) error {
		if err := tx.ReadOne(&removed, id); err != nil {
			return notFound(err)
		}
		return tx.Delete(&removed, id)
	})
	if err != nil {
		return err
	}
	m.publish(CalendarKindClosure, false, removed.SpecificDate, removed.SpecificDate)
	return nil
}

// ListBusinessHours devuelve las siete reglas semanales ordenadas por id. Una semana vacía no es un error.
// Ejemplo: rows, err := cal.ListBusinessHours("", 0)
func (m *Module) ListBusinessHours(after string, limit int64) ([]BusinessHours, error) {
	var out []BusinessHours
	err := readByID(m.store, func() model.Model { return &BusinessHours{} }, nil, after, limit,
		func(r model.Model) string { row := r.(*BusinessHours); out = append(out, *row); return row.Id })
	return out, err
}

// ListHolidays devuelve los feriados en [from, to] (medianoches UTC en segundos, ambos incluidos; 0 = sin límite).
// Ejemplo: rows, err := cal.ListHolidays(0, 0, "", "", 0)
func (m *Module) ListHolidays(from, to int64, search, after string, limit int64) ([]Holiday, error) {
	var conditions []storage.Condition
	if from != 0 {
		conditions = append(conditions, storage.Gte(Holiday_.SpecificDate, from))
	}
	if to != 0 {
		conditions = append(conditions, storage.Lte(Holiday_.SpecificDate, to))
	}
	if search != "" {
		conditions = append(conditions, storage.Like(Holiday_.Name, "%"+search+"%"))
	}
	var out []Holiday
	err := readByID(m.store, func() model.Model { return &Holiday{} }, conditions, after, limit,
		func(r model.Model) string { row := r.(*Holiday); out = append(out, *row); return row.Id })
	return out, err
}

// ListClosures devuelve los cierres en [from, to] (misma convención que ListHolidays).
// Ejemplo: rows, err := cal.ListClosures(0, 0, "", "", 0)
func (m *Module) ListClosures(from, to int64, search, after string, limit int64) ([]Closure, error) {
	var conditions []storage.Condition
	if from != 0 {
		conditions = append(conditions, storage.Gte(Closure_.SpecificDate, from))
	}
	if to != 0 {
		conditions = append(conditions, storage.Lte(Closure_.SpecificDate, to))
	}
	if search != "" {
		conditions = append(conditions, storage.Like(Closure_.Reason, "%"+search+"%"))
	}
	var out []Closure
	err := readByID(m.store, func() model.Model { return &Closure{} }, conditions, after, limit,
		func(r model.Model) string { row := r.(*Closure); out = append(out, *row); return row.Id })
	return out, err
}
```
Add `"webtyp.com/storage"` to the imports. `businessHoursByDay` moves to `hours.go` (Stage 4). Everything here compiles after Stage 3.

## 2.3 `store.go` (new)

```go
package businesscalendar

import "webtyp.com/crud"

// StoreBusinessHours is the store of the weekly rules.
func (m *Module) StoreBusinessHours() crud.Store { return m.store }

// StoreHoliday is the store of the holidays.
func (m *Module) StoreHoliday() crud.Store { return m.store }

// StoreClosure is the store of the closures.
func (m *Module) StoreClosure() crud.Store { return m.store }
```

## 2.4 `rows.go` (new), written exactly

```go
package businesscalendar

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

// firstRow answers the first row that matches conds, and whether there is one.
//
// Example: row, found, err := firstRow(tx, func() model.Model { return &Holiday{} }, storage.Eq(Holiday_.SpecificDate, day))
func firstRow(tx crud.Tx, newRow func() model.Model, conds ...storage.Condition) (model.Model, bool, error) {
	var first model.Model
	err := tx.ReadAll(newRow, crud.Query{Conditions: conds, Limit: 1}, func(r model.Model) {
		if first == nil {
			first = r
		}
	})
	return first, first != nil, err
}

// readByID reads the rows that match conds ordered by id, after the row `after`, up to limit rows (limit <= 0 means all
// of them), paging because one read answers at most crud.MaxPage rows. add receives each row and answers its id.
func readByID(tx crud.Tx, newRow func() model.Model, conds []storage.Condition, after string, limit int64, add func(model.Model) string) error {
	for {
		pageLimit := int64(crud.MaxPage)
		if limit > 0 && limit < pageLimit {
			pageLimit = limit
		}
		last, count := "", int64(0)
		err := tx.ReadAll(newRow, crud.Query{Conditions: conds, After: after, Limit: pageLimit}, func(r model.Model) {
			last = add(r)
			count++
		})
		if err != nil {
			return err
		}
		if limit > 0 {
			limit -= count
			if limit <= 0 {
				return nil
			}
		}
		if count < pageLimit {
			return nil
		}
		after = last
	}
}

// notFound turns the not-found error of a store into the module's ErrNotFound and leaves any other error as it is.
func notFound(err error) error {
	if model.KindOf(err) == model.ErrorNotFound {
		return ErrNotFound
	}
	return err
}
```

## 2.5 `checks.go` (new): the checks a rule and a kept method share (one place per rule), written exactly

```go
package businesscalendar

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

// normalizeHours checks a weekly rule and answers it in its single stored form: a closed day carries "00:00" twice.
func normalizeHours(h BusinessHours) (BusinessHours, error) {
	if h.DayOfWeek < 0 || h.DayOfWeek > 6 {
		return h, &model.FieldError{Field: BusinessHours_.DayOfWeek, Err: ErrInvalidDay}
	}
	if !h.IsOpen {
		h.OpenMin, h.CloseMin = "00:00", "00:00"
		return h, nil
	}
	openMin, openErr := parseClock(h.OpenMin)
	closeMin, closeErr := parseClock(h.CloseMin)
	if openErr != nil || closeErr != nil || openMin >= closeMin {
		return h, &model.FieldError{Field: BusinessHours_.OpenMin, Err: ErrInvalidMinutes}
	}
	return h, nil
}

// checkNewHoliday refuses a second holiday on the same date.
func checkNewHoliday(tx crud.Tx, h *Holiday) error {
	_, found, err := firstRow(tx, func() model.Model { return &Holiday{} }, storage.Eq(Holiday_.SpecificDate, h.SpecificDate))
	if err != nil {
		return err
	}
	if found {
		return ErrDuplicateDate
	}
	return nil
}

// checkNewClosure refuses a second closure on the same date.
func checkNewClosure(tx crud.Tx, c *Closure) error {
	_, found, err := firstRow(tx, func() model.Model { return &Closure{} }, storage.Eq(Closure_.SpecificDate, c.SpecificDate))
	if err != nil {
		return err
	}
	if found {
		return ErrDuplicateDate
	}
	return nil
}

// containsString answers whether list holds value.
func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
```
`hoursCloseTime` (from the old `module.go`: `func hoursCloseTime(prev, next BusinessHours) bool`) is NOT in the new `module.go`: move it, byte for byte, to the end of `checks.go`. Commit and push (the package compiles after Stage 3).
