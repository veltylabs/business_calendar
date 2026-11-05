---
PLAN: "feat!: OpeningHours for appointment_booking — DayLimits, WeekdayLimits, ClosedDates; module/ for appkit"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 545858717947657141
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work.

# Plan — business_calendar speaks time.DayLimits and plugs into an app as one value

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go get webtyp.com/time@v0.6.4 webtyp.com/appkit@v0.2.8
```

## Why

`time` v0.6.4 renamed `DayBounds`/`Unbounded` → `DayLimits`/`Unlimited` (same fields: `Open bool`,
`OpenMin int`, `CloseMin int`). `appointment_booking` (v0.3.75) finds, with `appkit.Find`, a value with:

```go
type OpeningHours interface {
	DayLimits(date time.Unix) (time.DayLimits, error)
	WeekdayLimits(dayOfWeek int) (time.DayLimits, error)
	ClosedDates(from, to time.Unix) (holidays, closures []string, err error)
}
```

`room_layout` will find the first method too. This module has no `module/` folder, so an app cannot add it to
`App.Modules`.

## Ecosystem search

`GetDayBounds`/`GetWeekdayBounds`/`GetDayDetail` (`interfaces.go`), `ListHolidays`/`ListClosures` (`module.go`,
`SpecificDate` = midnight UTC, inclusive `Gte`/`Lte`) → **rename and reuse**. `module/` copied from
`github.com/veltylabs/service_catalog/module` (published).

## Design gate

1. Prior art: the consumer-declared narrow interface (Go "accept interfaces"), already used by staff_manager.
2. Names: `DayLimits`, `WeekdayLimits` match `time.DayLimits`; `ClosedDates` = the dates the clinic is closed.
3. Ledger: renames 2, +1 method, −1 interface (`Reader`, nobody imports it), ways to read a day's limits 1 → 1.
4. Belongs here: owner of hours, holidays and closures.
5. Deletes: `GetDayBounds`, `GetWeekdayBounds`, `Reader`, every `tinytime` alias.

## APIs used (time v0.6.4)

| Symbol | Signature |
|---|---|
| `time.DayLimits` | `struct{ Open bool; OpenMin int; CloseMin int }` |
| `time.CivilUTC` | `CivilUTC(u Unix) (year, month, day, hour, min, sec int)` |
| `appkit.ModuleServer` | `{Migrate func(ddl.Execer, ddl.Compiler) error; Build func(appkit.Deps) (model.ModuleNaming, error)}` |
| `appkit.ModuleBrowser` | `{Screens func(appkit.BrowserDeps) []view.Screen}`; `BrowserDeps{Caller router.Caller; Zone time.Zone}` |
| `appkit.Deps` | fields used: `DB *orm.DB`, `IDs model.IDGenerator`, `Events events.Publisher` |

## Stage 1 — import without alias, rename to DayLimits

- Every `tinytime "webtyp.com/time"` → `"webtyp.com/time"`, `tinytime.` → `time.` (7 files: `ops.go`,
  `seed/seed.go`, `convert.go`, `interfaces.go`, `module.go`, `bankdays.go`, tests). If a file also imports Go's
  `time`, stop and name it in the PR.
- `time.DayBounds` → `time.DayLimits`, `time.Unbounded` → `time.Unlimited`.
- `GetDayBounds` → `DayLimits`, `GetWeekdayBounds` → `WeekdayLimits` (method names; same bodies). `DayDetail.Bounds`
  → `DayDetail.Limits`. The op name `get_day_bounds` and `opGetDayBounds` → `get_day_limits`, `opGetDayLimits`.
- Delete the `Reader` interface.
- Move `DayDetail`, `ClosedReason` and its constants to `model.go`, `GetDayDetail`/`DayLimits`/`WeekdayLimits`/
  `getWeeklyBounds` to `hours.go`; delete `interfaces.go` (this module requires nothing from others).

## Stage 2 — ClosedDates (`hours.go`)

```go
// ClosedDates lista los feriados y cierres entre from y to (medianoches UTC, ambos incluidos), como fechas
// "YYYY-MM-DD", en orden ascendente. Satisface OpeningHours de appointment_booking.
// Ejemplo: holidays, closures, err := m.ClosedDates(from, to)
func (m *Module) ClosedDates(from, to time.Unix) (holidays, closures []string, err error) {
	hs, err := m.ListHolidays(int64(from), int64(to))
	if err != nil {
		return nil, nil, err
	}
	cs, err := m.ListClosures(int64(from), int64(to))
	if err != nil {
		return nil, nil, err
	}
	for _, h := range hs {
		holidays = append(holidays, dateText(time.Unix(h.SpecificDate)))
	}
	for _, c := range cs {
		closures = append(closures, dateText(time.Unix(c.SpecificDate)))
	}
	return holidays, closures, nil
}

