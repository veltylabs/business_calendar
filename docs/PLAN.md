---
PLAN: "refactor!: business_calendar on the crud-by-declaration framework — Allow and Rules in definition.go, persistence by crud, only domain rules in rule files"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 3725613087550276499
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work. Commit after every stage.
> Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.
> Never edit `definition_gen.go`: a build error there means `definition.go` or a rule file is wrong; fix that and run `definitionc` again.
> This plan is dispatched in one wave with the other 11 modules without external calls (D53), after `patient_directory` (the reference) was executed and approved by the owner.


# Plan: `business_calendar` on the crud-by-declaration framework

Today the module is still on `router.OperationRegistry` + `view.MountSave`/`MountDelete` (`ops.go`). Its three tables have no `tenant_id`: one calendar per installation.

Reference implementation of the same migration (read it first, copy its shape): `patient_directory` — `definition.go` (`Definition`, `Allow`, `Rules`), `rule_patient_create.go`/`rule_patient_update.go`/`rule_patient_search.go` (the rules), `store.go`, `readers.go` (signatures kept, bodies on `m.store`).

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
go install webtyp.com/definitionc/cmd/definitionc@v0.1.41
go get webtyp.com/model@v0.2.18 webtyp.com/crud@v0.0.11 webtyp.com/orm@v0.12.30 webtyp.com/storage@v0.2.1 webtyp.com/ddl@v0.0.49 webtyp.com/router@v0.4.21 webtyp.com/definitionc@v0.1.41 webtyp.com/view@v0.6.72 webtyp.com/appkit@v0.2.33 webtyp.com/rpc@v0.2.21
```
Remove every `replace` from `go.mod`; `go mod tidy`. Before any change check, with `go doc`, that these exist: `model.ModuleDefinition`, `model.Rules` (`model.AddMyBusinessRule`, `model.HaveNoBusinessRule`), `crud.CurrentUser`, `crud.RuleAnswer`, `crud.Topic`, `crud.Loopback`, and that `definitionc` writes a `definition_gen.go` containing `func (m *Module) MountOperations(`. If one is missing, stop and say so in the PR.
Prerequisite of the whole wave: `patient_directory` was executed, reviewed and approved by the owner; it is the reference and the 17 other modules are dispatched together after it.

## Lessons of the reference module (`patient_directory`, executed locally)

- Needs `crud` ≥ v0.0.11 (a filter has no required field; v0.0.10 rejected every read of a table with a rule). `definitionc` ≥ v0.1.41 generates `Search(a, filter *X, text, m)`; a text of `""` is normal (the rule runs on every read).
- Running `definitionc` rewrites `go.mod` to older versions: run the `go get` list of «Tools» again after it, then `go mod tidy`; raise stale indirect dependencies (`rbac`, `mcp`, `appkit`) to the published ones.
- In tests, events reach the publisher given to `crud.LoopbackActingAs(pub, "u1", mod)`, not `Deps.Publisher`; a caller double takes `router.Operation`; a `ddl.Execer` double returns `(storage.Result, error)`; a rule that answers `0` fails with an internal error (assert the refusal and that nothing was stored).
- `langc sync .` can warn that other languages are untranslated: that is the repo's `lang.json`, not this plan.

## The rule of this plan (decisions D28–D50 of the architecture master)

The framework persists; the module declares and writes only domain rules.

| The module no longer writes | Who does it now |
|---|---|
| id, `tenant_id`, `created_at`/`updated_at`, the transaction loop, create/update/delete bodies, «not found», batch atomicity, idempotent retry by id, paging by `After`, `tenant_id` in every query, schema checks repeated from the model | `crud` + `orm.TenantStore` / `orm.GlobalStore` |
| an operation per verb (`list_*`, `save_*`, `upsert_*`, `delete_*`), its handler, `ops.go`, the permission resources and the event topics | `Allow` + `Rules` in `definition.go`; `definitionc` writes `MountOperations`; `crud` names the operation (`router.Operation{Module, Entity, Action}`; on the wire `GET\|POST\|PATCH\|DELETE /api/<module>/<entity>`, in MCP `<module>.<entity>.read\|create\|update\|delete`) and publishes `<module>.<entity>.created\|updated\|deleted` with a `*crud.Change` after the commit |
| an action with its own permission (approve, settle, receive stock…) | a **record created** in its own table, or a **read model** with its own name (D18) |

A module writes: `definition.go` (`var Definition = model.ModuleDefinition{…}` and, per table, `Allow` and `Rules`), one `rule_<entity>_<verb>.go` for every action declared `model.AddMyBusinessRule` (created once by `definitionc`; the skeleton does not compile until completed) and `store.go` (`func (m *Module) Store<Entity>() crud.Store`, one line per table). Every allowed action that has no rule below is declared `model.HaveNoBusinessRule`: `crud` alone checks it. Read the reference first: `patient_directory` (`definition.go`, `rule_patient_*.go`, `store.go`).

| Rule | Form (the receiver is the entity) |
|---|---|
| Create | `func (p *X) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error)` |
| Update | `func (next *X) Update(a crud.CurrentUser, tx crud.Tx, old *X, fields []string, m *Module) (crud.RuleAnswer, error)` (`next` carries only the sent `fields`) |
| Delete | `func (p *X) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error)` (`p` is the stored row) |
| Search (Read of a table) | `func (p *X) Search(a crud.CurrentUser, filter *X, text string, m *Module) ([]storage.Condition, error)` — AND conditions only; it runs on every read, `text` may be `""`, and `filter` is a record of the entity with only the fields the caller sent (D52: the entity is its own filter) |
| Read (model without a table) | `func (p *X) Read(a crud.CurrentUser, filter *X, args crud.ReadArgs, m *Module) (model.ModelSlice, error)` (`filter` carries the fields the caller sent, D52) |

A rule ends with `return crud.ApplyAndPublish, nil` (saves and announces) or `crud.ApplyWithoutPublishing` (saves without announcing); a rejection is `return 0, err`. It queries with `tx`, never with `m.db`. To write another row it calls `tx.Create`/`tx.UpdateFields`. The old `Committed` hook does not exist: `crud` publishes.

## Tables, stores and rules

An action with a rule below is declared `model.AddMyBusinessRule` in the table's `Rules`; every other allowed action is `model.HaveNoBusinessRule`. A `Search` below is the rule of `Read`.

| Table (`<X>Model`) | Store | `Allow` | Rules (`rule_<entity>_<verb>.go`) — where their rules come from today |
|---|---|---|---|
| `BusinessHours` | `orm.GlobalStore` | `model.Create \| model.Read \| model.Update` | `Create`, `Update`: the checks of `UpsertBusinessHours` (one row per `day_of_week`, open < close, HH:MM) — the uniqueness check reads with `tx.ReadAll` |
| `Holiday` | `orm.GlobalStore` | `model.Create \| model.Read \| model.Delete` | `Create`: the checks of `AddHoliday` before its insert |
| `Closure` | `orm.GlobalStore` | `model.Create \| model.Read \| model.Delete` | `Create`: the checks of `AddClosure` before its insert |

## Read models (D18 — no table, `Allow: model.Read` only, `Rules.Read: model.AddMyBusinessRule`)

| Model | Filter | Answered by |
|---|---|---|
| `DayDetail` (existing result of `GetDayDetail`; add `Allow: model.Read`, no `DB:` marks) | `DayLimitsFilter{date int64}` (renamed from `GetDayLimitsArgsModel`) | the `Read` rule of `DayDetail` (`rule_<model name>_read.go`) — the fields of the filter `DayLimitsFilter` become fields of this model and the rule reads them from its `filter` parameter (D52), body of `m.readDayLimits` → `GetDayDetail(time.Unix(f.Date))` |

A read model is a `model.Definition` **without** `DB:` marks, with `Allow: model.Read` and `Rules: model.Rules{Read: model.AddMyBusinessRule}`, listed in `Definition.Entities`; `definitionc` mounts it and creates its `rule_<model name>_read.go`. Its permission resource is its own `ModelName`: list it in the README permissions table (roles must be granted it).

## Operation mapping (every operation of today has exactly one row)

«After» is `router.Operation{Module, Entity, Action}` written `<entity>.<verb>`; the wire form is `crud`'s (REST `/api/<module>/<entity>` by method, MCP `<module>.<entity>.<verb>`).

| Today | After |
|---|---|
| `list_business_hours` | `business_hours.read` (MountBusinessHours) |
| `upsert_business_hours` | `business_hours.create` (new day) / `business_hours.update` (existing day) |
| `list_holidays` | `holiday.read` |
| `add_holiday` | `holiday.create` |
| `remove_holiday` | `holiday.delete` |
| `list_closures` | `closure.read` |
| `add_closure` | `closure.create` |
| `remove_closure` | `closure.delete` |
| `get_day_limits` | read model `DayDetail` (resource = its ModelName) |

## Stage 1 — `definition.go`

- Set `Allow` on each table exactly as in «Tables, stores and rules»; a table not listed there gets no `Allow` (internal).
- Mark `ServerOnly: true`: `tenant_id`, `created_at`, `updated_at`, and every field the server fills (actor/author fields).
- Delete every `*Args` / `*Result` definition that only carried a removed operation (keep `GetDayLimitsArgsModel` renamed to `DayLimitsFilterModel`, name `day_limits_filter`, `Wire: true`)
  (`grep -n "ArgsModel\|ResultModel" model.go` → only the ones this plan keeps).
- The `Describe*` texts (`DescListHolidays`…) are deleted: the MCP description now comes from the model and the action.
- **Keep every `*Args` model that an exported reader's signature still uses** (`grep -n "^func (m \*Module) [A-Z][A-Za-z]*([^)]*Args" *.go` lists them): mark each `Wire: true`; delete only the `*Args` no remaining signature uses.
- D52 (the entity is its own filter): a filter model (`*Filter`, `*FilterModel`, an `*ArgsModel` that only carried filter values) is not kept: its fields become fields of the read model or table (`OmitEmpty`), and the rule reads them from its receiver; delete the filter type and its `Wire`/`NewFilter` plumbing.
- Add `var Definition = model.ModuleDefinition{Name: "business_calendar", Entities: []model.Definition{…every table and read model of this plan…}}` and, on every table and read model, `Rules` as «Tables, stores and rules» says (`AddMyBusinessRule` where a rule is named, `HaveNoBusinessRule` for the other allowed actions; a read model: `Read: model.AddMyBusinessRule`). Delete the hand topics (`Topic*`) and `Err*`/resources no code uses any more.

## Stage 2 — `module.go`

`Deps` already has `IDs`; make it required (`depErr("IDs")`).
`Module` gains `store *orm.Store`, built once in `New`: `m.store = orm.GlobalStore(db, deps.IDs)`.
Delete the hand-written persistence (create/update/upsert/patch/delete/list/get bodies that only move rows) once its rules moved to the rule files: `UpsertBusinessHours`, `AddHoliday`, `RemoveHoliday`, `AddClosure`, `RemoveClosure` bodies that write rows (their checks move to the rules); `businessHoursViewList` and every `op*` handler.

## Stage 3 — generate, then write the rules

1. Delete `ops.go`, `crud.go`, `queries.go`, `model_crud.go`, `model_orm.go` and `definition_crud.go` (those that exist). Run `definitionc`: it writes one `definition_gen.go` (the types, the schemas and, on `Module`, `ModelName()`, `Definition()`, `MountOperations()`) and the skeleton `rule_<entity>_<verb>.go` of every action declared `AddMyBusinessRule`.
2. Add `store.go`: for each table of «Tables, stores and rules» that has a Store, `// Store<Entity> is the store of <entity>.` `func (m *Module) Store<Entity>() crud.Store { return m.store }` (the module's global store for a table the plan lists with `orm.GlobalStore`).
3. Fill each skeleton with the body of the rule named in «Tables, stores and rules» (and «Read models»), in the forms of «The rule of this plan». An old rule `func(c crud.Caller, tx, rec) error` becomes the rule method with the same body, `c.UserID` becomes `a.UserID`, a nil error becomes `return crud.ApplyAndPublish, nil`, an error `e` becomes `return 0, e`.
4. Specific to this module: Every rule named above is a rule file. `checkBusinessHoursChange` merges `old` with only the fields in `fields` before checking (as `patient_directory.checkPatientChange`).
Delete every `Op*` constant, `res*`/`Resource*` variable, every `op*` handler, every `view.MountSave`/`view.MountDelete`/`reg.Operation` call and the `webtyp.com/router`, `webtyp.com/rpc` imports that only they used (`grep -rn "reg.Operation\|view.MountSave\|view.MountDelete\|router.Context\|rpc.WriteError" --include=*.go . | grep -v _test` → empty).

