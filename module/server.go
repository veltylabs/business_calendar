//go:build !browser

package module

import (
	businesscalendar "github.com/veltylabs/business_calendar"
	"webtyp.com/appkit"
	"webtyp.com/model"
)

var server = appkit.ModuleServer{
	Build: func(d appkit.Deps) (model.ModuleNaming, error) {
		return businesscalendar.New(d.DB, businesscalendar.Deps{IDs: d.IDs, Publisher: d.Events})
	},
}

var browser appkit.ModuleBrowser
