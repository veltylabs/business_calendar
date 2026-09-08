---
PLAN: "feat: institutional calendar — business hours, holidays and closures as data"
EXECUTOR: jules
REVIEWER: none
---

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.
>
> **This is the GATE of the agenda domain work.** `appointment_booking` cannot
> land its validation until this module publishes. Orchestrator:
> [webtyp/docs/AGENDA_DOMAIN_MASTER_PLAN.md](https://github.com/webtyp/webtyp/blob/main/docs/AGENDA_DOMAIN_MASTER_PLAN.md).
>
> Read `AGENTS.md` in this repo root **before writing any code** — it is the
> module contract, copied verbatim from `veltylabs/modules/AGENTS.md`.

# Plan — `business_calendar`

## 1. What this module owns

The **institutional calendar**: when the establishment is open, and when it is
closed regardless of any professional's schedule.

It owns three things:

1. **Business hours** — the weekly opening window, one entry per day of week.
2. **Holidays** — dated closures with a legal/administrative origin, editable at
   runtime (a new holiday appears by law; nobody recompiles).
3. **Closures** — dated closures with a local origin (maintenance, an event).

It does **not** own any professional's agenda. `appointment_booking` owns that
and consumes this module as a read-only upstream bound.

This repo was scaffolded with `gonew` at `v0.0.1` and currently holds only the
stub `business_calendar.go`, `go.mod`, `LICENSE`, `README.md` and `AGENTS.md`.

## 2. Why a new module instead of growing `business_hours`

`github.com/veltylabs/business_hours` has been **archived by the owner**
(2026-09-08); its working tree remains at
`~/Dev/Project/veltylabs/modules/business_hours` as the source for the content
move. Five verified reasons it is replaced rather than extended:

1. **Zero consumers.** Nothing imported it outside its own tests.
   `work_schedule` only mentioned it in a comment, having duplicated its
   `dayNames`.
2. **The name named the shape, not the concern.** "business_hours" describes
   seven weekly rows. The concern is the institutional calendar — hours **plus**
   holidays **plus** closures. A holiday is not an "hour".
3. **It was read-only and could not grow.** `NewView` had no
   `WithSaveOp`/`WithDeleteOp`; the only op was `get_business_hours`. Holidays
   need full CRUD; there was no write path to preserve.
4. **Its time encoding contradicted `appointment_booking`.** It stored
   `open_time`/`close_time` as TEXT `"HH:MM"`; the agenda domain uses **integer
   minutes from midnight**. Two encodings of one quantity, at the boundary that
   must compare them. **This module fixes the encoding to minutes int.**
5. **It hardcoded Spanish** — `dayNames = [7]string{"Domingo",…}` in `view.go`.
   A library never fixes a human language.

## 3. Design gate

This module is entirely new public API. Per skill **api-design**:

### 3.1 Prior art

- **Google Calendar / Outlook working hours + holiday calendars.** Working hours
  are a weekly recurring window; holidays are a *separate subscribed calendar*
  of dated entries. The split — recurring window vs dated closure — is the same
  one taken here.
- **Cal.com / Calendly.** Availability is weekly + date overrides; neither has an
  institutional layer above the individual, which is exactly the gap this module
  fills for a clinic where the building itself has hours.
- **Epic Cadence / Cerner.** Department-level "operating hours" and a holiday
  master file sit **above** provider templates, and provider templates are
  validated against them. That is the hierarchy adopted here.

Why this ecosystem differs: none of the three exposes the institutional layer as
a standalone, replaceable module. Here it must be one, because
`appointment_booking` has to consume it through a typed contract without
depending on a clinic-specific implementation.

### 3.2 The novice-name test

- `BusinessHours` — *"the hours the business is open."*
- `Holiday` — *"a day the business is closed by law."*
- `Closure` — *"a day the business decided to close."*
- `GetDayBounds(date)` — *"give me the open/close bounds for that date."* Reads
  as the question the caller actually has.
- `IsOpen(date)` — *"is the business open that day?"*

### 3.3 Complexity ledger

| Row | Δ |
|---|---|
| Concepts the developer must learn | **+3** (`BusinessHours`, `Holiday`, `Closure`) — genuinely new capability: none existed |
| Files they must touch to add a holiday | **−1** — was "edit Go, recompile, redeploy"; becomes a row |
| Lines at the call site | **+1** (`bounds, err := cal.GetDayBounds(d)`) |
| Ways to do the same thing | **−1** — `business_hours` is archived in the same change, not kept |
| Exported surface | **+3 records, +6 ops, +2 reader methods** |

### 3.4 Where it belongs

One concern: *when is the establishment available*. It is not a second concern
inside `appointment_booking`, which owns *when a professional is available* —
two different owners, two different editors, two different permissions.

### 3.5 What it deletes

`github.com/veltylabs/business_hours` in its entirety: its module path, its
`"HH:MM"` text encoding, its hardcoded Spanish day names, and the compiled-in
`holidaysCL2026()` function in `app-demo/config/holidays.go` (deleted by the
`app-demo` plan, which consumes this module instead).

## 4. Use cases this plan must satisfy

From the master plan §4. **Every one gets a test.**

- **CU-01** — the administrator defines business hours per day of week.
- **CU-03** — widening the hours is visible to consumers with no recompile.
- **CU-04** — a new holiday is registered at runtime; that day stops being
  bookable.
- **CU-05** — a holiday is removed or moved; the affected days flip correctly.
- **CU-06** — a one-off closure that is not a national holiday, distinguishable
  by origin.
- **CU-07** — the bounds this module reports are what `appointment_booking`
  rejects against (this module supplies the answer; the neighbour enforces).

## 5. Stage 1 — `model.go`

No build tag. `model.Definition` literals only; `ormc` generates `model_orm.go`.

```go
package businesscalendar

// BusinessHoursModel: one row per day of week. day_of_week is unique — this
// module never writes more than seven rows.
//
// Times are MINUTES FROM MIDNIGHT (0..1439), the same encoding
// appointment_booking uses for work blocks. The archived business_hours module
// stored "HH:MM" text and forced a conversion at the one boundary that has to
// compare the two; the encoding is fixed here instead.
var BusinessHoursModel = model.Definition{
	Name: "business_hours",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "day_of_week", Type: model.Int(), NotNull: true, DB: &model.FieldDB{Unique: true}},
		{Name: "open_min", Type: model.Int(), NotNull: true},
		{Name: "close_min", Type: model.Int(), NotNull: true},
		{Name: "is_open", Type: model.Bool(), NotNull: true},
		{Name: "notes", Type: model.Text()},
		{Name: "updated_at", Type: model.Int(), NotNull: true},
	},
}

// HolidayModel: a dated closure with a legal/administrative origin. Editable at
// runtime — a holiday that appears by law must never require a recompile.
var HolidayModel = model.Definition{
	Name: "holiday",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "specific_date", Type: model.Int(), NotNull: true}, // midnight UTC, seconds
		{Name: "name", Type: model.Text(), NotNull: true},
		{Name: "notes", Type: model.Text()},
		{Name: "updated_at", Type: model.Int(), NotNull: true},
	},
}

// ClosureModel: a dated closure the establishment itself decided (maintenance,
// an event). Separate from Holiday because the ORIGIN is what the UI shows and
// what an administrator filters by — collapsing both into one table with a
// "type" column would make the origin a magic string.
var ClosureModel = model.Definition{
	Name: "closure",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "specific_date", Type: model.Int(), NotNull: true},
		{Name: "reason", Type: model.Text(), NotNull: true},
		{Name: "updated_at", Type: model.Int(), NotNull: true},
	},
}
```

Domain errors and event topics, in the same file:

```go
var (
	ErrNotFound       = fmt.Err("business_calendar: record not found")
	ErrInvalidDay     = fmt.Err("business_calendar: day_of_week must be 0..6")
	ErrInvalidMinutes = fmt.Err("business_calendar: minutes must be 0..1439 and open_min < close_min")
	ErrDuplicateDate  = fmt.Err("business_calendar: a record already exists for that date")
)

// EventCalendarChanged fires on every write. appointment_booking subscribes so
// a professional's editor reflects widened hours without a recompile (CU-03),
// a new holiday closes the day for everyone (CU-04), and reservations already
// booked on a day that just became closed are recomputed into conflict
// (CU-25…CU-29).
const EventCalendarChanged = "business.calendar.changed"

// CalendarChangedPayload says WHAT changed, not merely that something did.
// A subscriber that only learns "the calendar changed" has to recompute every
// reservation of every professional; with the range it recomputes a window.
//
// Implements model.Encodable, which events.Event.Payload requires.
type CalendarChangedPayload struct {
	// FromDate/ToDate bound the affected dates (midnight UTC seconds).
	// For a holiday or closure both equal that single date. For a business
	// hours change they are 0/0 — a weekly rule has no bounded range, and the
	// subscriber must recompute its whole horizon.
	FromDate int64
	ToDate   int64
	// Kind names what moved, so a subscriber can skip work it does not care
	// about: "BUSINESS_HOURS" | "HOLIDAY" | "CLOSURE".
	Kind string
	// Closed reports the direction. true when the change CLOSES time (a
	// holiday added, hours narrowed) — the only direction that can invalidate
	// an existing reservation. false when it opens time, which never does.
	Closed bool
}
```

`Closed` is what makes CU-29 cheap: a subscriber recomputing conflicts can
return immediately when time was only opened, because opening time cannot put a
reservation outside anything.

Use `fmt.Err`, never stdlib `errors`. Never a bare string literal in logic.

## 6. Stage 2 — `module.go`

`Module` struct, `New(db *orm.DB, deps Deps) (*Module, error)`, and the service
methods. Copy the `Deps`/`New` shape from the archived `business_hours/mcp.go`
working tree — it is already correct: `Deps{IDs model.IDGenerator; Publisher
events.Publisher}`, `IDs` required, `Publisher` optional, and the
`ddl.Compiler` type assertion so `storage/mem` works without DDL.

Service methods:

```go
func (m *Module) UpsertBusinessHours(h BusinessHours) error
func (m *Module) ListBusinessHours() ([]BusinessHours, error)

func (m *Module) AddHoliday(h Holiday) error
func (m *Module) RemoveHoliday(id string) error
func (m *Module) ListHolidays(from, to int64) ([]Holiday, error)

func (m *Module) AddClosure(c Closure) error
func (m *Module) RemoveClosure(id string) error
func (m *Module) ListClosures(from, to int64) ([]Closure, error)
```

Every write validates (`ErrInvalidDay`, `ErrInvalidMinutes`, `ErrDuplicateDate`)
and, on success, publishes `EventCalendarChanged` with a fully populated
`CalendarChangedPayload` when `m.pub != nil`.

Getting `Closed` right per method — this is the field a subscriber trusts:

| Method | `Kind` | `Closed` |
|---|---|---|
| `AddHoliday` | `HOLIDAY` | `true` |
| `RemoveHoliday` | `HOLIDAY` | `false` |
| `AddClosure` | `CLOSURE` | `true` |
| `RemoveClosure` | `CLOSURE` | `false` |
| `UpsertBusinessHours` | `BUSINESS_HOURS` | `true` when the new window is **narrower** than the stored one, or `is_open` goes true→false; otherwise `false` |

`UpsertBusinessHours` is the one that needs care: it must **read the existing
row before writing** to decide the direction. Widening (08:00→07:00 open, or
20:00→22:00 close) is `false`; narrowing either edge, or closing the day, is
`true`. A day that both opens earlier and closes earlier is **narrowing** —
`Closed: true`, because some previously-bookable time disappeared.

When in doubt, publish `Closed: true`: a subscriber that recomputes
unnecessarily is slow, one that skips a real closure leaves a patient with an
appointment on a day the clinic is shut.

**Anti-footgun:** `AddHoliday` must reject a second row for the same date
(`ErrDuplicateDate`) — a duplicated holiday would double-close a day and make
`RemoveHoliday` look broken. `day_of_week` uniqueness is enforced by the DB;
date uniqueness on holidays/closures is **not**, so check it in the service.

## 7. Stage 3 — `interfaces.go`, the contract neighbours consume

> ### ⚠️ AMENDMENT (2026-09-08) — this stage is already implemented and must change
>
> `interfaces.go` shipped with a **locally-defined** `DayBounds` and a `Reader`
> whose method returns it. That makes any consumer import this module to name
> the type — which is exactly the coupling
> `veltylabs/appointment_booking` must not have (its `go.mod` has zero veltylabs
> dependencies and must keep them).
>
> **The change:** the value that crosses becomes **`tinytime.DayBounds`** from
> `webtyp.com/time` — a neutral leaf **this module already imports**
> (`interfaces.go:6`, `module.go:9`) and so does `appointment_booking`
> (`service.go:9`). Neither module then imports the other, and the neighbour's
> `BoundsReader` is satisfied **structurally**, with no adapter anywhere.
>
> Concretely:
>
> - `GetDayBounds(date int64) (tinytime.DayBounds, error)` — the port's value is
>   the neutral `{Open, OpenMin, CloseMin}`.
> - **`ClosedBy` does not cross.** It is this module's own vocabulary and stays
>   here, exposed through a separate richer method for this module's UI:
>   `GetDayDetail(date int64) (DayDetail, error)`, where
>   `DayDetail{Bounds tinytime.DayBounds; ClosedBy ClosedReason}` is what
>   `get_day_bounds` responds with and what implements `model.Encodable`.
> - `Reader` stays declared here as a **convenience alias for this module's own
>   consumers**, but it is no longer the contract `appointment_booking` names —
>   that one lives in `appointment_booking`. Two interfaces with the same method
>   set is not duplication: Go satisfies both structurally, and each package
>   names the dependency it actually has.
> - The resolution order, the precedence rationale and the `GetDayBounds`
>   implementation below are **unchanged** — only the return type and the place
>   `ClosedBy` is reported move.
>
> Depends on `webtyp.com/time` publishing `DayBounds`
> ([time/docs/PLAN.md](https://github.com/webtyp/time/blob/main/docs/PLAN.md)).
> Rationale in full: `webtyp/docs/AGENDA_DOMAIN_MASTER_PLAN.md` §A12.

### Original specification (superseded above where they disagree)

This is the load-bearing file. `appointment_booking` must depend on a **typed
port**, never on this module's concrete type.

```go
package businesscalendar

// DayBounds is the open window of the establishment for one concrete date.
// Minutes from midnight. Open == false means closed, and the minute fields
// carry no meaning.
type DayBounds struct {
	Open     bool
	OpenMin  int
	CloseMin int
	// ClosedBy names why a closed day is closed, for the UI to say so.
	// "" when Open is true.
	ClosedBy ClosedReason
}

type ClosedReason string

const (
	ClosedWeekly  ClosedReason = "WEEKLY"  // business_hours says closed that weekday
	ClosedHoliday ClosedReason = "HOLIDAY" // a Holiday row falls on that date
	ClosedLocal   ClosedReason = "CLOSURE" // a Closure row falls on that date
)

// Reader is the READ-ONLY port a neighbour module depends on. It is declared
// HERE, by the owner of the concern — a consumer that declared its own local
// interface would fork this contract and the copy could never be reused.
type Reader interface {
	// GetDayBounds answers "is the establishment open on this date, and
	// between which minutes". date is midnight UTC in seconds.
	GetDayBounds(date int64) (DayBounds, error)
}

var _ Reader = (*Module)(nil)
```

`GetDayBounds` resolution order, in this exact precedence:

1. A `Closure` on that date → `{Open: false, ClosedBy: ClosedLocal}`.
2. A `Holiday` on that date → `{Open: false, ClosedBy: ClosedHoliday}`.
3. `BusinessHours` for that weekday with `is_open == false` →
   `{Open: false, ClosedBy: ClosedWeekly}`.
4. Otherwise `{Open: true, OpenMin, CloseMin}`.

A local closure outranks a holiday because it is the more specific, more recent
decision, and because a holiday the establishment chose to work through must not
be reopened by removing the closure.

**No `map`** anywhere in resolution — these sets are tiny (7 weekly rows, a
handful of dated rows per range); linear scan over a slice, per `AGENTS.md`.

## 8. Stage 4 — `ops.go`

```go
const (
	OpListBusinessHours   = "list_business_hours"
	OpUpsertBusinessHours = "upsert_business_hours"
	OpListHolidays        = "list_holidays"
	OpAddHoliday          = "add_holiday"
	OpRemoveHoliday       = "remove_holiday"
	OpListClosures        = "list_closures"
	OpAddClosure          = "add_closure"
	OpRemoveClosure       = "remove_closure"
	OpGetDayBounds        = "get_day_bounds"
)
```

`MountOperations` registers each with `Requires(resource, action)`. Follow the
precedent set in `appointment_booking/ops.go`: an **upsert declares
`model.Create|model.Update`**, because it really can do both and declaring only
`Update` would let an update-only principal create rows — a closed-by-default
violation. Reads are `model.Read`; removes are `model.Delete`.

Resource names: `"business_hours"` for the hours ops, `"holiday"` for holiday
ops, `"closure"` for closure ops. Three resources, because an administrator who
may add a holiday is not necessarily the one who may rewrite opening hours.

## 9. Stage 5 — `view.go`

Three presenters, so a `crudview` can mount each without custom config:

```go
func NewBusinessHoursView(caller router.Caller) view.Presenter
func NewHolidaysView(caller router.Caller) view.Presenter
func NewClosuresView(caller router.Caller) view.Presenter
```

Hours are read/update (`WithSaveOp`); holidays and closures are full CRUD
(`WithSaveOp` + `WithDeleteOp`) — that is the whole point of CU-04.

**Day names must NOT be hardcoded.** The archived module had
`dayNames = [7]string{"Domingo",…}`; that is the defect §2.5 names. Render the
canonical English name from `webtyp.com/date` (`date.WeekdayName(i)`) through
`lang.Translate`, and let the consuming app register the dictionary. Titles go
through `lang.Translate` too — `view.WithTitle(lang.Translate("Business hours").String())`.

Translation keys this module introduces, to be listed in `README.md`:
`Business hours`, `Holidays`, `Closures`, `Open`, `Closed`, plus the seven
weekday names from `date.WeekdayName`.

## 10. Stage 6 — tests (`tests/`)

Consumer-shaped, through the real stack, `storage/mem` as the only fake. One
test per use case, named for it:

| Test | CU |
|---|---|
| `TestAdminDefinesWeeklyBusinessHours` | CU-01 |
| `TestWideningHoursIsVisibleToReaderImmediately` | CU-03 |
| `TestNewHolidayClosesTheDayWithoutRecompile` | CU-04 |
| `TestRemovingHolidayReopensTheDay` | CU-05 |
| `TestMovingHolidayFlipsBothDays` | CU-05 |
| `TestLocalClosureIsDistinguishableFromHoliday` | CU-06 |
| `TestGetDayBoundsPrecedenceClosureOverHolidayOverWeekly` | CU-06 |
| `TestDuplicateHolidayOnSameDateIsRejected` | — (anti-footgun §6) |
| `TestInvalidMinutesRejected` | — |

Plus a `tests/conformance_test.go` mirroring the archived module's: the module
satisfies `router.OperationModule`, and `*Module` satisfies `Reader`.

`CU-03` is the one that is easy to write wrongly: it must assert that a
`Reader.GetDayBounds` call made **after** an `UpsertBusinessHours` sees the new
bounds, with no restart and no cache invalidation step — that is what "without
recompiling" means at this layer.

## 11. Stage 7 — docs

- `README.md` — replace the `gonew` stub. Purpose, the three records, the ops
  table, the `Reader` port, translation keys, and an explicit note that this
  module replaces the archived `business_hours` and why the time encoding is
  minutes int.
- `docs/ARCHITECTURE.md` — domain scope, the `GetDayBounds` precedence rule, and
  why holidays and closures are separate tables.
- `docs/diagrams/database.md` — the three tables.

## 12. Constraints — read before writing code

- **`AGENTS.md` in this repo root is the contract.** Read it first.
- **No stdlib**: `webtyp.com/fmt`, never `errors`/`strconv`/`strings`.
- **No `map`** anywhere — linear scan over slices. These sets are tiny.
- **No build tags.** `model.go`, `module.go`, `ops.go`, `view.go` import only
  isomorphic packages (`model`, `orm`, `ddl`, `router`, `view`, `events`) and
  need no `!wasm`. If you reach for one, you imported a concrete driver by
  mistake.
- **`model_orm.go` is generated by `ormc` — never hand-edit it.**
- **No hardcoded human language** in this module.
- **No `TODO`, no commented-out code, no deprecated path.** Before closing:
  `grep -rn "TODO\|FIXME\|Deprecated" --include='*.go' .` — every hit must
  predate this change.
- `gotest`, never `go test`.

## 13. Acceptance criteria

| # | Check | Expected |
|---|-------|----------|
| 1 | `gotest ./...` | green, every test in §10 present |
| 2 | `grep -rn "HH:MM\|open_time\|close_time" --include='*.go' .` | **empty** — minutes int only |
| 3 | `grep -rn "Domingo\|Lunes\|Martes" --include='*.go' .` | **empty** — no hardcoded Spanish |
| 4 | `grep -rn "map\[" --include='*.go' .` | **empty** |
| 5 | `grep -rn "\"errors\"\|\"strconv\"\|\"strings\"" --include='*.go' .` | **empty** |
| 6 | `grep -rn "go:build" --include='*.go' .` | **empty** unless justified inline |
| 7 | `grep -n "var _ Reader" interfaces.go` | present |
| 8 | `grep -n "var _ router.OperationModule" ops.go` | present |
| 9 | `GOOS=js GOARCH=wasm go build ./...` | compiles |
| 10 | `grep -rn "TODO\|FIXME\|Deprecated" --include='*.go' .` | no hit introduced here |

## 14. Stages

| # | Stage | Files | Done when |
|---|-------|-------|-----------|
| 1 | Data model | `model.go` | three Definitions, errors, event topic |
| 2 | Service | `module.go` (+ `model_orm.go` via `ormc`) | eight methods, validation, publish |
| 3 | The port | `interfaces.go` | `Reader`, `DayBounds`, precedence, `var _ Reader` |
| 4 | Ops | `ops.go` | nine ops, correct `Requires` bitmasks |
| 5 | Views | `view.go` | three presenters, zero hardcoded language |
| 6 | Tests | `tests/` | every CU in §4 green |
| 7 | Docs | `README.md`, `docs/` | ops table, port, translation keys |

## 15. What this plan does NOT do

- It does not touch `appointment_booking`. The consumption of `Reader` and the
  rejection of out-of-bounds blocks (CU-07's enforcement half) belong to that
  repo's plan.
- It does not seed Chilean holidays. `holidaysCL2026()` lives in
  `app-demo/config/holidays.go` and is deleted there, by the `app-demo` plan,
  which seeds these rows through the real ops instead.
- It does not add per-professional anything. That is `appointment_booking`.