## Stage 4 — readers used by other modules

Signatures of every exported reader stay (other modules and apps call them). Bodies read through `m.store` (`ReadOne`, `ReadAll` with `crud.Query`), never `m.db` directly; a `tenantID` parameter different from the module's tenant answers «not found» (`found=false` / empty, `err=nil`). Keep `BankDayAfter`, `GetDayDetail`, `DayLimits`, `WeekdayLimits`, `ClosedDates`, `ListBusinessHours`, `ListHolidays`, `ListClosures` (their date ranges become `storage.Gte`/`Lte` conditions in `crud.Query`).

## Stage 5 — browser side (D15, D16)

- `view.go` (`//go:build browser`): every `view.NewCallerLister(caller, view.Ops{…}, newList)` takes
  `func() model.TableSlice { return &<Entity>List{} }` and `view.Ops` has no `Allow` (the screen reads `<Entity>List.Allow()`).
- `module/browser.go`, `module/server.go` (if present): `appkit.ModuleBrowser`/`appkit.BrowserDeps` → `view.ModuleBrowser`/`view.BrowserDeps`.
- Demo: `seed/demo.go` (`//go:build wasm`) returning `view.Demo{Label, Timezone: "America/Santiago", ActingAs: "u1",
  Modules: []crud.Module{mod}, Migrations: <the module's ddl.Step list, or nil if it has no migrations>,
  Screens: func(d view.BrowserDeps) []view.Screen { return ui.Screens(<what today's demo main passes>) }}`, built with
  `orm.New(mem.New())`, `New(db, Deps{…})` with the same fakes today's demo uses, and `seed.Load` (when the module has one).
  Delete the old demo: `web/browser/main.go` (`test ! -e web`).
