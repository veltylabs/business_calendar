package module

import (
	"webtyp.com/appkit"
)

var Module = appkit.Module{
	Name: "business_calendar",
}

func init() {
	if server != nil {
		Module.Server = *server
	}
	if browser != nil {
		Module.Browser = *browser
	}
}

var server *appkit.ModuleServer
var browser *appkit.ModuleBrowser
