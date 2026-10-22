---
PLAN: "feat: BankDayAfter — the bank business day N days after a date (weekends and holidays skipped)"
TAG: v0.5.0
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 17751961171220088934
---

> Read AGENTS.md at the repo root before any change.

# Plan — `BankDayAfter`

## Why

A card processor deposits a payment "N business days" later (TUU: 1, 2, 5 or 10). Business days for a bank are
Monday to Friday, minus legal holidays. This module already stores the legal holidays (`Holiday`), so it answers
"which day is N bank days after this one". `payment_manager` will use it (through a one-method interface of its own)
to date the deposit of every card payment. The establishment's opening hours and its own closures do **not** count:
a bank does not close because the clinic does.

Language: identifiers and error strings in English; **README, docs, diagrams and comments in Spanish**, each exported
symbol with a `// Ejemplo: …` line.

## Delivery rules

- **Commit after every stage.** Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash`.
- Never pause to ask permission or to report progress. When the Delivery check passes: push and open the PR. Stuck
  for 30 minutes: open the PR anyway, listing what fails.
- A PR that does not compile, or whose tests could not run, is a failed delivery.
- The code blocks below are the implementation: copy them.

## Tools and pinned versions

```bash
go install webtyp.com/devflow/cmd/gotest@latest
export GOPRIVATE='webtyp.com/*,github.com/veltylabs/*' GOFLAGS=-mod=mod
go get webtyp.com/storage@v0.1.10 && go mod tidy
```

## APIs used (`webtyp.com/time`, version in `go.mod`, imported here as `tinytime`)

`tinytime.Day Unix = 86400`, `tinytime.MidnightUTC(u Unix) Unix`, `tinytime.Weekday(u Unix) int` (0 = Sunday … 6 =
Saturday). This module's own `ListHolidays(from, to int64) ([]Holiday, error)` (`Holiday.SpecificDate` = midnight UTC).

## Stage 1 — `bankdays.go` (new, complete)

```go
package businesscalendar

import tinytime "webtyp.com/time"

// ErrNegativeDays: BankDayAfter con días negativos.
const ErrNegativeDays = domainError("business_calendar: days must be zero or more")

// BankDayAfter devuelve la medianoche UTC del día hábil bancario que está days días hábiles después de from.
// Día hábil bancario: lunes a viernes que no es feriado legal (Holiday). El horario y los cierres propios del
// establecimiento no cuentan. days == 0 devuelve el mismo día si es hábil, si no el siguiente hábil.
// Ejemplo: deposito, err := cal.BankDayAfter(cobradoEl, 2) // cobro del viernes con abono a 2 días → martes
func (m *Module) BankDayAfter(from tinytime.Unix, days int) (tinytime.Unix, error) {
	if days < 0 {
		return 0, ErrNegativeDays
	}
	day := tinytime.MidnightUTC(from)
	// Feriados de un margen amplio: days hábiles nunca ocupan más de days*2+14 días corridos.
	holidays, err := m.ListHolidays(int64(day), int64(day+tinytime.Unix(days*2+14)*tinytime.Day))
	if err != nil {
		return 0, err
	}
	isBankDay := func(d tinytime.Unix) bool {
		wd := tinytime.Weekday(d)
		if wd == 0 || wd == 6 {
			return false
		}
		for _, h := range holidays {
			if tinytime.Unix(h.SpecificDate) == d {
				return false
			}
		}
		return true
	}
	for !isBankDay(day) {
		day += tinytime.Day
	}
	for left := days; left > 0; {
		day += tinytime.Day
		if isBankDay(day) {
			left--
		}
	}
	return day, nil
}
```

`domainError` is declared in `model.go` (next to `ErrDuplicateDate`); `Holiday.SpecificDate` is `int64`. **Commit.**

## Stage 2 — Tests (`tests/bankdays_test.go`)

Dates are midnight UTC built with `tinytime.Date(y, m, d, 0, 0, 0)`. 2026-09-18 is a Friday (Chilean holiday) and
2026-09-19 a Saturday (holiday).

| Case | Setup | Call | Expect |
|---|---|---|---|
| B1 | none | Mon 2026-10-05 + 2 | Wed 2026-10-07 |
| B2 | none | Fri 2026-10-09 + 1 | Mon 2026-10-12 |
| B3 | none | Fri 2026-10-09 + 2 | Tue 2026-10-13 |
| B4 | holiday Mon 2026-10-12 | Fri 2026-10-09 + 1 | Tue 2026-10-13 |
| B5 | none | Sat 2026-10-10 + 0 | Mon 2026-10-12 |
| B6 | none | Wed 2026-10-07 14:30 UTC + 0 | Wed 2026-10-07 00:00 UTC |
| B7 | holidays 2026-09-18 and 2026-09-19 | Thu 2026-09-17 + 1 | Mon 2026-09-21 |
| B8 | closure (not holiday) Tue 2026-10-13 | Mon 2026-10-12 + 1 | Tue 2026-10-13 (closures do not count) |
| B9 | none | days −1 | `ErrNegativeDays` |
| B10 | none | Mon + 10 | Mon two weeks later |

**Commit.**

## Stage 3 — Docs (Spanish)

`README.md`: one line in the API list — `BankDayAfter(desde, días)`: día hábil bancario N días después (lunes a
viernes sin feriados; el horario de la clínica no cuenta). `docs/ARCHITECTURE.md`: short section `## Días hábiles
bancarios` with the B4 example. **Commit.**

## Stage 4 — Spanish (only what is still in English)

Every `README.md`, `docs/*` file, diagram and code comment in Spanish; translate only what is still English; each
exported symbol's comment has a `// Ejemplo: …` line. **Commit.**

## Delivery check

```bash
gotest
```

## Stages

| # | What | Files |
|---|---|---|
| 1 | `BankDayAfter` | `bankdays.go` |
| 2 | Tests | `tests/bankdays_test.go` |
| 3 | Docs | `README.md`, `docs/ARCHITECTURE.md` |
| 4 | Spanish | every English `*.md` and `*.go` comment |
