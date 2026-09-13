---
PLAN: "refactor: move schema creation out of New into a migrate subpackage"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 366016548153605398
PR: https://github.com/veltylabs/business_calendar/pull/1
---

> This plan is dispatched via the CodeJob workflow. See skill: **agents-workflow**.

# Plan — extract DDL from `New` into `migrate/`

You are an agent with **no prior context** and you have **only this repository**
(`github.com/veltylabs/business_calendar`). Everything you need is inline.

## 1. The problem

`New` creates this module's three tables as a side effect of construction:

```go
// module.go — current code
func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.IDs == nil {
		return nil, fmt.Err("business_calendar: Deps.IDs is required")
	}
	// ddl.Compiler is an optional capability — only SQL backends (sqlt,
	// postgres) implement it; storage/mem creates tables lazily and needs no
	// DDL. The type assertion, not an unconditional call, is what keeps the
	// module backend-agnostic here.
	if ddlCompiler, ok := db.RawConn().(ddl.Compiler); ok {
		if err := ddl.New(db.RawConn(), ddlCompiler).CreateTable(&BusinessHours{}); err != nil {
			return nil, err
		}
		if err := ddl.New(db.RawConn(), ddlCompiler).CreateTable(&Holiday{}); err != nil {
			return nil, err
		}
		if err := ddl.New(db.RawConn(), ddlCompiler).CreateTable(&Closure{}); err != nil {
			return nil, err
		}
	}
	return &Module{db: db, ids: deps.IDs, pub: deps.Publisher}, nil
}
```

Two defects follow from that:

1. **Every application boot runs DDL.** Schema reconciliation is a deploy-time
   step with its own ordering and its own failure mode; running it on every
   process start means a running server can alter a database, and an app has no
   way to boot the module without granting it DDL rights.
2. **`webtyp.com/ddl` is pulled into the WASM build graph.** A consuming app's
   client imports this package for `NewView`. Because `New` references `ddl`
   from the root package, `ddl` is in the graph of every build that imports the
   root package, regardless of build tags on the consumer's side.

Consumers of this ecosystem run migrations from a dedicated `cmd/migrate`
binary that calls `<module>/migrate.Migrate(conn, compiler)` once per module,
in a fixed order. Four sibling modules already follow it. This one does not,
so it cannot be wired without breaking that contract.

## 2. The target shape — copy it exactly

This is the sibling `github.com/veltylabs/device_manager`'s `migrate/migrate.go`,
already published and in production. Reproduce its structure, its package name,
and the spirit of its doc comment:

```go
package migrate

import (
	"webtyp.com/ddl"

	devicemanager "github.com/veltylabs/device_manager"
)

// Migrate reconciles the database schema device_manager owns: Device.
//
// It is deliberately NOT called by New, and deliberately lives in its own
// package rather than a new file in the root package: nothing on a
// consuming app's WASM build path (its view.go, which imports the root
// devicemanager package for devicemanager.NewView) ever imports
// "github.com/veltylabs/device_manager/migrate" — so webtyp.com/ddl never
// enters that build graph, regardless of build tags on the consumer's side.
//
// conn is a ddl.Execer, not an *orm.DB, so a deploy-time transport that can
// only execute DDL satisfies it. An *orm.DB's RawConn() also satisfies it,
// for local/test callers:
//
//	conn, _ := postgres.Open(dsn)
//	compiler, _ := conn.(ddl.Compiler)
//	err := migrate.Migrate(conn, compiler)
func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error {
	return ddl.New(conn, ddlCompiler).CreateTable(&devicemanager.Device{})
}
```

## 3. Design gate

This plan adds one exported symbol (`migrate.Migrate`) and removes a side
effect. The five answers:

1. **Prior art.** Django and Rails both separate `migrate` from application
   boot into a dedicated command; Ent and Atlas expose schema reconciliation as
   an explicit API the application calls when it chooses. None reconciles
   schema inside the object constructor. We follow the same split; we differ
   from Django/Rails only in having no migration-history table — every
   statement is `CREATE TABLE IF NOT EXISTS`, so the step is idempotent and
   needs no version ledger.
2. **Novice-name test.** "Migrate the connection with this compiler" reads as a
   sentence. The package name `migrate` and the function `Migrate` are the
   names the four sibling modules already use — a reader who has seen one
   knows this one.
3. **Complexity ledger.**
   - Concepts: `+0` (the `migrate` package is an existing ecosystem concept, not a new one).
   - Files: `+2` (`migrate/migrate.go`, `migrate/migrate_test.go`).
   - Call-site lines: `−14` in `module.go`, `+1` per deploying app.
   - Ways to do it: `−1` — schema creation had two homes (construction and, for
     apps that already ran `cmd/migrate`, the migrate step). Now it has one.
   - Net: negative.
4. **Where it belongs.** Its own package, not a file in the root package,
   because the point is to keep `webtyp.com/ddl` out of the root package's
   import graph. A `migrate.go` in the root package would fix the boot-time
   side effect and leave the WASM graph defect untouched.
5. **What it deletes.** The whole `if ddlCompiler, ok := ...` block in `New`,
   and the `webtyp.com/ddl` import from `module.go`.

## 4. Stages

### Stage 1 — create `migrate/migrate.go`

New file `migrate/migrate.go`, `package migrate`. One exported function:

