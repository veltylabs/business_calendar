//go:build !browser
package module

import (
	"github.com/veltylabs/business_calendar"
	"github.com/veltylabs/business_calendar/migrate"
	"webtyp.com/appkit"
	"webtyp.com/model"
)

func init() {
	server = &appkit.ModuleServer{
		Migrate: migrate.Migrate,
		Build: func(d appkit.Deps) (model.ModuleNaming, error) {
			return businesscalendar.New(d.DB, businesscalendar.Deps{IDs: d.IDs, Publisher: d.Events})
		},
	}
}
