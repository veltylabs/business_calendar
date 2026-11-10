//go:build wasm

package seed
import (
	businesscalendar "github.com/veltylabs/business_calendar"
	"github.com/veltylabs/business_calendar/migrate"
	"github.com/veltylabs/business_calendar/ui"
	"webtyp.com/crud"
	"webtyp.com/ddl"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
	"webtyp.com/unixid"
	"webtyp.com/view"
)
func Demo() (view.Demo, error) {
	db := orm.New(mem.New())
	ids, err := unixid.NewUnixID()
	if err != nil { return view.Demo{}, err }
	mod, err := businesscalendar.New(db, businesscalendar.Deps{IDs: ids})
	if err != nil { return view.Demo{}, err }
	if _, err := Load(mod); err != nil { return view.Demo{}, err }
	return view.Demo{
		Label:      ui.Label,
		Timezone:   "America/Santiago",
		ActingAs:   "u1",
		Modules:    []crud.Module{mod},
		Migrations: []ddl.Step{{Name: businesscalendar.ModelName, Run: migrate.Migrate}},
		Screens: func(d view.BrowserDeps) []view.Screen { return ui.Screens(d.Caller) },
	}, nil
}
