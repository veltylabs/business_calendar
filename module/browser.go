//go:build browser
package module

import (
	"webtyp.com/appkit"
	"webtyp.com/view"
	"github.com/veltylabs/business_calendar/ui"
)

var browser = &appkit.ModuleBrowser{
	Screens: func(d appkit.BrowserDeps) []view.Screen { return ui.Screens(d.Caller) },
}
