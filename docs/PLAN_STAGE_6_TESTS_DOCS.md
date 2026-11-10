← [Stage 5](PLAN_STAGE_5_BROWSER.md)

# Stage 6 — tests, `docs/TESTS.md`, documentation

## 6.1 The existing tests

Because every exported method is kept, most tests stay as they are. Open each file once and apply exactly this table:

| File | Do |
|---|---|
| `bankdays_test.go`, `business_hours_test.go`, `closedates_test.go`, `closures_test.go`, `events_test.go`, `holidays_test.go`, `weekday_bounds_test.go`, `seed_test.go`, `errors_test.go`, `setup_test.go` | **keep**; they call the kept methods (`UpsertBusinessHours`, `AddHoliday`, `AddClosure`, `Remove*`, `List*`, `GetDayDetail`, `ClosedDates`…). Fix only what fails to compile or asserts an error that changed |
| `ops_test.go`, `ops_describe_test.go`, `hours_view_test.go`, `module_test.go` | **delete**: they test the handlers of the deleted `ops.go`, the deleted `businessHoursView` types and the removed `appkit.Deps`/`Server.Build` |
| `conformance_test.go`, `screen_save_test.go`, `ui_view_wasm_test.go` | **rewrite** the lines that use `router/loopback`, a string operation name or `view.Ops{List, Save, Delete}`: operations are `router.Operation{Module: businesscalendar.ModelName, Entity: "<entity>", Action: model.<Action>}`, callers are `crud.LoopbackActingAs(nil, "u1", mod)`; a `router.Caller` double has `Call(op router.Operation, args model.Encodable, into model.Decodable, done func(error))` and `Dispatch(op router.Operation, args model.Encodable)` |
| `migrate_test.go` | the double of the connection answers the new `Exec` contract: `func (c *migrateConn) Exec(string, ...any) (storage.Result, error) { return storage.Affected(0), nil }` (import `webtyp.com/storage`); the three `TestMigrate_*` keep their assertions |

A test is never skipped, and a test of a kept behaviour is never deleted.

## 6.2 New crud-level tests, `tests/crud_test.go` (package `tests`), with these helpers written exactly

```go
package tests

import (
	"testing"

	businesscalendar "github.com/veltylabs/business_calendar"
	"webtyp.com/crud"
	"webtyp.com/events"
	"webtyp.com/model"
	"webtyp.com/router"
)

// call runs one operation as `user` through the loopback the demo uses and answers the callback error. The module
// name is ALWAYS businesscalendar.ModelName, never a literal.
func call(user string, pub events.Publisher, mod *businesscalendar.Module, entity string, action model.Action, args model.Encodable, into model.Decodable) error {
	var callErr error
	crud.LoopbackActingAs(pub, user, mod).Call(
		router.Operation{Module: businesscalendar.ModelName, Entity: entity, Action: action},
		args, into, func(err error) { callErr = err })
	return callErr
}

// failedField answers the field a refusal points at, or "".
func failedField(err error) string {
	if f, ok := err.(interface{ FailedField() string }); ok {
		return f.FailedField()
	}
	return ""
}
```
Build the module with `newModule(t, pub)` of `tests/setup_test.go` (it returns the module and the fake id generator). Dates are the constants of that file (`sep08_2026_Tue`, `dec25_2026_Fri`…). Every Create record is sent WITHOUT `Id` (the entity declares `OmitEmpty` on `id`, `crud` stamps it). The first thing a test does with a callback error is assert it, before reading the result.

