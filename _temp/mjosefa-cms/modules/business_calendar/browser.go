package business_calendar

import (
	businesscalendar "github.com/veltylabs/business_calendar"
	"webtyp.com/components/decktabs"
	"webtyp.com/layout/crudview"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/svg"
)

// Browser composes this module's three own screens (business hours,
// holidays, closures) into a single nav item with decktabs — not a plain
// crudview.New like the other modules, because this module has no single
// Presenter (it has three). Each carries its own ParentID: three crudviews
// on the same screen, each with its own form mounted, cannot share the id
// they mount on. tenantID is unused — see server.go's comment on the same
// parameter.
func Browser(caller router.Caller, ids model.IDGenerator, tenantID string) (platformd.UIModule, error) {
	hoursView, err := crudview.New(crudview.Config{
		ParentID:  ID + ".hours",
		Presenter: businesscalendar.NewBusinessHoursView(caller),
		IDs:       ids,
	})
	if err != nil {
		return nil, err
	}
	holidaysView, err := crudview.New(crudview.Config{
		ParentID:  ID + ".holidays",
		Presenter: businesscalendar.NewHolidaysView(caller),
		IDs:       ids,
	})
	if err != nil {
		return nil, err
	}
	closuresView, err := crudview.New(crudview.Config{
		ParentID:  ID + ".closures",
		Presenter: businesscalendar.NewClosuresView(caller),
		IDs:       ids,
	})
	if err != nil {
		return nil, err
	}

	tabs := &decktabs.DeckTabs{
		Label: "Calendario",
		Items: []decktabs.Item{
			{ID: "hours", Label: "Horario", Panel: hoursView},
			{ID: "holidays", Label: "Feriados", Panel: holidaysView},
			{ID: "closures", Label: "Cierres", Panel: closuresView},
		},
	}
	return platformd.NewUIModule(ID, Label, svg.Icon(ID), tabs), nil
}
