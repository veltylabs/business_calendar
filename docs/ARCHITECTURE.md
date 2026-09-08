# business_calendar Architecture

## 1. Domain scope

The **institutional calendar**: when the establishment is open, and when it is
closed regardless of any professional's schedule. One bounded concern — *when is
the establishment available* — owned by this module, consumed read-only by
`appointment_booking` (which owns *when a professional is available*: a
different owner, a different editor, a different permission).

It owns three entities:

- **BusinessHours** — one row per day of week (0=Sunday … 6=Saturday),
  `day_of_week` UNIQUE. Times are **minutes from midnight** (`0..1439`), the same
  encoding `appointment_booking` uses for work blocks.
- **Holiday** — a dated closure with a legal/administrative origin (`name`),
  editable at runtime (a new holiday appears by law; nobody recompiles).
- **Closure** — a dated closure the establishment itself decided (`reason`):
  maintenance, an event.

### Why holidays and closures are separate tables

The **origin** is what the UI shows and what an administrator filters by.
Collapsing both into one table with a `type` column would make the origin a
magic string; two tables make it a first-class, greppable `ClosedReason`. Both
are dated, but a local closure **outranks** a holiday in resolution (below) —
the more specific, more recent decision wins, and a holiday the establishment
chose to work through must not be reopened by removing the closure.

## 2. Not multi-tenant (signed-off decision)

Unlike most modules in this ecosystem, `business_calendar` has **no `tenant_id`
field** on any table: the institutional calendar is one global schedule per
composition-root app, not scoped per tenant. This is a deliberate, explicit
decision (per `AGENTS.md` "Domain-specific notes"); do not add `tenant_id` as a
casual change — that would be a domain change, not a harness-adoption change.

Because there is no tenant column, `UPDATE`/`DELETE` conditions are keyed on the
row `id` (and `day_of_week` for hours), which is safe for a single-tenant
singleton. The cross-tenant-write concern that `tenant_id` guards against does
not exist here.

## 3. Patterns applied

Coupled only to the published `webtyp.com/*` ports, never to concrete
infrastructure — see `AGENTS.md` (repo root) for the whitelist/blacklist:

