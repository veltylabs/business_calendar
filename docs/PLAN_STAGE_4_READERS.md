← [Stage 3](PLAN_STAGE_3_RULES.md) | Next → [Stage 5](PLAN_STAGE_5_BROWSER.md)

# Stage 4 — `hours.go` on the store

Only two functions of `hours.go` touch the database; everything else (`getWeeklyBounds`, `DayLimits`, `WeekdayLimits`, `ClosedDates`, `dateText`) is unchanged. `bankdays.go` and `convert.go` are unchanged. Edit by hand.

1. Imports of `hours.go` become `webtyp.com/fmt`, `webtyp.com/model`, `webtyp.com/storage`, `webtyp.com/time` (drop `webtyp.com/orm`).
2. Add `businessHoursByDay` (it left `module.go` in Stage 2), written exactly:

```go
// businessHoursByDay reads the weekly rule of a day of the week; a day without a row is ErrNotFound.
func (m *Module) businessHoursByDay(day int) (BusinessHours, error) {
	row, found, err := firstRow(m.store, func() model.Model { return &BusinessHours{} },
		storage.Eq(BusinessHours_.DayOfWeek, int64(day)))
	if err != nil {
		return BusinessHours{}, err
	}
	if !found {
		return BusinessHours{}, ErrNotFound
	}
	return *row.(*BusinessHours), nil
}
```
3. Replace the body of `GetDayDetail` (its doc comment stays, word for word) with exactly:

```go
func (m *Module) GetDayDetail(date time.Unix) (DayDetail, error) {
	if _, found, err := firstRow(m.store, func() model.Model { return &Closure{} },
		storage.Eq(Closure_.SpecificDate, int64(date))); err != nil {
		return DayDetail{}, err
	} else if found {
		return DayDetail{ClosedBy: ClosedLocal}, nil
	}
	if _, found, err := firstRow(m.store, func() model.Model { return &Holiday{} },
		storage.Eq(Holiday_.SpecificDate, int64(date))); err != nil {
		return DayDetail{}, err
	} else if found {
		return DayDetail{ClosedBy: ClosedHoliday}, nil
	}
	return m.getWeeklyBounds(time.WeekdayUTC(date))
}
```
Precedence stays: closure, then holiday, then the weekly template. `seed/seed.go` is unchanged: it calls the kept methods `UpsertBusinessHours`, `AddHoliday`, `ListBusinessHours`, `ListHolidays`.

`gotest` compiles the package now. Several old tests still call the removed `ops.go` handlers: Stage 6 deletes or rewrites them. Commit and push.