- `grep -rn "webtyp.com/layout\|appkit.BrowserDeps\|appkit.ModuleBrowser\|router/loopback" .` → empty, `go.mod` included.

## Stage 6 — tests (`tests/`, through `crud.LoopbackActingAs(nil, "u1", mod)`; rows in `docs/TESTS.md`)

Delete tests of removed code. Operations are `router.Operation{Module: ModelName, Entity: "<entity>", Action: model.<Action>}`, never a string; fixtures follow `crud` v0.0.10 (`reg.AddStoredEntity`, `crud.CurrentUser`). Add `TestMount_MatchesDefinition` (`crud.Mount(reg, nil, nil, mod)` returns nil) and one test that an allowed write publishes a `*crud.Change` on `crud.Topic(ModelName, "<entity>", model.<Action>)`. Required:

| Test | Setup → call → expected |
|---|---|
| `TestBusinessHours_OneRowPerDay` | create two rows with the same `day_of_week` → second: field error on `day_of_week` |
| `TestBusinessHours_OpenBeforeClose` | create with open ≥ close → field error |
| `TestBusinessHours_UpdateOnlyListedFields` | update `["close"]` → `open` unchanged |
| `TestHoliday_NoUpdateOperation` | calling `holiday_u` → unknown operation |
| `TestDayDetail_ClosedOnHoliday` | holiday on a date → read model for that date says closed |
| `TestReaders_ClosedDates` | `ClosedDates` returns holidays and closures in range |
| `TestDemo_Valid` (`//go:build wasm`) | `seed.Demo()` → no error, `Validate()` nil, at least one screen |