- **`orm.DB` for storage** — backend-agnostic over whatever `storage.Conn` the
  composition root injects (`storage/mem` in this module's own tests).
- **`ddl`** for the module's own schema migration in `New()`, behind the
  `db.RawConn().(ddl.Compiler)` type assertion — a no-op against `storage/mem`.
- **`router.OperationModule`** (`ModelName()` + `MountOperations`) for transport
  — the module never sees a concrete server or `net/http`.
- **`model.IDGenerator`** for identity (`Deps.IDs`, required — the module never
  builds one).
- **`events.Publisher`** (`Deps.Publisher`, optional — nil disables silently)
  for `EventCalendarChanged` after every successful write.
- **`view.Presenter`** (`NewBusinessHoursView`/`NewHolidaysView`/
  `NewClosuresView(caller router.Caller)`) built with only `view`+`model`+`router`
  (+ `date` and `fmt/lang` for translation) — the app chooses the renderer.

### NotNull semantics and where validation lives

On a base `model.Int()`/`model.Bool()` kind, `NotNull` rejects the **zero value**
(`ValidateFields`: `NotNull && IsZeroPtr`) — so `day_of_week` 0 (Sunday),
`is_open == false`, and midnight (`0` minutes) are all legitimate values that
must not be rejected. The domain records therefore carry `NotNull` only to drive
`NOT NULL` DDL; the *semantic* invariants live in the service methods — the
single fail-closed chokepoint every write path crosses:

| Invariant | Sentinel |
|-----------|----------|
| `day_of_week` in 0..6 | `ErrInvalidDay` |
| open window within 0..1439, `open_min < close_min` | `ErrInvalidMinutes` |
| second holiday/closure on the same date | `ErrDuplicateDate` |
| missing row | `ErrNotFound` |

Transport args repeat required **text** fields with `NotNull` (holiday `name`,
closure `reason`, removal `id`) so the op boundary rejects empty input with a
400 before reaching the service; every op that mutates still funnels through the
service's domain checks.

## 4. `GetDayBounds`/`GetDayDetail` and resolution precedence

> **Amended 2026-09-08.** The port originally returned a module-local
> `DayBounds` carrying `ClosedBy`. That forced any consumer to import this
> module just to name the type — exactly the coupling
> `veltylabs/appointment_booking` must not have. See
> `webtyp/docs/AGENDA_DOMAIN_MASTER_PLAN.md` §3-bis for the full rationale.

`interfaces.go` splits the answer in two, by audience:

```go
// The value a sibling domain module (e.g. appointment_booking) consumes. It
// declares its OWN interface returning tinytime.DayBounds — a neutral type in
// webtyp/time that both sides already import — and this Module satisfies it
// structurally. No import in either direction, no adapter.
func (m *Module) GetDayBounds(date int64) (tinytime.DayBounds, error)

// This module's own richer answer: the same bounds PLUS why a closed day is
// closed. Used by the get_day_bounds op and this module's own UI.
func (m *Module) GetDayDetail(date int64) (DayDetail, error)

type DayDetail struct {
    Bounds   tinytime.DayBounds
    ClosedBy ClosedReason // "" when Bounds.Open is true
}
```

`Reader` (declared here) is a **convenience alias for this module's own
consumers**, not the contract a sibling module names — a sibling declares its
own interface shaped like `GetDayBounds`'s signature and this `Module` satisfies
it structurally:

```go
type Reader interface {
    GetDayBounds(date int64) (tinytime.DayBounds, error)
}
```

`GetDayBounds` is a thin wrapper — `GetDayDetail` is the single resolution path;
`GetDayBounds` calls it and discards `ClosedBy`. There is exactly one place the
precedence is decided.

Resolution, in this exact order (unchanged by the amendment):

1. A `Closure` on that date → `{Open: false, ClosedBy: ClosedLocal}`.
2. A `Holiday` on that date → `{Open: false, ClosedBy: ClosedHoliday}`.
3. `BusinessHours` for that weekday with `is_open == false` (or no row at all —
   absence of a schedule is closed) → `{Open: false, ClosedBy: ClosedWeekly}`.
4. Otherwise `{Open: true, OpenMin, CloseMin}`.

## 5. Events

Every successful mutation publishes `EventCalendarChanged` with a fully
populated `CalendarChangedPayload` (implements `model.Encodable`):

| Method | Kind | Closed | Range |
|--------|------|--------|-------|
| `AddHoliday` | `HOLIDAY` | true | `[date, date]` |
| `RemoveHoliday` | `HOLIDAY` | false | `[date, date]` |
| `AddClosure` | `CLOSURE` | true | `[date, date]` |
| `RemoveClosure` | `CLOSURE` | false | `[date, date]` |
| `UpsertBusinessHours` (new row) | `BUSINESS_HOURS` | false | `0,0` |
| `UpsertBusinessHours` (narrowing) | `BUSINESS_HOURS` | true | `0,0` |
| `UpsertBusinessHours` (widening) | `BUSINESS_HOURS` | false | `0,0` |

`Closed` is the field a subscriber trusts: `true` only when the change closes
time — the only direction that can invalidate an existing reservation. When in
doubt the module publishes `Closed: true` (a recompute is cheap; a skipped real
closure strands a patient on a shut day). A weekly rule has no bounded range, so
business-hours events carry `0,0` and a subscriber recomputes its whole horizon.

## 6. Ops

| Op | Action | Resource | Description |
|----|--------|----------|--------------|
| `list_business_hours` | `r` | `business_hours` | All 7 weekday rows, by `day_of_week` |
| `upsert_business_hours` | `cr` | `business_hours` | Create-or-update one weekday window |
| `get_day_bounds` | `r` | `business_hours` | `DayDetail` for one date (bounds + `ClosedBy`) |
| `list_holidays` | `r` | `holiday` | All holidays |
| `add_holiday` | `c` | `holiday` | Register a holiday (duplicate date → 409) |
| `remove_holiday` | `d` | `holiday` | Remove by id (missing → 404) |
| `list_closures` | `r` | `closure` | All closures |
| `add_closure` | `c` | `closure` | Register a closure (duplicate date → 409) |
| `remove_closure` | `d` | `closure` | Remove by id (missing → 404) |

Status mapping: `400` decode/validation (`ErrInvalidDay`/`ErrInvalidMinutes`) ·
`404` not-found (`ErrNotFound`) · `409` conflict (`ErrDuplicateDate`) · `500`
genuine internal errors only.

The upsert op declares `model.Create|model.Update` (a bitmask) because it really
can do both; declaring only `Update` would let an update-only principal create
rows. Three resources (`business_hours`, `holiday`, `closure`) because an
administrator who may add a holiday is not necessarily the one who may rewrite
opening hours.

Range filtering (`ListHolidays(from, to)`, `ListClosures(from, to)`) lives on
the **service** methods for in-process consumers; the list **ops** take no args
and return the whole horizon (a view lists all).

## 7. Composition root example

```go
cal, err := businesscalendar.New(db, businesscalendar.Deps{
    IDs:      unixid.NewUnixID(), // injected — the module never builds one
    Publisher: broker,            // events.Broker, or nil
})
if err != nil { /* ... */ }

cal.MountOperations(opRegistry) // router.OperationRegistry

// appointment_booking never imports this package — it declares its own
// interface shaped like GetDayBounds's signature, and *cal satisfies it
// structurally. This module's own Reader alias is for THIS module's callers:
var reader businesscalendar.Reader = cal
bounds, err := reader.GetDayBounds(midnightUTC) // tinytime.DayBounds — no ClosedBy

detail, err := cal.GetDayDetail(midnightUTC) // DayDetail — WITH ClosedBy, this module's own UI

hoursView := businesscalendar.NewBusinessHoursView(caller) // router.Caller
holidaysView := businesscalendar.NewHolidaysView(caller)
closuresView := businesscalendar.NewClosuresView(caller)
```

## 8. Schema

See [`docs/diagrams/database.md`](diagrams/database.md).