| Test (all in `tests/crud_test.go`) | setup → call → expected |
|---|---|
| `TestBusinessHours_SundayAndClosedAreAllowed` | `business_hours.create` `{DayOfWeek: 0, IsOpen: false}` → no error (the zero values must pass the schema) and the stored row has `OpenMin == "00:00"` and `CloseMin == "00:00"` |
| `TestBusinessHours_OneRowPerDay` | create `{DayOfWeek: 1, OpenMin: "08:00", CloseMin: "18:00", IsOpen: true}` twice → the second answers `model.KindOf(err) == model.ErrorConflict` |
| `TestBusinessHours_OpenBeforeClose` | create `{DayOfWeek: 2, OpenMin: "18:00", CloseMin: "08:00", IsOpen: true}` → `failedField(err) == "open_min"` |
| `TestBusinessHours_InvalidDay` | create `{DayOfWeek: 7, …}` → `failedField(err) == "day_of_week"` |
| `TestBusinessHours_UpdateOnlyListedFields` | create day 1 `08:00`–`18:00`; `business_hours.update` with `&crud.UpdateArgs{IDs: []string{id}, Fields: []string{"close_min"}, Record: &businesscalendar.BusinessHours{CloseMin: "17:00"}}` → no error; read the row (`&crud.ReadArgs{ID: id}`) → `OpenMin == "08:00"`, `CloseMin == "17:00"` |
| `TestBusinessHours_UpdateRejectsOtherFields` | `Fields: []string{"id"}` → `failedField(err) == "id"` |
| `TestBusinessHours_CloseDayNormalizesMinutes` | create day 3 open; update `Fields: []string{"is_open"}` with `IsOpen: false` → the stored row has `IsOpen == false` and `OpenMin == "00:00"`, `CloseMin == "00:00"` (the rule wrote them itself) |
| `TestHoliday_DuplicateDateRejected` | `holiday.create` `{SpecificDate: dec25_2026_Fri, Name: "Navidad"}` twice → second `errors` text contains `a record already exists for that date` and kind `model.ErrorConflict` |
| `TestHoliday_NoUpdateOperation` | `holiday.update` → error text contains `unknown operation` |
| `TestHoliday_DeleteReopensAndPublishes` | with a publisher given as the FIRST argument: create then `holiday.delete` (`&crud.DeleteArgs{IDs: []string{id}}`) → the publisher holds two `EventCalendarChanged` events, the first with `Payload.(*businesscalendar.CalendarChangedPayload).Closed == true` and the second `false` |
| `TestClosure_DuplicateDateRejected` | same as the holiday test with `closure.create` `{SpecificDate: …, Reason: "Maintenance"}` |
| `TestDayDetail_ClosedOnHoliday` | create a holiday on `sep12_2026_Sat`; `var list businesscalendar.DayStatusList`; `call("u1", nil, m, "day_detail", model.Read, &crud.ReadArgs{Filter: &businesscalendar.DayStatus{Date: sep12_2026_Sat}}, &list)` → one row with `Open == false` and `ClosedBy == "HOLIDAY"` |
| `TestDayDetail_NeedsDate` | the same read with an empty `DayStatus{}` → `failedField(err) == "date"` |
| `TestReaders_ClosedDates` | a holiday and a closure inside a range and one outside → `m.ClosedDates(from, to)` returns only the inside ones as `YYYY-MM-DD` |
| `TestMount_MatchesDefinition` | `crud.Mount(nil, nil, nil, m)` returns nil |
| `TestCreate_PublishesChange` | `holiday.create` with a publisher given as the FIRST argument → one event on `crud.Topic(businesscalendar.ModelName, (&businesscalendar.Holiday{}).ModelName(), model.Create)` whose payload is `*crud.Change` |
| `TestDemo_Valid` (`tests/demo_test.go`, `//go:build wasm`) | copy https://github.com/veltylabs/patient_directory/blob/main/tests/demo_test.go with the import path `github.com/veltylabs/business_calendar/seed` |

Events reach the publisher given to `crud.LoopbackActingAs(pub, user, mod)` AND to `Deps.Publisher`; in these tests give the same recording publisher to both when a test counts `EventCalendarChanged` events (the rules publish through `Deps.Publisher`; `crud` publishes its own `*crud.Change` through the loopback's).

## 6.3 `docs/TESTS.md`

One row per `func Test…` of the repo (old and new), header `| Test | Rule |`, separator `|---|---|`, each row `| [TestName](../tests/file_test.go) | one line rule |` — the link IS the file; **no File column** (it fails the `gotest` tests-index check). If the existing file has another header, rewrite it to this one. Rules in Spanish.

## 6.4 Documentation (Spanish, `veltylabs`)

`langc sync .`: zero pending for `es`, zero reuse warnings. `README.md` ≤150 lines, four sections in this order: `## ¿Por qué existe?` (hours, holidays and closures of an establishment; **Sin esto vs. Con esto**; first mentions linked), `## Antes de comenzar` (Go, `GOPRIVATE`, a `*orm.DB` with the tables of `migrate`, `Deps.IDs`), `## Guía paso a paso` (Step 1 create the module over `orm.New(mem.New())` — a complete `main` importing `"webtyp.com/fmt"` and printing `mod.ModelName()` → `business_calendar`; Step 2 add a holiday with `AddHoliday` and ask `GetDayDetail`, linking [tests/holidays_test.go](tests/holidays_test.go); Step 3 open the demo, linking [seed/demo.go](seed/demo.go)), `## Mapa de documentación` (table `Orden | Documento | Qué pregunta responde y cuándo leerlo`). Include the operations table of the mapping in `PLAN.md`, a permissions table with the resources `business_hours`, `holiday`, `closure`, `day_detail`, and the note that the event `business.calendar.changed` keeps being published on every write. `docs/ARCHITECTURE.md`: four sections (`## 1. Visión general en un minuto`, `## 2. Piezas y estructura`, `## 3. Flujos paso a paso`, `## 4. Contratos y límites de seguridad`): why a rule and a kept method share `normalizeHours`, `checkNewHoliday` and `checkNewClosure`, why the rules publish before the commit. Translate to Spanish every README, `docs/*`, diagram and code comment still in English; exported symbols without a usage example get one. Permanent docs never link `docs/PLAN.md` or its stage files.

Run `gotest` and `langc sync .` one last time, read their output, check the **Delivery check** of `PLAN.md`, commit, push and make sure the pull request is open.
