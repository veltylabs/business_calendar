← [Stage 4](PLAN_STAGE_4_READERS.md) | Next → [Stage 6](PLAN_STAGE_6_TESTS_DOCS.md)

# Stage 5 — `view.go`, `module/`, demo, `migrate/`

## 5.1 `view.go` — replace the whole file with exactly

`view.Ops` no longer has `List`, `Save`, `Delete` or `Allow`; the operations come from the module and the entity.

```go
package businesscalendar

import (
	"webtyp.com/lang"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

// NewBusinessHoursView construye el Presenter de horas semanales.
//
// Ejemplo: pres := NewBusinessHoursView(caller)
func NewBusinessHoursView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(view.NewRecords().Caller(caller),
		view.Ops{Module: ModelName, Page: view.Page{First: view.All}},
		func() model.TableSlice { return &BusinessHoursList{} })
	allOpts := append([]view.Option{view.WithTitle(lang.Translate("Business hours").String())}, opts...)
	return view.New(b, &BusinessHours{}, allOpts...)
}

// NewHolidaysView construye el Presenter de feriados.
//
// Ejemplo: pres := NewHolidaysView(caller)
func NewHolidaysView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(view.NewRecords().Caller(caller),
		view.Ops{Module: ModelName, Page: view.Page{First: view.All}},
		func() model.TableSlice { return &HolidayList{} })
	allOpts := append([]view.Option{view.WithTitle(lang.Translate("Holidays").String())}, opts...)
	return view.New(b, &Holiday{}, allOpts...)
}

// NewClosuresView construye el Presenter de cierres.
//
// Ejemplo: pres := NewClosuresView(caller)
func NewClosuresView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(view.NewRecords().Caller(caller),
		view.Ops{Module: ModelName, Page: view.Page{First: view.All}},
		func() model.TableSlice { return &ClosureList{} })
	allOpts := append([]view.Option{view.WithTitle(lang.Translate("Closures").String())}, opts...)
	return view.New(b, &Closure{}, allOpts...)
}
```
If `view.NewCallerLister`/`view.Ops` differ in `view@v0.6.74`, run `go doc webtyp.com/view Ops` and copy https://github.com/veltylabs/patient_directory/blob/main/view.go; name the difference in the PR.

## 5.2 `module/` — KEEP the package (`mjosefa-cms` imports it). Two edits only

- `module/browser.go`: `appkit.ModuleBrowser` → `view.ModuleBrowser` and `appkit.BrowserDeps` → `view.BrowserDeps`:

```go
//go:build browser

package module

import (
	"github.com/veltylabs/business_calendar/ui"
	"webtyp.com/appkit"
	"webtyp.com/view"
)

var server appkit.ModuleServer

var browser = view.ModuleBrowser{
	Screens: func(d view.BrowserDeps) []view.Screen { return ui.Screens(d.Caller) },
}
```
- `module/server.go`: the line `var browser appkit.ModuleBrowser` becomes `var browser view.ModuleBrowser` and the file imports `webtyp.com/view`. `module/module.go` is unchanged.

## 5.3 `migrate/migrate.go`

The `Sync(...)` list is `&businesscalendar.BusinessHours{}, &businesscalendar.Holiday{}, &businesscalendar.Closure{}` (the read model is not a table). If it already is, leave the file.

## 5.4 Demo (replaces `web/browser/main.go`): `seed/demo.go`, `//go:build wasm`, written exactly; then `rm -r web` (`test ! -e web`)

```go
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

// Demo crea una demostración ejecutable del módulo business_calendar sobre una base en memoria con datos semilla.
//
// Ejemplo: demo, err := seed.Demo()
func Demo() (view.Demo, error) {
	db := orm.New(mem.New())
	ids, err := unixid.NewUnixID()
	if err != nil {
		return view.Demo{}, err
	}
	mod, err := businesscalendar.New(db, businesscalendar.Deps{IDs: ids})
	if err != nil {
		return view.Demo{}, err
	}
	if _, err := Load(mod); err != nil {
		return view.Demo{}, err
	}
	return view.Demo{
		Label:      ui.Label,
		Timezone:   "America/Santiago",
		ActingAs:   "u1",
		Modules:    []crud.Module{mod},
		Migrations: []ddl.Step{{Name: businesscalendar.ModelName, Run: migrate.Migrate}},
		Screens: func(d view.BrowserDeps) []view.Screen {
			return ui.Screens(d.Caller)
		},
	}, nil
}
```
Read `git show main:web/browser/main.go` for anything else the old demo wired (a clock, a publisher) and keep only what `Demo` needs.

## 5.5 Greps (all empty)

`grep -rn "webtyp.com/layout\|appkit.BrowserDeps\|appkit.ModuleBrowser\|router/loopback" --include=*.go . | grep -v "^./ui/"`. `gotest` compiles on every destination now except the old tests Stage 6 rewrites. Commit and push.