```go
func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error
```

It creates the three tables this module owns, **in this exact order**:
`&businesscalendar.BusinessHours{}`, `&businesscalendar.Holiday{}`,
`&businesscalendar.Closure{}`. Return the first error unwrapped; do not wrap it.

There is **no** `ddl.Compiler` type assertion in this function — the caller
passes the compiler explicitly. The optional-capability check existed only
because `New` received an `*orm.DB` that might be backed by `storage/mem`;
`Migrate` is never called against `storage/mem`.

Import the root package with the alias `businesscalendar` (matching its
`package` clause), the same way the device_manager example aliases its own.

Write the doc comment on `Migrate` explaining **both** reasons this lives in
its own package (deploy-time step, and keeping `webtyp.com/ddl` out of the WASM
build graph) and showing the three-line caller example. Do not copy
device_manager's comment verbatim — it names the wrong module and the wrong
tables.

### Stage 2 — strip the side effect from `New`

In `module.go`, `New` becomes:

```go
// New connects the module to an already-connected *orm.DB; the schema is assumed
// to already exist — see the migrate subpackage.
func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.IDs == nil {
		return nil, fmt.Err("business_calendar: Deps.IDs is required")
	}
	return &Module{db: db, ids: deps.IDs, pub: deps.Publisher}, nil
}
```

Delete the `webtyp.com/ddl` import from `module.go`. Do not leave a deprecated
wrapper, a flag, or a "migrate on first use" fallback — there is no compatibility
shim of any kind.

**Anti-footgun:** `webtyp.com/ddl` must stay in `go.mod` `require` — the new
`migrate` package uses it. Do not "clean up" that dependency.

### Stage 3 — `migrate/migrate_test.go`

`package migrate_test`. Mirror device_manager's test, adapted to three tables:

```go
type dummyExecer struct{ calls []string }

func (d *dummyExecer) Exec(query string, args ...any) error {
	d.calls = append(d.calls, query)
	return nil
}

type dummyCompiler struct{}

func (d *dummyCompiler) CompileDDL(stmt ddl.Stmt, m model.Model) (string, []any, error) {
	return stmt.Table, nil, nil
}
```

Two test functions:

- `TestMigrate_CreatesThreeTables` — asserts `len(execer.calls) == 3`.
- `TestMigrate_TableOrder` — asserts `execer.calls` equals
  `[]string{"business_hours", "holiday", "closure"}` in that order. The dummy
  compiler returns `stmt.Table`, so each recorded call is the table name. The
  names are the `Name:` fields of `BusinessHoursModel`, `HolidayModel` and
  `ClosureModel` in `model.go` — read them there, do not guess.

**Repo rule:** tests live in `tests/` — **except** this one. `migrate` is a
separate package and its test is the package's own unit test, exactly as
`device_manager/migrate/migrate_test.go` is. Place it in `migrate/`, not in
`tests/`. Do not move device_manager-style migrate tests into `tests/`.

### Stage 4 — existing tests must keep passing untouched

`tests/setup_test.go` builds the module over `webtyp.com/storage/mem`:

```go
m, err := businesscalendar.New(db, businesscalendar.Deps{IDs: ids, Publisher: pub})
```

`storage/mem` does not implement `ddl.Compiler`, so the deleted block was
already a no-op there. **No test file under `tests/` changes.** If you find
yourself editing one, you have changed behaviour you were not asked to change —
stop and re-read stage 2.

Run `gotest ./...` — everything green.

### Stage 5 — documentation

`README.md` currently says:

> `New` migrates the module's own schema (`business_hours`, `holiday`, `closure`)
> when the injected backend exposes the `ddl.Compiler` capability; against
> `storage/mem` (the module's own tests) it is a no-op.

Replace that paragraph with the new contract: `New` assumes the schema exists;
the deploying application calls `migrate.Migrate(conn, compiler)` once, at
deploy time. Update the "Quick start" block to show the migrate call before
`New`, and add a `migrate/` row to the "Key files" table.

In `docs/ARCHITECTURE.md`, find every statement that says construction creates
tables and correct it. Do **not** add a link to this plan file from any
permanent document — `docs/PLAN.md` is deleted when this lands.

## 5. Stages table

| # | Stage | Files | Acceptance |
|---|---|---|---|
| 1 | `migrate` package | `migrate/migrate.go` (new) | `migrate.Migrate(conn, compiler)` creates the three tables in order |
| 2 | Strip `New` | `module.go` | `grep -n "ddl" module.go` → empty |
| 3 | Unit test | `migrate/migrate_test.go` (new) | both tests pass |
| 4 | Regression | — | `gotest ./...` green, **zero** files changed under `tests/` |
| 5 | Docs | `README.md`, `docs/ARCHITECTURE.md` | no doc claims `New` migrates |

## 6. Acceptance criteria

- `grep -rn "webtyp.com/ddl" *.go` at the repository root → **empty** (the only
  `ddl` import in the repo is inside `migrate/`).
- `grep -rn "CreateTable" *.go` at the repository root → **empty**.
- `webtyp.com/ddl` still present in `go.mod` `require`.
- `gotest ./...` green.
- No file under `tests/` modified.
- `README.md` and `docs/ARCHITECTURE.md` describe `migrate.Migrate` as the only
  schema path; neither links to `docs/PLAN.md`.
