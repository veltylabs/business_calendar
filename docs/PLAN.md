---
PLAN: "feat: operations say what they do (Route.Describe), and list_business_hours adds readable days and times"
TAG: v0.4.0
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 8057019763708886764
---

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.

# Plan — `business_calendar`: operations an AI assistant can read

## 0. Context

This module owns a clinic's business hours, holidays and closures. It registers its operations
on a `router.OperationRegistry` in `MountOperations` (`ops.go`). Through `webtyp.com/mcp`, every
operation becomes a **tool** that an AI assistant can call. The first such assistant is Jose,
in the clinic's staff chat. It runs Qwen3.5-0.8B, a small model, in the staff member's browser.

Two things measured with that model (2026-09-30) make the tools hard for it:

1. **No descriptions.** `tools/list` shows only names such as
   `business_calendar.list_business_hours`. `webtyp.com/router` v0.3.0 added
   `Route.Describe(text string) Route`, and `webtyp.com/mcp` v0.2.39 publishes that text as the
   tool's description. This module does not call it yet.
2. **Minutes since midnight.** `list_business_hours` answers rows such as
   `{"day_of_week":2,"open_min":480,"close_min":1080,"is_open":true,…}`. Asked "¿Hasta qué hora
   atendemos hoy?", the model had to turn 1080 into 18:00 and pick today's row, and answered
   correctly **5 times in 10**. Given the same rows with a readable day and times
   (`"day":"Tuesday","opens":"08:00","closes":"18:00"`), it answered correctly **9 times in 10**.
   English day names matter: the assistant's date line says "Tuesday", and with "martes" in the
   data it dropped to 6 in 10.

**The fix:** every operation declares a description, and `list_business_hours` adds three
readable fields to each row. Every existing field stays, so no current caller breaks.

## Development rules (inline)

- Follow this repository's `AGENTS.md` (the whitelist of imports, the test layout in `tests/`).
  Library code compiles for the browser: never `fmt`, `errors`, `strings`, `strconv` (use
  `webtyp.com/fmt`), `encoding/json` or `map[K]V` in non-test files.
- `go get webtyp.com/router@v0.3.0` (already in `go.mod`).
- Do not edit `model_orm.go` by hand (ormc output).
- No `TODO`. `gotest` green.

## Design gate (api-design — five answers)

1. **Prior art.**
   - OpenAPI operations carry `description`, and their examples use ISO or `HH:MM` times, never
     minute counts.
   - Google Business Profile's `regularHours` uses `openDay: "TUESDAY"` and `openTime` as hours
     and minutes.
   - schema.org `OpeningHoursSpecification` uses `dayOfWeek` and `opens`/`closes` "HH:MM".

   The new field names (`day`, `opens`, `closes`) are schema.org's.
2. **Novice-name test.** `opens: "08:00"`, `closes: "18:00"`, `day: "Tuesday"` read as what
   they are.
3. **Complexity ledger.**

   ```
   Concepts the developer must learn   +0
   Files they must touch to do X       +0
   Lines at the call site              +1 per operation (.Describe)
   Ways to do the same thing           +0 (the minutes stay the stored truth; the new fields are its rendering)
   ```

4. **Where it belongs.** In this module: it knows what its operations do and how its data reads.
5. **What it deletes.** Nothing. The additions are new fields.

## Stage 1 — descriptions (`ops.go`)

Add `.Describe(...)` to every operation in `MountOperations`, with exactly these texts:

| Operation | Description |
|---|---|
| `list_business_hours` | `Horario de atención del consultorio para cada día de la semana: día (Monday … Sunday), si abre, y la hora de apertura y de cierre (HH:MM).` |
| `upsert_business_hours` | `Crea o cambia el horario de atención de un día de la semana.` |
| `get_day_bounds` | `Si el consultorio abre en una fecha y en qué minutos del día, considerando feriados y cierres.` |
| `list_holidays` | `Feriados registrados del consultorio.` |
| `add_holiday` | `Registra un feriado: el consultorio no atiende esa fecha.` |
| `remove_holiday` | `Elimina un feriado registrado.` |
| `list_closures` | `Cierres extraordinarios del consultorio (fechas u horas sin atención).` |
| `add_closure` | `Registra un cierre extraordinario.` |
| `remove_closure` | `Elimina un cierre extraordinario.` |

Put the texts in named constants next to the `Op…` names (`DescListBusinessHours`, …).

## Stage 2 — readable business hours (`hours_view.go`, `ops.go`)

- New unexported type `businessHoursView struct{ row *BusinessHours }` with `EncodeFields(w)`:
  it writes **every field** `BusinessHours.EncodeFields` writes, in the same order (call
  `v.row.EncodeFields(w)`), and then:
  - `day`: English weekday name of `day_of_week` (0 `Sunday`, 1 `Monday` … 6 `Saturday`), from a
    `[7]string` constant table;
  - only when `is_open` is true, `opens` and `closes`: `open_min` and `close_min` as `HH:MM`
    (zero-padded, 24-hour: 480 → `08:00`, 1080 → `18:00`, 0 → `00:00`, 1440 → `24:00`).
    Build the string with a small helper `hhmm(min int64) string`, with no `fmt.Sprintf`
    padding tricks that TinyGo cannot compile.
  - It implements whatever `model` interfaces `ctx.Encode` needs for a list, following how
    `BusinessHoursList` is encoded today (read `model_orm.go` for `BusinessHoursList` and mirror
    it in a `businessHoursViewList`).
- `opListBusinessHours` encodes a `businessHoursViewList` built from the rows instead of the
  `BusinessHoursList`.

## Stage 3 — tests (`tests/`)

- `tests/ops_describe_test.go`: harvest the module with `mcp.HarvestOps` if `webtyp.com/mcp` is
  already in the test dependencies; otherwise use the router's mock registry the other op tests
  use (`tests/ops_test.go`). Every operation has a non-empty description equal to its constant.
- `tests/ops_test.go` (extend): `list_business_hours` over a Tuesday row
  `{day_of_week 2, open_min 480, close_min 1080, is_open true}` and a closed Sunday row encodes:
  - the Tuesday row with every old field plus `"day":"Tuesday","opens":"08:00","closes":"18:00"`;
  - the Sunday row with `"day":"Sunday"` and no `opens`/`closes`.
- `hhmm`: 0, 480, 1080, 1439 and 1440 give `00:00`, `08:00`, `18:00`, `23:59` and `24:00`.

## Stage 4 — docs

`README.md` (or this repository's operations doc): the `list_business_hours` response shows the
new fields with the example above, and one sentence says why they exist (assistants read times,
not minute counts).

## Stages

| Stage | Files | Acceptance |
|---|---|---|
| 1 | `ops.go` | every `reg.Operation` chain calls `.Describe` |
| 2 | `hours_view.go`, `ops.go` | builds; old fields unchanged |
| 3 | `tests/` | new tests pass; existing tests green |
| 4 | docs | the new fields documented |
| all | — | `gotest` green; `GOOS=js GOARCH=wasm go build ./...` |
