package ui

import (
	businesscalendar "github.com/veltylabs/business_calendar"
	"webtyp.com/lang"
	"webtyp.com/model"
	"webtyp.com/view"
)

// Identidad de cada pantalla, para que la app las ponga en su menú.
//
// Ejemplo: Screens: []string{ui.HoursID, ui.HolidaysID, ui.ClosuresID}
const (
	HoursID    = ID + "." + TabHoursID
	HolidaysID = ID + "." + TabHolidaysID
	ClosuresID = ID + "." + TabClosuresID
)

const (
	TabHoursID       = "hours"
	TabHoursLabel    = "Horario"
	TabHolidaysID    = "holidays"
	TabHolidaysLabel = "Feriados"
	TabClosuresID    = "closures"
	TabClosuresLabel = "Cierres"
)

const (
	descCalendarGroup lang.Text = "Business calendar: opening hours, official holidays and scheduled closures."
	descHours         lang.Text = "Regular weekly business hours and schedule by day of week."
	descHolidays      lang.Text = "Public and official holidays where the business remains closed."
	descClosures      lang.Text = "Special scheduled closures and extraordinary non-working days."
)

// Screens declara las pantallas que este módulo ofrece al menú de la app.
//
// Ejemplo: screens := ui.Screens(caller)
func Screens(d view.BrowserDeps) []view.Screen {
	caller := d.Caller
	return []view.Screen{{
		ID:    ID,
		Label: Label,
		Icon:  ID,
		Help:  descCalendarGroup,
		Children: []view.Screen{
			{
				ID:         HoursID,
				Label:      TabHoursLabel,
				Icon:       ID,
				Permission: model.ResourceOf(&businesscalendar.BusinessHoursModel),
				Presenter:  businesscalendar.NewBusinessHoursView(caller, view.WithDescription(descHours)),
			},
			{
				ID:         HolidaysID,
				Label:      TabHolidaysLabel,
				Icon:       ID,
				Permission: model.ResourceOf(&businesscalendar.HolidayModel),
				Presenter:  businesscalendar.NewHolidaysView(caller, view.WithDescription(descHolidays)),
			},
			{
				ID:         ClosuresID,
				Label:      TabClosuresLabel,
				Icon:       ID,
				Permission: model.ResourceOf(&businesscalendar.ClosureModel),
				Presenter:  businesscalendar.NewClosuresView(caller, view.WithDescription(descClosures)),
			},
		},
	}}
}
