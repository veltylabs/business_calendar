package businesscalendar

import (
	"webtyp.com/model"
	"webtyp.com/router"
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
)

func (m *Module) MountOperations(reg router.OperationRegistry) {
	reg.Operation(OpListBusinessHours, m.opListBusinessHours).
		Requires("business_hours", model.Read).
		Accepts(nil)
	// Upsert creates on the not-found branch AND updates otherwise — it must
	// declare BOTH actions (model.Action is a bitmask). Declaring only Update
	// would let an update-only principal create rows.
	reg.Operation(OpUpsertBusinessHours, m.opUpsertBusinessHours).
		Requires("business_hours", model.Create|model.Update).
		Accepts(&UpsertBusinessHoursArgs{})
	reg.Operation(OpGetDayBounds, m.opGetDayBounds).
		Requires("business_hours", model.Read).
		Accepts(&GetDayBoundsArgs{})

	reg.Operation(OpListHolidays, m.opListHolidays).
		Requires("holiday", model.Read).
		Accepts(nil)
	reg.Operation(OpAddHoliday, m.opAddHoliday).
		Requires("holiday", model.Create).
		Accepts(&AddHolidayArgs{})
	reg.Operation(OpRemoveHoliday, m.opRemoveHoliday).
		Requires("holiday", model.Delete).
		Accepts(&RemoveHolidayArgs{})

	reg.Operation(OpListClosures, m.opListClosures).
		Requires("closure", model.Read).
		Accepts(nil)
	reg.Operation(OpAddClosure, m.opAddClosure).
		Requires("closure", model.Create).
		Accepts(&AddClosureArgs{})
	reg.Operation(OpRemoveClosure, m.opRemoveClosure).
		Requires("closure", model.Delete).
		Accepts(&RemoveClosureArgs{})
}

var _ router.OperationModule = (*Module)(nil)

// writeError maps known sentinels to a status and writes the message. A real
// internal error is never collapsed into 404 — it propagates as the 500 it is.
func writeError(ctx router.Context, err error) {
	switch err {
	case ErrNotFound:
		ctx.WriteStatus(404)
	case ErrDuplicateDate:
		ctx.WriteStatus(409)
	case ErrInvalidDay, ErrInvalidMinutes:
		ctx.WriteStatus(400)
	default:
		ctx.WriteStatus(500)
	}
	ctx.Write([]byte(err.Error()))
}

func (m *Module) opListBusinessHours(ctx router.Context) {
	rows, err := m.ListBusinessHours()
	if err != nil {
		ctx.WriteStatus(500)
		return
	}
	list := make(BusinessHoursList, len(rows))
	for i := range rows {
		list[i] = &rows[i]
	}
	if err := ctx.Encode(&list); err != nil {
		ctx.WriteStatus(500)
	}
}

func (m *Module) opUpsertBusinessHours(ctx router.Context) {
	var args UpsertBusinessHoursArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	h := BusinessHours{
		DayOfWeek: args.DayOfWeek,
		OpenMin:   args.OpenMin,
		CloseMin:  args.CloseMin,
		IsOpen:    args.IsOpen,
		Notes:     args.Notes,
	}
	if err := m.UpsertBusinessHours(h); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.WriteStatus(200)
}

func (m *Module) opGetDayBounds(ctx router.Context) {
	var args GetDayBoundsArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	bounds, err := m.GetDayBounds(args.Date)
	if err != nil {
		ctx.WriteStatus(500)
		return
	}
	if err := ctx.Encode(&bounds); err != nil {
		ctx.WriteStatus(500)
	}
}

func (m *Module) opListHolidays(ctx router.Context) {
	// No args — lists the whole horizon. Range filtering lives on the service
	// method (ListHolidays(from, to)) for in-process consumers; a view lists all.
	rows, err := m.ListHolidays(0, 0)
	if err != nil {
		ctx.WriteStatus(500)
		return
	}
	list := make(HolidayList, len(rows))
	for i := range rows {
		list[i] = &rows[i]
	}
	if err := ctx.Encode(&list); err != nil {
		ctx.WriteStatus(500)
	}
}

func (m *Module) opAddHoliday(ctx router.Context) {
	var args AddHolidayArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	if err := args.Validate(model.ActionCreate); err != nil {
		ctx.WriteStatus(400)
		return
	}
	h := Holiday{SpecificDate: args.SpecificDate, Name: args.Name, Notes: args.Notes}
	if err := m.AddHoliday(h); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.WriteStatus(200)
}

func (m *Module) opRemoveHoliday(ctx router.Context) {
	var args RemoveHolidayArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	if err := m.RemoveHoliday(args.Id); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.WriteStatus(200)
}

func (m *Module) opListClosures(ctx router.Context) {
	// No args — lists the whole horizon. Range filtering lives on the service
	// method (ListClosures(from, to)) for in-process consumers; a view lists all.
	rows, err := m.ListClosures(0, 0)
	if err != nil {
		ctx.WriteStatus(500)
		return
	}
	list := make(ClosureList, len(rows))
	for i := range rows {
		list[i] = &rows[i]
	}
	if err := ctx.Encode(&list); err != nil {
		ctx.WriteStatus(500)
	}
}

func (m *Module) opAddClosure(ctx router.Context) {
	var args AddClosureArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	if err := args.Validate(model.ActionCreate); err != nil {
		ctx.WriteStatus(400)
		return
	}
	c := Closure{SpecificDate: args.SpecificDate, Reason: args.Reason}
	if err := m.AddClosure(c); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.WriteStatus(200)
}

func (m *Module) opRemoveClosure(ctx router.Context) {
	var args RemoveClosureArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	if err := m.RemoveClosure(args.Id); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.WriteStatus(200)
}
