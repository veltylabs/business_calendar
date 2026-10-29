package businesscalendar

import (
	"errors"
	"webtyp.com/model"

	"webtyp.com/rpc"
	"webtyp.com/router"
	tinytime "webtyp.com/time"
	"webtyp.com/view"
)

const (
	OpListBusinessHours   = "list_business_hours"
	OpUpsertBusinessHours = "upsert_business_hours"
	OpListHolidays        = "list_holidays"
	OpAddHoliday          = "add_holiday"
	OpRemoveHoliday       = "remove_holiday"
	OpListClosures        = "list_closures"
	OpAddClosure          = "add_closure"
	OpRemoveClosure       = "remove_closure"
	OpGetDayBounds        = "get_day_bounds"

	DescListBusinessHours   = "Horario de atención del consultorio para cada día de la semana: día (Monday … Sunday), si abre, y la hora de apertura y de cierre (HH:MM)."
	DescUpsertBusinessHours = "Crea o cambia el horario de atención de un día de la semana."
	DescGetDayBounds        = "Si el consultorio abre en una fecha y en qué minutos del día, considerando feriados y cierres."
	DescListHolidays        = "Feriados registrados del consultorio."
	DescAddHoliday          = "Registra un feriado: el consultorio no atiende esa fecha."
	DescRemoveHoliday       = "Elimina un feriado registrado."
	DescListClosures        = "Cierres extraordinarios del consultorio (fechas u horas sin atención)."
	DescAddClosure          = "Registra un cierre extraordinario."
	DescRemoveClosure       = "Elimina un cierre extraordinario."
)

func (m *Module) MountOperations(reg router.OperationRegistry) {
	reg.Operation(OpListBusinessHours, m.opListBusinessHours).
		Describe(DescListBusinessHours).
		Requires(model.ResourceOf(&BusinessHoursModel), model.Read).
		Accepts(nil)
	// Upsert creates on the not-found branch AND updates otherwise — it must
	// declare BOTH actions (model.Action is a bitmask). Declaring only Update
	// would let an update-only principal create rows.
	view.MountSave(reg, OpUpsertBusinessHours, func(h *BusinessHours) error {
		if err := m.UpsertBusinessHours(*h); err != nil {
			return err
		}
		saved, err := m.businessHoursByDay(int(h.DayOfWeek))
		if err == nil {
			*h = saved
		}
		return err
	}).Describe(DescUpsertBusinessHours).Requires(model.ResourceOf(&BusinessHoursModel), model.Create|model.Update)
	reg.Operation(OpGetDayBounds, m.opGetDayBounds).
		Describe(DescGetDayBounds).
		Requires(model.ResourceOf(&BusinessHoursModel), model.Read).
		Accepts(&GetDayBoundsArgs{})

	reg.Operation(OpListHolidays, m.opListHolidays).
		Describe(DescListHolidays).
		Requires(model.ResourceOf(&HolidayModel), model.Read).
		Accepts(nil)
	view.MountSave(reg, OpAddHoliday, func(h *Holiday) error {
		if err := m.AddHoliday(*h); err != nil {
			return err
		}
		rows, err := m.ListHolidays(h.SpecificDate, h.SpecificDate)
		if err == nil && len(rows) > 0 {
			*h = rows[0]
		}
		return err
	}).Describe(DescAddHoliday).Requires(model.ResourceOf(&HolidayModel), model.Create)
	view.MountDelete(reg, OpRemoveHoliday, func(ids []string) error {
		for _, id := range ids {
			if err := m.RemoveHoliday(id); err != nil {
				return err
			}
		}
		return nil
	}).Describe(DescRemoveHoliday).Requires(model.ResourceOf(&HolidayModel), model.Delete)

	reg.Operation(OpListClosures, m.opListClosures).
		Describe(DescListClosures).
		Requires(model.ResourceOf(&ClosureModel), model.Read).
		Accepts(nil)
	view.MountSave(reg, OpAddClosure, func(c *Closure) error {
		if err := m.AddClosure(*c); err != nil {
			return err
		}
		rows, err := m.ListClosures(c.SpecificDate, c.SpecificDate)
		if err == nil && len(rows) > 0 {
			*c = rows[len(rows)-1]
		}
		return err
	}).Describe(DescAddClosure).Requires(model.ResourceOf(&ClosureModel), model.Create)
	view.MountDelete(reg, OpRemoveClosure, func(ids []string) error {
		for _, id := range ids {
			if err := m.RemoveClosure(id); err != nil {
				return err
			}
		}
		return nil
	}).Describe(DescRemoveClosure).Requires(model.ResourceOf(&ClosureModel), model.Delete)
}

var _ router.OperationModule = (*Module)(nil)

// writeError maps known sentinels to a status and writes the message. A real
// internal error is never collapsed into 404 — it propagates as the 500 it is.


func (m *Module) opListBusinessHours(ctx router.Context) {
	rows, err := m.ListBusinessHours()
	if err != nil {
		rpc.WriteError(ctx, errors.New("internal error"))
		return
	}
	list := make(businessHoursViewList, len(rows))
	for i := range rows {
		list[i] = &businessHoursView{row: &rows[i]}
	}
	if err := ctx.Encode(&list); err != nil {
		rpc.WriteError(ctx, errors.New("internal error"))
	}
}

func (m *Module) opGetDayBounds(ctx router.Context) {
	var args GetDayBoundsArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	// GetDayDetail, not GetDayBounds: an over-the-wire caller (this module's
	// own UI, an admin tool) gets ClosedBy too. A Go-level sibling that only
	// wants the neutral bounds calls GetDayBounds directly through Reader —
	// it never goes through this op.
	detail, err := m.GetDayDetail(tinytime.Unix(args.Date))
	if err != nil {
		rpc.WriteError(ctx, errors.New("internal error"))
		return
	}
	if err := ctx.Encode(&detail); err != nil {
		rpc.WriteError(ctx, errors.New("internal error"))
	}
}

func (m *Module) opListHolidays(ctx router.Context) {
	// No args — lists the whole horizon. Range filtering lives on the service
	// method (ListHolidays(from, to)) for in-process consumers; a view lists all.
	rows, err := m.ListHolidays(0, 0)
	if err != nil {
		rpc.WriteError(ctx, errors.New("internal error"))
		return
	}
	list := make(HolidayList, len(rows))
	for i := range rows {
		list[i] = &rows[i]
	}
	if err := ctx.Encode(&list); err != nil {
		rpc.WriteError(ctx, errors.New("internal error"))
	}
}

func (m *Module) opListClosures(ctx router.Context) {
	// No args — lists the whole horizon. Range filtering lives on the service
	// method (ListClosures(from, to)) for in-process consumers; a view lists all.
	rows, err := m.ListClosures(0, 0)
	if err != nil {
		rpc.WriteError(ctx, errors.New("internal error"))
		return
	}
	list := make(ClosureList, len(rows))
	for i := range rows {
		list[i] = &rows[i]
	}
	if err := ctx.Encode(&list); err != nil {
		rpc.WriteError(ctx, errors.New("internal error"))
	}
}