## Stage 7 — translations and documentation (Spanish, veltylabs)

`langc sync .` (zero pending, zero reuse warnings). `README.md` (≤150 lines, 4 mandatory sections; «Sin/Con»: hand-written
operations vs `Allow` + rules) with the operations table of the mapping above. `docs/ARCHITECTURE.md`: rules vs persistence.
Translate to Spanish every README, `docs/*`, diagram and code comment still in English; exported symbols without a usage example get one.

## Delivery check

`gotest` passes; `langc sync .` clean; every grep of this plan empty. A tree that does not compile or whose tests did not run is a
failed delivery.

## Additions

A first session of this plan was rejected. Do not repeat its failures; each line below is a check the reviewer runs.

- **`definitionc`:** use `v0.1.42` everywhere the plan says `v0.1.41` (`go install webtyp.com/definitionc/cmd/definitionc@v0.1.42`, `go get webtyp.com/definitionc@v0.1.42`). A read model (`day_detail`) is `Allow: model.Read`, no `DB:` marks (or `Wire: true`), and its rule is `func (p *DayDetail) Read(a crud.CurrentUser, filter *DayDetail, args crud.ReadArgs, m *Module) (model.ModelSlice, error)`: the date the caller asks for arrives in `filter.Date` (declare `date` on the read model with `OmitEmpty`). A caller and a test pass it as `&crud.ReadArgs{Filter: &DayDetail{Date: d}}`.
- **Never edit `definition_gen.go`** (the first session hand-edited it). Change `definition.go` or a rule file and run `definitionc` again; after every run repeat the `go get` line of «Tools» and `go mod tidy`. Never write a hash into `go.sum` by hand: only `go mod tidy` writes it.
- **Delete the hand-written mutators** `UpsertBusinessHours`, `AddHoliday`, `RemoveHoliday`, `AddClosure`, `RemoveClosure` (nobody outside this repo calls them). Tests that need rows as fixtures create them through `crud.LoopbackActingAs` (`&crud.CreateArgs{Records: …}`) or through the `Import…` methods of `import.go` (add `ImportClosure` next to `ImportHoliday`, same shape as https://github.com/veltylabs/patient_directory/blob/main/import.go). Tests of a removed method are rewritten as `Test<Area>_<Rule>` through the caller, never skipped or left calling the removed method.
- **Stage 5 is not optional:** `seed/demo.go` (`//go:build wasm`) exactly like https://github.com/veltylabs/patient_directory/blob/main/seed/demo.go (`Demo() (view.Demo, error)`), and delete `web/browser/main.go`. `tests/module_test.go` uses the removed `appkit.Deps`/`Server.Build`: delete it and write `TestMount_MatchesDefinition` (`crud.Mount(nil, nil, nil, mod)` returns nil, see the reference `tests/crud_test.go`). `gotest` must end green on EVERY destination (the `browser` and `wasm` ones included); a red destination is a failed delivery.
- **Stage 6 tests are all required,** by name: `TestBusinessHours_OneRowPerDay`, `TestBusinessHours_OpenBeforeClose`, `TestBusinessHours_UpdateOnlyListedFields`, `TestHoliday_NoUpdateOperation`, `TestDayDetail_ClosedOnHoliday`, `TestReaders_ClosedDates`, `TestDemo_Valid` (`tests/demo_test.go`, `//go:build wasm`, copy the reference `tests/demo_test.go`), `TestMount_MatchesDefinition`, and one test that an allowed write publishes a `*crud.Change` on `crud.Topic(ModelName, "<entity>", model.<Action>)`. Every test needs a row in `docs/TESTS.md` as `| [TestName](../tests/file_test.go) | rule |` under `| Test | Rule |` (the link is the file; no File column) or `gotest` fails «tests index».
- **Do not commit scratch files** (no `*.py`, `*.sh`, `commit_message.txt`). `docs/PLAN.md` «Executor notes» must only state what is true: do not write that a file was restored if it was deleted.
- **Stage 7:** `README.md` with the four mandatory sections in Spanish (`¿Por qué existe?`, `Antes de comenzar`, `Guía paso a paso` with a complete compilable Go example importing `webtyp.com/fmt`, `Mapa de documentación`) and the «Sin/Con» operations table; `langc sync .` reports zero pending. Run `gotest` and `langc sync .` and read their output before opening the PR.
- **Do not stop to confirm.** Do all stages in order; when the delivery check passes, commit, push and open the PR.

## Additions (third attempt)

The second session stopped answering after Stage 5 and never pushed a branch, so its work was lost. Two rules for this one:

- **Push early.** After EVERY stage run `git push origin HEAD` (the branch your session created) so the work exists outside your machine. After Stage 3 open the pull request; keep pushing to it.
- **Never ask a question that is a yes/no on progress** («am I on the right track?», «shall I proceed?», «shall I push?»): the answer is always yes, proceed. Ask only when a compile error or a signature in this plan is wrong, and name the symbol.
