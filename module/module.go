package module

import (
	"webtyp.com/appkit"
)

var Module = appkit.Module{
	Name:    "business_calendar",
	Server:  server,
	Browser: browser,
}