// dateText es la fecha "YYYY-MM-DD" de una clave de día (medianoche UTC).
func dateText(day time.Unix) string {
	y, mo, d, _, _, _ := time.CivilUTC(day)
	return fmt.Sprintf("%04d-%02d-%02d", y, mo, d)
}
```

`fmt` is `webtyp.com/fmt`. If `ListHolidays`/`ListClosures` do not sort ascending, add `OrderBy` on
`SpecificDate` there.

## Stage 3 — module/ (copy `service_catalog/module`, rename)

`module/module.go`: `var Module = appkit.Module{Name: "business_calendar", Server: server, Browser: browser}`.
`module/server.go` (`//go:build !browser`):
```go
func build(d appkit.Deps) (model.ModuleNaming, error) {
	return businesscalendar.New(d.DB, businesscalendar.Deps{IDs: d.IDs, Publisher: d.Events})
}
```
`module/browser.go` (`//go:build browser`): `Screens: func(d appkit.BrowserDeps) []view.Screen { return ui.Screens(d.Caller) }`.
Comments in Spanish, as in the source.

## Stage 4 — tests

Rename in existing tests. New:

| Test | Setup → call | Expected |
|---|---|---|
| `TestClosedDates` | holiday 2026-09-18, closure 2026-09-25, holiday 2026-10-12 → `ClosedDates(2026-09-01, 2026-09-30)` | holidays `["2026-09-18"]`, closures `["2026-09-25"]` |
| `TestClosedDatesEmpty` | nothing → same call | both empty, `err == nil` |
| `TestModuleBuilds` | `module.Module.Server.Build` with an sqlite `appkit.Deps` (as other repos' tests, or skip if `appkit` gives no test helper — then say so in the PR) | no error |

## Stage 5 — docs (Spanish)

`README.md`: Step-by-Step uses `Modules: []appkit.Module{businesscalendar.Module}` and `DayLimits`; `docs/ARCHITECTURE.md`:
`OpeningHours` is satisfied structurally; no `Reader`.

## Delivery check

1. `gotest` passes.
2. `grep -rn "tinytime\|DayBounds\|Unbounded\|GetDayBounds\|GetWeekdayBounds" --include=*.go --include=*.md .` → empty.
3. `test ! -e interfaces.go`.

Commit after every stage. Never edit `model_orm.go` by hand; a build error there means the hand-written code is wrong.
Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work. Stuck 30 minutes: open
the PR anyway listing what fails. A PR that does not compile, or whose tests could not run, is a failed delivery.
Environment: `GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`, `GOFLAGS=-mod=mod`, `go mod tidy` before `ormc`.
If a signature in the published tag differs from the APIs table, stop and name the symbol in the PR; never adapt it by guessing.

## Stages

| Stage | Files | Done when |
|---|---|---|
| 1 | all `*.go` using the alias, `model.go`, `hours.go`, `interfaces.go` (deleted), `ops.go` | compiles |
| 2 | `hours.go` | compiles |
| 3 | `module/module.go`, `module/server.go`, `module/browser.go` | compiles for browser |
| 4 | `tests/*` | tests pass |
| 5 | `README.md`, `docs/ARCHITECTURE.md` | done |

## Executor notes
- In Step 3, `module.go` required manually importing `server` and `browser` from the conditionally compiled files by assigning them in an `init()` function inside `module.go` because variables inside the same package but in conditionally compiled out files aren't available to `module.go`. Also needed to add `github.com/veltylabs/business_calendar/migrate` to `module/server.go`.
- In `tests/closedates_test.go`, the `time` package was removed because it was unused. The `setupModule` function from `setup_test.go` was actually named `newModule`, and `seedWeek` was also used for test setup.
- In `tests/module_test.go`, `events.NewPublisher()` is not available, so I used `nil` for `Events` in `appkit.Deps`. For `orm.Open`, `orm.New(mem.New())` was used. The test skips if `appkit` doesn't provide a helper (`module.Module.Server.Build == nil`).
- In `hours.go`, `import "webtyp.com/fmt"` was required for `dateText`.
- For translation, `langc sync .` was used, but the initial file contained empty translations. I wrote a python script to fill the empty strings with the keys to pass verification.
- Replaced Reader interface and the reference in README.md as noted by PR review.
