//go:build browser

package module

import (
	"github.com/veltylabs/business_calendar/ui"
	"webtyp.com/appkit"
	"webtyp.com/view"
)

var server appkit.ModuleServer

var browser = appkit.ModuleBrowser{
	Screens: func(d appkit.BrowserDeps) []view.Screen { return ui.Screens(d.Caller) },
}
