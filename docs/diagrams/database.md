# business_calendar — Database Schema

Three tables, all single-tenant (no `tenant_id` — see `ARCHITECTURE.md` §2).

```mermaid
erDiagram
    BUSINESS_HOURS {
        text id PK
        int day_of_week UK "0=Sunday..6=Saturday"
        int open_min "minutes from midnight"
        int close_min "minutes from midnight"
        bool is_open
        text notes
        int updated_at "unix seconds"
    }

    HOLIDAY {
        text id PK
        int specific_date "midnight UTC, seconds"
        text name
        text notes
        int updated_at "unix seconds"
    }

    CLOSURE {
        text id PK
        int specific_date "midnight UTC, seconds"
        text reason
        int updated_at "unix seconds"
    }
```

Notes:

- `business_hours.day_of_week` is UNIQUE — the module never writes more than
  seven rows. `open_min`/`close_min` are minutes from midnight (`0..1439`);
  a closed day normalizes them to `0`.
- `holiday.specific_date` and `closure.specific_date` are unique **by
  convention enforced in the service** (`ErrDuplicateDate`) — there is no DB
  unique constraint on them, so the duplicate guard lives in `AddHoliday` /
  `AddClosure`.
- `Holiday` and `Closure` are separate tables because their **origin** is
  domain-meaningful (`ClosedReason`), not a magic `type` string.
