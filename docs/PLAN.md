---
PLAN: "refactor!: business_calendar on the crud-by-declaration framework — Allow and Rules in definition.go, persistence by crud, only domain rules in rule files"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 15130981108390694134
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work. Commit after every stage.
> Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.
> **Push early:** after EVERY stage run `git push origin HEAD`; after Stage 3 open the pull request and keep pushing to it.
> **Never ask a yes/no question about progress** («am I on the right track?», «shall I proceed?», «is there anything else?»): the answer is always yes. Ask only when a compile error or a signature in this plan is wrong, and name the symbol.
> Never edit `definition_gen.go`. Never write `go.mod`/`go.sum` lines by hand: only `go get` and `go mod tidy` do. No scratch files and no sed/regex/python scripts over the source: edit by hand, file by file, and run `gotest` after each file.

# Plan: `business_calendar` on the crud-by-declaration framework

Three earlier sessions of this plan stalled on questions it did not answer, and an earlier note told them to delete the writer methods that `mjosefa-cms` still calls (`cal.AddHoliday` in `migrate/pastdb/holidays.go`): that note was wrong. This version keeps every exported method and is split into stages, one file each. **Every answer they needed is written inside the stage that needs it.**

After: three tables (`BusinessHours`, `Holiday`, `Closure`) mounted by `crud` with rules; one flat read model (`day_detail`); the kept methods (`UpsertBusinessHours`, `AddHoliday`, `RemoveHoliday`, `AddClosure`, `RemoveClosure` and all the readers) now read and write through `m.store`; the event `EventCalendarChanged` (`appointment_booking` subscribes to it) keeps being published on every write.

Reference module, already migrated (copy its shape): https://github.com/veltylabs/patient_directory/blob/main — `definition.go`, `rule_patient_create.go`, `rule_patient_update.go`, `store.go`, `import.go`, `module/server.go`, `seed/demo.go`, `tests/setup_test.go`, `tests/crud_test.go`, `tests/demo_test.go`.

## Stages

| # | File | What | Files touched |
|---|---|---|---|
| 1 | [PLAN_STAGE_1_TOOLS_DEFINITION.md](PLAN_STAGE_1_TOOLS_DEFINITION.md) | tools, `definition.go` | `definition.go`, `go.mod`, `go.sum` |
| 2 | [PLAN_STAGE_2_MODULE_HELPERS.md](PLAN_STAGE_2_MODULE_HELPERS.md) | `module.go` on `m.store`, `rows.go`, `checks.go` | `module.go`, `store.go`, `rows.go`, `checks.go`, `ops.go` (deleted), `hours_view.go` (deleted) |
| 3 | [PLAN_STAGE_3_RULES.md](PLAN_STAGE_3_RULES.md) | generate; the rule files | `rule_*.go` |
| 4 | [PLAN_STAGE_4_READERS.md](PLAN_STAGE_4_READERS.md) | `hours.go` on the store | `hours.go` |
| 5 | [PLAN_STAGE_5_BROWSER.md](PLAN_STAGE_5_BROWSER.md) | `view.go`, `module/`, demo, `migrate/` | `view.go`, `module/*.go`, `seed/demo.go`, `migrate/migrate.go`, `web/` (deleted) |
| 6 | [PLAN_STAGE_6_TESTS_DOCS.md](PLAN_STAGE_6_TESTS_DOCS.md) | tests, `docs/TESTS.md`, README | `tests/*.go`, `docs/TESTS.md`, `README.md`, `docs/ARCHITECTURE.md` |

## Operation mapping (every operation of today has exactly one row)

`<entity>.<verb>` is `router.Operation{Module: ModelName, Entity: "<entity>", Action: model.<Verb>}`; `ModelName` is the constant `definitionc` writes (`"business_calendar"`): never a literal.

| Today | After |
|---|---|
| `list_business_hours` | `business_hours.read` |
| `upsert_business_hours` | `business_hours.create` / `business_hours.update` |
| `get_day_limits` | read model `day_detail` with filter `{date}` |
| `list_holidays` | `holiday.read` |
| `add_holiday` / `remove_holiday` | `holiday.create` / `holiday.delete` |
| `list_closures` | `closure.read` |
| `add_closure` / `remove_closure` | `closure.create` / `closure.delete` |

## Delivery check

1. `gotest` ends green on EVERY destination (`wasm` and `browser` included) and «tests index» ✅.
2. `langc sync .` reports zero untranslated keys for `es` and zero reuse warnings.
3. `grep -rn "reg.Operation\|view.MountSave\|view.MountDelete\|router.Context\|rpc.WriteError\|m\.db\." --include=*.go . | grep -v "_test.go\|^./ui/\|^./definition_gen.go"` → only `m.db` inside `New` (it builds the store).
4. `ls *.py *.sh commit_message.txt 2>/dev/null` → empty; `grep -c XXXX go.sum` → 0; `test ! -e web`.
5. `git diff --stat main` lists only the files of the stages table plus badge files `gotest` rewrites; `module/` is still there.
