← [Stage 2](PLAN_STAGE_2_MODULE_HELPERS.md) | Next → [Stage 4](PLAN_STAGE_4_READERS.md)

# Stage 3 — generate, then write the rules

## 3.1 Generate

Run `definitionc` in the repo root. It writes `definition_gen.go` (types `BusinessHours`, `Holiday`, `Closure`, `DayStatus`; their lists; the constants `BusinessHours_`, `Holiday_`, `Closure_`, `DayStatus_`; `const ModelName = "business_calendar"`, `ModelName()`, `Definition()`, `MountOperations()`) and one skeleton per action declared `AddMyBusinessRule`: `rule_business_hours_create.go`, `rule_business_hours_update.go`, `rule_holiday_create.go`, `rule_holiday_delete.go`, `rule_closure_create.go`, `rule_closure_delete.go`, `rule_day_detail_read.go`. If you deleted `const ModelName` from `definition.go` do it now (the generated one replaces it). Then repeat the `go get` line of Stage 1 and `go mod tidy`. **Keep the signatures the skeletons have; replace only the bodies.** Stage 2's files compile now.

## 3.2 The rule files

A rule that accepts ends `return crud.ApplyAndPublish, nil`; one that refuses ends `return 0, err`. The rules publish `EventCalendarChanged` with `m.publish(...)` at their end (`appointment_booking` subscribes to it): it runs before the commit, so a rollback after it can only cause one extra recompute in the subscribers, never a lost change. The kept methods of Stage 2 publish after their own commit.

**`rule_business_hours_create.go`** — one row per day: the unique index of `day_of_week` answers a second create as a conflict
```go
func (p *BusinessHours) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	normalized, err := normalizeHours(*p)
	if err != nil {
		return 0, err
	}
	p.OpenMin, p.CloseMin = normalized.OpenMin, normalized.CloseMin
	m.publish(CalendarKindBusinessHours, false, 0, 0)
	return crud.ApplyAndPublish, nil
}
```
**`rule_business_hours_update.go`** — only the listed columns change; the rule checks the row AS IT WILL BE (the old row with the sent columns applied) and writes itself the columns normalization changes that the caller did not send
```go
func (next *BusinessHours) Update(a crud.CurrentUser, tx crud.Tx, old *BusinessHours, fields []string, m *Module) (crud.RuleAnswer, error) {
	merged := *old
	for _, field := range fields {
		switch field {
		case BusinessHours_.DayOfWeek:
			merged.DayOfWeek = next.DayOfWeek
		case BusinessHours_.OpenMin:
			merged.OpenMin = next.OpenMin
		case BusinessHours_.CloseMin:
			merged.CloseMin = next.CloseMin
		case BusinessHours_.IsOpen:
			merged.IsOpen = next.IsOpen
		case BusinessHours_.Notes:
			merged.Notes = next.Notes
		default:
			return 0, &model.FieldError{Field: field, Err: ErrFieldNotEditable}
		}
	}
	normalized, err := normalizeHours(merged)
	if err != nil {
		return 0, err
	}
	next.OpenMin, next.CloseMin = normalized.OpenMin, normalized.CloseMin // crud writes these if they were sent
	var derived []string
	if !containsString(fields, BusinessHours_.OpenMin) && normalized.OpenMin != old.OpenMin {
		derived = append(derived, BusinessHours_.OpenMin)
	}
	if !containsString(fields, BusinessHours_.CloseMin) && normalized.CloseMin != old.CloseMin {
		derived = append(derived, BusinessHours_.CloseMin)
	}
	if len(derived) > 0 {
		if err := tx.UpdateFields(&normalized, old.Id, derived); err != nil {
			return 0, err
		}
	}
	m.publish(CalendarKindBusinessHours, hoursCloseTime(*old, normalized), 0, 0)
	return crud.ApplyAndPublish, nil
}
```
**`rule_holiday_create.go`**
```go
func (p *Holiday) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if err := checkNewHoliday(tx, p); err != nil {
		return 0, err
	}
	m.publish(CalendarKindHoliday, true, p.SpecificDate, p.SpecificDate)
	return crud.ApplyAndPublish, nil
}
```
**`rule_holiday_delete.go`** (`p` is the stored row: removing a holiday reopens its day)
```go
func (p *Holiday) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	m.publish(CalendarKindHoliday, false, p.SpecificDate, p.SpecificDate)
	return crud.ApplyAndPublish, nil
}
```
**`rule_closure_create.go`** and **`rule_closure_delete.go`**: the same two bodies with `*Closure`, `checkNewClosure` and `CalendarKindClosure`.

**`rule_day_detail_read.go`** — the read model; `date` arrives in `filter`
```go
func (p *DayStatus) Read(a crud.CurrentUser, filter *DayStatus, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	if filter == nil || filter.Date == 0 {
		return nil, &model.FieldError{Field: DayStatus_.Date, Err: ErrDateRequired}
	}
	detail, err := m.GetDayDetail(time.Unix(filter.Date))
	if err != nil {
		return nil, err
	}
	list := DayStatusList{&DayStatus{
		Date:     filter.Date,
		Open:     detail.Limits.Open,
		OpenMin:  int64(detail.Limits.OpenMin),
		CloseMin: int64(detail.Limits.CloseMin),
		ClosedBy: string(detail.ClosedBy),
	}}
	return &list, nil
}
```
A caller (and a test) passes the date as `&crud.ReadArgs{Filter: &businesscalendar.DayStatus{Date: day}}`. If `detail.Limits.OpenMin` is not an integer type, convert with the conversion the compiler asks for (`int64(...)` of its underlying integer).

## 3.3 Done when

`go vet ./...` compiles (the `hours.go` readers still use the removed `m.db` until Stage 4: if the package does not compile yet because of them, do Stage 4 first for that file only). Commit, push, and **open the pull request now** (title: the `PLAN:` line).
