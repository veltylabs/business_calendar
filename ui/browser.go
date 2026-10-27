//go:build browser

package ui

import (
	businesscalendar "github.com/veltylabs/business_calendar"
	"webtyp.com/lang"
	"webtyp.com/router"
	"webtyp.com/view"
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
func Screens(caller router.Caller) []view.Screen {
	return []view.Screen{{
		ID:    ID,
		Label: Label,
		Icon:  ID,
		Help:  descCalendarGroup,
		Children: []view.Screen{
			{
				ID:         ID + "." + TabHoursID,
				Label:      TabHoursLabel,
				Icon:       ID,
				Permission: "business_hours",
				Presenter:  businesscalendar.NewBusinessHoursView(caller, view.WithDescription(descHours)),
			},
			{
				ID:         ID + "." + TabHolidaysID,
				Label:      TabHolidaysLabel,
				Icon:       ID,
				Permission: "holiday",
				Presenter:  businesscalendar.NewHolidaysView(caller, view.WithDescription(descHolidays)),
			},
			{
				ID:         ID + "." + TabClosuresID,
				Label:      TabClosuresLabel,
				Icon:       ID,
				Permission: "closure",
				Presenter:  businesscalendar.NewClosuresView(caller, view.WithDescription(descClosures)),
			},
		},
	}}
}