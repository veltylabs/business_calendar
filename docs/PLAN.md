---
PLAN: "fix(ui): business hours editor shows a 24h clock, not minutes-since-midnight"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 15090588848331827000
---

> Read AGENTS.md at the repo root before any change. This plan is complete: do not guess; if something does not fit, stop and ask in the PR.

## Goal

The admin screen to edit weekly business hours (`ui/browser.go` → `hoursView`, built from `BusinessHoursModel`)
renders `open_min`/`close_min` as a plain number input today, because their `Kind` is `input.Number()`. The
person configuring the clinic's hours has to compute and type minutes-since-midnight ("480" for 08:00) instead
of picking a time. Fix: the stored field becomes a 24h clock string ("HH:MM") via `input.Hour()`, and every
place that currently treats it as an int minutes value converts at that one boundary.

**Never in scope:** the list view already renders readable hours (`hours_view.go`'s `hhmm`, `view.go`'s
`minutesHHMM`) — only the edit form is wrong. Do not touch how `Holiday`/`Closure`'s `specific_date` work; this
plan is only about `BusinessHoursModel.open_min`/`close_min`.

## Git rules (strict)

- Work on the branch codejob created. Never run `git reset`, `git checkout -- <file>`, `git clean`, `git stash`
  or any command that discards files. Commit often (`git add -A && git commit -m "..."`).
- Do not edit `model_orm.go` by hand: it is generated (Stage 2).
- **Never pause to ask permission or to report progress.** When the delivery check passes: commit, push and
  open the pull request. If something still fails after 30 minutes, open the pull request anyway and list in
  it exactly what fails (command and output).
- If `gotest` cannot run its browser WASM lane (`wasmbrowsertest` missing), run `GOOS=js GOARCH=wasm go vet
  ./...` instead and say so in the pull request; it does not block delivery.

## Tools

```bash
go install webtyp.com/ormc/cmd/ormc@latest
go install webtyp.com/devflow/cmd/gotest@latest
```

## Stage 1 — `model.go`: the field becomes a clock, not a number

Change, in `BusinessHoursModel` (currently lines 27-28):

```go
{Name: "open_min", Type: input.Number(), Label: "Opens at (minutes)", NotNull: true},
{Name: "close_min", Type: input.Number(), Label: "Closes at (minutes)", NotNull: true},
```

to:

```go
{Name: "open_min", Type: input.Hour(), Label: "Opens at", NotNull: true},
{Name: "close_min", Type: input.Hour(), Label: "Closes at", NotNull: true},
```

`input.Hour()` (`webtyp.com/input/hour.go`) renders an `<input type="time">` and stores the value as the
"HH:MM" string the browser gives it (its `Storage()` is `model.FieldText`, inherited from `Base`); it already
rejects `24:xx` in `Validate`. The Go field in the regenerated `model_orm.go` becomes `OpenMin string` /
`CloseMin string` (was `int64`/`int16` — check the exact prior type by reading the current `model_orm.go`
before regenerating, so you know what every reader expects).

Also retype `UpsertBusinessHoursArgsModel`'s `open_min`/`close_min` (currently `model.Int()`, lines ~151-152)
to `model.Text()`: this struct is what the edit form actually submits (see Stage 3), so its shape must match.

## Stage 2 — regenerate

```bash
cp go.mod /tmp/x.mod.bak && ormc && cp /tmp/x.mod.bak go.mod && go mod tidy && gofmt -w model_orm.go
```

## Stage 3 — fix every reader of `OpenMin`/`CloseMin` as an int

Add two small helpers in `convert.go` (new file) wrapping `webtyp.com/time`'s existing `ParseTime(string)
(int16, error)` and `FormatMinutes(int) string` — do not duplicate their logic:

```go
package businesscalendar

import tinytime "webtyp.com/time"

func parseClock(hhmm string) (int, error) {
	m, err := tinytime.ParseTime(hhmm)
	if err != nil {
		return 0, ErrInvalidMinutes
	}
	return int(m), nil
}

