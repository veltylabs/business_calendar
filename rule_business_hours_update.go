package businesscalendar

import (
	"webtyp.com/crud"
	"webtyp.com/model"
)

// Update es la regla de negocio de actualizar business_hours: old es la fila guardada y el receptor trae solo los campos enviados (fields).
// crud ya validó el esquema, los campos solo del servidor, los límites y el permiso antes de llamarte.
func (next *BusinessHours) Update(a crud.CurrentUser, tx crud.Tx, old *BusinessHours, fields []string, m *Module) (crud.RuleAnswer, error) {
	if next.DayOfWeek < 0 || next.DayOfWeek > 6 {
		return 0, &model.FieldError{Field: "day_of_week", Err: ErrInvalidDay}
	}
	if next.IsOpen {
		openMin, err1 := parseClock(next.OpenMin)
		closeMin, err2 := parseClock(next.CloseMin)
		if err1 != nil || err2 != nil || openMin >= closeMin {
			return 0, &model.FieldError{Field: "open_min", Err: ErrInvalidMinutes}
		}
	} else {
		next.OpenMin, next.CloseMin = "00:00", "00:00"
	}
	closed := hoursCloseTime(*old, *next)
	m.publish(CalendarKindBusinessHours, closed, 0, 0)
	return crud.ApplyAndPublish, nil
}
