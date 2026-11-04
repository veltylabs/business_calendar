package module

import (
	"webtyp.com/appkit"
	"github.com/veltylabs/business_calendar/ui"
)

var Module = appkit.Module{
	Name: "business_calendar",
	Server: server,
	Browser: browser,
}
