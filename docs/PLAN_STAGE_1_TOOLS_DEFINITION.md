← [Index](PLAN.md) | Next → [Stage 2](PLAN_STAGE_2_MODULE_HELPERS.md)

# Stage 1 — tools and `definition.go`

## 1.1 Tools (run first, exactly)

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
go install webtyp.com/definitionc/cmd/definitionc@v0.1.42
export PATH="$PATH:$(go env GOPATH)/bin"
go get webtyp.com/model@v0.2.19 webtyp.com/crud@v0.0.12 webtyp.com/orm@v0.12.33 webtyp.com/storage@v0.2.1 webtyp.com/ddl@v0.0.49 webtyp.com/router@v0.4.22 webtyp.com/definitionc@v0.1.42 webtyp.com/view@v0.6.74 webtyp.com/appkit@v0.2.35
go mod tidy
```

`GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`, `GOFLAGS=-mod=mod`; remove every `replace` from `go.mod`. **`definitionc` rewrites `go.mod` to older versions every time it runs: after EVERY run repeat the `go get` line and `go mod tidy`.** If a module cannot be fetched, stop and say so in the PR; never invent a hash.

Facts you need (asked before; do not ask again):

- Arguments of an operation: `crud.CreateArgs{Records []model.Model}`, `crud.UpdateArgs{IDs []string, Fields []string, Record model.Model}`, `crud.DeleteArgs{IDs []string}`, `crud.ReadArgs{ID, Search, After string, Limit int64, Filter model.Model}`; never a bare entity pointer, a `*List` or a map. Kinds are flat (`model.Pointer`/`List`/`Map` do not exist).
- A rule's `tx` is a `crud.Tx`: `ReadOne(&rec, id)`, `ReadAll(newRow, crud.Query{Conditions, Order, After, Limit}, onRow)`, `Create`, `UpdateFields(&rec, id, cols)`, `Delete(&rec, id)`. Conditions are `storage.Eq/Neq/Gt/Gte/Lt/Lte/In/Like` (AND only). A read answers at most `crud.MaxPage` rows.
- **Update rules:** `crud` writes `next` after the rule with exactly the columns in `Fields` (the ones the caller sent). A column the rule derives and the caller did NOT send must be written by the rule itself with `tx.UpdateFields(&row, id, []string{...})`.
- `NotNull` on an `Int` or `Bool` rejects the zero value (`0`, `false`): that is why `day_of_week` (Sunday = 0) and `is_open` (false) must NOT be `NotNull` (their rules check them).

## 1.2 `definition.go` — exact edits (keep every constant, error, `CalendarChangedPayload`, `ClosedReason` and the hand-written `DayDetail` struct as they are)

1. **Delete** the three argument models `GetDayLimitsArgsModel`, `ListHolidaysArgsModel`, `ListClosuresArgsModel`: no exported method uses them.
2. **`BusinessHoursModel`** (`Name: "business_hours"`, a table without `tenant_id`): add `Allow: model.Create | model.Read | model.Update` and `Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.HaveNoBusinessRule, Update: model.AddMyBusinessRule}`. Field edits: `id` gets `OmitEmpty: true`; **remove `NotNull: true` from `day_of_week` and from `is_open`** (keep it on `open_min`, `close_min`); `updated_at` gets `ServerOnly: true`.
3. **`HolidayModel`** (`Name: "holiday"`) and **`ClosureModel`** (`Name: "closure"`): add `Allow: model.Create | model.Read | model.Delete` and `Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.HaveNoBusinessRule, Delete: model.AddMyBusinessRule}`; `id` gets `OmitEmpty: true`; `updated_at` gets `ServerOnly: true`. (Delete has a rule because it must publish the calendar event.)
4. **New read model**, written exactly. Its Go type is `DayStatus` (its variable name without `Model`) because the name `DayDetail` is already the hand-written struct that `GetDayDetail` returns:

```go
// DayStatusModel answers whether the establishment opens on a date and why not: one row. The caller sends the date
// as the filter; the other fields are the answer.
var DayStatusModel = model.Definition{
	Name:  "day_detail",
	Wire:  true,
	Allow: model.Read,
	Rules: model.Rules{Read: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "date", Type: model.Int(), OmitEmpty: true},
		{Name: "open", Type: model.Bool(), OmitEmpty: true},
		{Name: "open_min", Type: model.Int(), OmitEmpty: true},
		{Name: "close_min", Type: model.Int(), OmitEmpty: true},
		{Name: "closed_by", Type: model.Text(), OmitEmpty: true},
	},
}
```
5. **`Definition`** (new), after the models:

```go
var Definition = model.ModuleDefinition{
	Name: "business_calendar",
	Entities: []model.Definition{BusinessHoursModel, HolidayModel, ClosureModel, DayStatusModel},
}
```
6. **Two error constants to add** next to the others (exact text):

```go
	ErrFieldNotEditable model.Invalid = "business_calendar: that field cannot be changed"
	ErrDateRequired     model.Invalid = "business_calendar: date is required"
```
7. **Delete from `definition.go`** the line `const ModelName = "business_calendar"` and its comment (it moves to `module.go`'s deletion in Stage 2: `definitionc` writes `const ModelName` itself). Keep it until Stage 3 if the package must compile in between.

`go vet ./...` still fails (`ops.go` and the old methods reference removed types): expected until Stage 2. Commit and push.
