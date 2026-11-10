package businesscalendar
import (
	"webtyp.com/crud"
	"webtyp.com/model"
)
func (next *BusinessHours) Update(a crud.CurrentUser, tx crud.Tx, old *BusinessHours, fields []string, m *Module) (crud.RuleAnswer, error) {
	merged := *old
	for _, field := range fields {
		switch field {
		case "day_of_week": merged.DayOfWeek = next.DayOfWeek
		case "open_min": merged.OpenMin = next.OpenMin
		case "close_min": merged.CloseMin = next.CloseMin
		case "is_open": merged.IsOpen = next.IsOpen
		case "notes": merged.Notes = next.Notes
		default: return 0, &model.FieldError{Field: field, Err: ErrFieldNotEditable}
		}
	}
	normalized, err := normalizeHours(merged)
	if err != nil { return 0, err }
	next.OpenMin, next.CloseMin = normalized.OpenMin, normalized.CloseMin
	var derived []string
	if !containsString(fields, "open_min") && normalized.OpenMin != old.OpenMin { derived = append(derived, "open_min") }
	if !containsString(fields, "close_min") && normalized.CloseMin != old.CloseMin { derived = append(derived, "close_min") }
	if len(derived) > 0 {
		if err := tx.UpdateFields(&normalized, old.Id, derived); err != nil { return 0, err }
	}
	m.publish(CalendarKindBusinessHours, hoursCloseTime(*old, normalized), 0, 0)
	return crud.ApplyAndPublish, nil
}
