//go:build browser

package ui

import (
	businesscalendar "github.com/veltylabs/business_calendar"
	"webtyp.com/layout/crudview"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/svg"
)

const (
	TabHoursID       = "hours"
	TabHoursLabel    = "Horario"
	TabHolidaysID    = "holidays"
	TabHolidaysLabel = "Feriados"
	TabClosuresID    = "closures"
	TabClosuresLabel = "Cierres"
)

// Browser composes this module's three own screens (business hours, holidays,
// closures) into one navigation group, because this module has no single Presenter
// (it has three). Each carries its own ParentID: three crudviews on the same screen,
// each with its own form mounted, cannot share the id they mount on. tenantID is
// unused — see server.go's comment on the same parameter.
func Browser(caller router.Caller, ids model.IDGenerator, tenantID string) (platformd.UIModule, error) {
	hoursView, err := crudview.New(crudview.Config{
		ParentID:  ID + "." + TabHoursID,
		Presenter: businesscalendar.NewBusinessHoursView(caller),
		IDs:       ids,
	})
	if err != nil {
		return nil, err
	}
	holidaysView, err := crudview.New(crudview.Config{
		ParentID:  ID + "." + TabHolidaysID,
		Presenter: businesscalendar.NewHolidaysView(caller),
		IDs:       ids,
	})
	if err != nil {
		return nil, err
	}
	closuresView, err := crudview.New(crudview.Config{
		ParentID:  ID + "." + TabClosuresID,
		Presenter: businesscalendar.NewClosuresView(caller),
		IDs:       ids,
	})
	if err != nil {
		return nil, err
	}

	hoursMod := platformd.NewUIModule(ID+"."+TabHoursID, TabHoursLabel, svg.Icon(ID), hoursView)
	holidaysMod := platformd.NewUIModule(ID+"."+TabHolidaysID, TabHolidaysLabel, svg.Icon(ID), holidaysView)
	closuresMod := platformd.NewUIModule(ID+"."+TabClosuresID, TabClosuresLabel, svg.Icon(ID), closuresView)

	return platformd.NewGroupUIModule(ID, Label, svg.Icon(ID), []platformd.UIModule{
		hoursMod, holidaysMod, closuresMod,
	}), nil
}
