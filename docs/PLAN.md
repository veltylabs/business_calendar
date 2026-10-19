---
PLAN: "feat(ui)!: Calendario is a platformd group (Horario, Feriados, Cierres) — decktabs is gone from components"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 7873193190015770557
PR: https://github.com/veltylabs/business_calendar/pull/6
REVIEWED_COMMIT: cd82c498343cac89c4ed8772c64354892db06cf8
---

> Read AGENTS.md at the repo root before any change.

Part of the cross-library UI-consistency wave: libraries published `widget` v0.6.43, `components` v0.8.34,
`form` v0.4.43 and `layout` v0.3.52; this plan moves the module onto them.

## Tools

```
go install webtyp.com/devflow/cmd/gotest@latest
```

## Why

`webtyp.com/components` v0.8.31 **deleted the `decktabs` package**: grouping several screens under one
navigation item is the navigation shell's job, not a widget's. `ui/browser.go` still imports
`webtyp.com/components/decktabs`, so `go mod tidy` fails here:

```
module webtyp.com/components@latest found (v0.8.34), but does not contain package webtyp.com/components/decktabs
```

The shell's answer is `platformd.NewGroupUIModule`: one navigation entry that holds child modules, each
with its own screen. The reference implementation is
[`item_catalog/ui/browser.go`](https://github.com/veltylabs/item_catalog/blob/main/ui/browser.go).

## Design gate

Public API: none changes. `ui.Browser(caller, ids, tenantID) (platformd.UIModule, error)` keeps its signature
and its `ID`/`Label` constants; only the concrete value it returns changes (a group instead of a single module).

1. Prior art: `item_catalog/ui/browser.go` (group of two crudviews). 2. Novice-name test: nothing new is named.
3. Complexity ledger: −1 import, −1 widget. 4. Where it belongs: here, the module composes its own screens.
5. What it deletes: the `decktabs` use and its doc comment.

## Stage 1 — dependencies

In `go.mod` set (then `go mod tidy`):

```
go get webtyp.com/layout@v0.3.52 webtyp.com/components@v0.8.34 webtyp.com/form@v0.4.43 webtyp.com/widget@v0.6.43
go mod tidy
```

## Stage 2 — `ui/browser.go`

Replace the `decktabs.DeckTabs` composition with a group. Keep the three `crudview.New` calls exactly as they
are (same `ParentID`, `Presenter`, `IDs`). Then:

```go
hoursMod := platformd.NewUIModule(ID+"."+TabHoursID, TabHoursLabel, svg.Icon(ID), hoursView)
holidaysMod := platformd.NewUIModule(ID+"."+TabHolidaysID, TabHolidaysLabel, svg.Icon(ID), holidaysView)
closuresMod := platformd.NewUIModule(ID+"."+TabClosuresID, TabClosuresLabel, svg.Icon(ID), closuresView)

return platformd.NewGroupUIModule(ID, Label, svg.Icon(ID), []platformd.UIModule{
	hoursMod, holidaysMod, closuresMod,
}), nil
```

- All three children reuse the module's own icon `svg.Icon(ID)` (already registered in `ui/svg.go`); do **not**
  add new artwork or sprite entries.
- Rewrite the doc comment of `Browser`: it composes the module's three screens (business hours, holidays,
  closures) into **one navigation group**, because the module has three presenters; each crudview keeps its
  own `ParentID` (three crudviews cannot share the id they mount on). Remove every mention of `decktabs`.
- Delete the import `webtyp.com/components/decktabs`.

## Stage 3 — tests and docs

- `tests/ui_view_wasm_test.go` calls `m.View()` on what `ui.Browser` returns. A group's `View()` is `nil`
  by design: change that test to assert the group (`m.(platformd.GroupModule)`), that `Children()` has exactly
  three modules whose `ModelName()` are `business_calendar.hours`, `business_calendar.holidays` and
  `business_calendar.closures`, and then run the existing render/assertions against each child's `View()`.
  Extend the existing test; do not add a second test that repeats what it already checks.
- `docs/ARCHITECTURE.md`: replace any description of tabs with the group description. `README.md`: same, if it
  mentions tabs.

## Delivery check

```
grep -rn "decktabs" . --include=*.go --include=*.md    # → empty
test -z "$(grep -n '^replace' go.mod)"                 # → no local replace
gotest                                                  # vet, race, tests and wasm green
git diff --stat main                                    # only the files named above
```

## Stages

| # | Stage | Files |
|---|---|---|
| 1 | dependencies | `go.mod`, `go.sum` |
| 2 | group instead of tabs | `ui/browser.go` |
| 3 | tests and docs | `tests/ui_view_wasm_test.go`, `docs/ARCHITECTURE.md`, `README.md` |

## Executor notes
- Added the executor notes below.
- Executed stages 1-3.
- Using `~/go/bin/gotest` instead of just `gotest`.
- Used manual wasm build checking due to missing `wasmbrowsertest` binary.
- Did not update README.md because it did not mention tabs.
