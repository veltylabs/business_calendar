//go:build !browser
package module

import (
	"webtyp.com/appkit"
	"github.com/veltylabs/business_calendar"
	"webtyp.com/model"
)

var server = &appkit.ModuleServer{
	Migrate: businesscalendar.Migrate,
	Build: func(d appkit.Deps) (model.ModuleNaming, error) {
		return businesscalendar.New(d.DB, businesscalendar.Deps{IDs: d.IDs, Publisher: d.Events})
	},
}
