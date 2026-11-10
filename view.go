package businesscalendar

import (
	"webtyp.com/lang"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

// NewBusinessHoursView construye el Presenter de horas semanales.
//
// Ejemplo: pres := NewBusinessHoursView(caller)
func NewBusinessHoursView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(view.NewRecords().Caller(caller),
		view.Ops{Module: ModelName, Page: view.Page{First: view.All}},
		func() model.TableSlice { return &BusinessHoursList{} })
	allOpts := append([]view.Option{view.WithTitle(lang.Translate("Business hours").String())}, opts...)
	return view.New(b, &BusinessHours{}, allOpts...)
}

// NewHolidaysView construye el Presenter de feriados.
//
// Ejemplo: pres := NewHolidaysView(caller)
func NewHolidaysView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(view.NewRecords().Caller(caller),
		view.Ops{Module: ModelName, Page: view.Page{First: view.All}},
		func() model.TableSlice { return &HolidayList{} })
	allOpts := append([]view.Option{view.WithTitle(lang.Translate("Holidays").String())}, opts...)
	return view.New(b, &Holiday{}, allOpts...)
}

// NewClosuresView construye el Presenter de cierres.
//
// Ejemplo: pres := NewClosuresView(caller)
func NewClosuresView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(view.NewRecords().Caller(caller),
		view.Ops{Module: ModelName, Page: view.Page{First: view.All}},
		func() model.TableSlice { return &ClosureList{} })
	allOpts := append([]view.Option{view.WithTitle(lang.Translate("Closures").String())}, opts...)
	return view.New(b, &Closure{}, allOpts...)
}