func formatClock(minutes int) string { return tinytime.FormatMinutes(minutes) }
```

Then, exact call sites:

- **`module.go`, `hoursCloseTime`** (around line 75): `next.OpenMin > prev.OpenMin || next.CloseMin <
  prev.CloseMin` compares strings lexicographically today by accident of type change — parse both sides first:
  `nextOpen, _ := parseClock(next.OpenMin)` etc. (the values are already validated by `UpsertBusinessHours`
  before this runs, so the error is unreachable here — ignore it, do not propagate a new error return from
  `hoursCloseTime`, which stays `bool`).
- **`module.go`, `UpsertBusinessHours`** (around line 90-100): replace the int bounds check
  (`h.OpenMin < 0 || ...`) with: parse both `h.OpenMin`/`h.CloseMin` via `parseClock`, requiring no error, and
  comparing the resulting minutes (`openMin >= closeMin → ErrInvalidMinutes`). When `!h.IsOpen`, normalize to
  `h.OpenMin, h.CloseMin = "00:00", "00:00"` (was `0, 0`).
- **`interfaces.go`, `getWeeklyBounds`** (around line 97): `tinytime.DayBounds{Open: true, OpenMin:
  int(bh.OpenMin), CloseMin: int(bh.CloseMin)}` — replace `int(bh.OpenMin)` with `mustParseClock(bh.OpenMin)`
  (add a `mustParseClock` that parses and, since the row was validated at write time, panics only if the
  invariant is broken — follow the pattern already used elsewhere in this repo for "stored data is already
  valid" assumptions; if no such pattern exists, just ignore the error and default to 0, never panic in a
  read path).
- **`ops.go`, `opUpsertBusinessHours`** (around line 117): `OpenMin: args.OpenMin, CloseMin: args.CloseMin` —
  unchanged, both sides are now strings.
- **`hours_view.go`**: `hhmm(min int64) string` and its two call sites (`w.String("opens", hhmm(v.row.OpenMin))`,
  `w.String("opens", ...)`) — `v.row.OpenMin`/`CloseMin` are now already "HH:MM" strings, so delete `hhmm` and
  write the string fields directly: `w.String("opens", v.row.OpenMin)`, `w.String("closes", v.row.CloseMin)`.
- **`view.go`, `hoursLabel`/`minutesHHMM`** (around line 60-70): `minutesHHMM(int(r.OpenMin)) + "–" +
  minutesHHMM(int(r.CloseMin))` — delete `minutesHHMM` (now dead) and write `r.OpenMin + "–" + r.CloseMin`
  directly.

Search the whole module for any other `.OpenMin`/`.CloseMin` read (`grep -rn "OpenMin\|CloseMin" --include="*.go" .`,
excluding `model_orm.go` and `_test.go`) and fix every one the same way — the list above is from today's code
and must not be treated as exhaustive if it has drifted.

## Stage 4 — tests

Fix every existing test that builds a `BusinessHours{OpenMin: 480, ...}` (int literal) to use the "HH:MM"
string instead (`OpenMin: "08:00"`). Add one new test: `UpsertBusinessHours` rejects an invalid clock string
("25:00", "8:00" without the leading zero, "08:60") with `ErrInvalidMinutes`, and rejects open >= close
("09:00"/"09:00") the same way.

## Stage 5 — docs

`README.md`/`docs/ARCHITECTURE.md` if either documents the `open_min`/`close_min` encoding as "minutes since
midnight" — update to "24h clock string (HH:MM)".

## Delivery check

- `gotest` green (vet, race, tests, wasm).
- `grep -rn "OpenMin\|CloseMin" --include="*.go" .` outside `model_orm.go`/tests shows no remaining int
  arithmetic on these two fields.
- `git diff --stat main --diff-filter=D` empty (no file deleted outside what this plan names: none).
- No behaviour change beyond the edit form's widget and the string encoding — `GetDayBounds`/`GetWeekdayBounds`
  still return `tinytime.DayBounds{OpenMin, CloseMin int}` unchanged to their consumers (`room_layout`,
  `appointment_booking`): this plan only moves the boundary conversion from "never" to "inside this module".

## Stages

| # | File(s) | What |
|---|---|---|
| 1 | `model.go` | `open_min`/`close_min` → `input.Hour()`; `UpsertBusinessHoursArgsModel` → `model.Text()` |
| 2 | `model_orm.go` | regenerate |
| 3 | `convert.go` (new), `module.go`, `interfaces.go`, `hours_view.go`, `view.go` | boundary conversion, delete now-dead `hhmm`/`minutesHHMM` |
| 4 | `tests/*` | string literals, new validation tests |
| 5 | `README.md`/`docs/ARCHITECTURE.md` | encoding note, if present |
