package businesscalendar

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

// Create es la regla de negocio de crear business_hours: valida lo que solo tu dominio sabe y responde si se guarda y si se avisa.
// crud ya validó el esquema, los campos solo del servidor, los límites y el permiso antes de llamarte.
func (p *BusinessHours) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if p.DayOfWeek < 0 || p.DayOfWeek > 6 {
		return 0, &model.FieldError{Field: "day_of_week", Err: ErrInvalidDay}
	}
	if p.IsOpen {
		openMin, err1 := parseClock(p.OpenMin)
		closeMin, err2 := parseClock(p.CloseMin)
		if err1 != nil || err2 != nil || openMin >= closeMin {
			return 0, &model.FieldError{Field: "open_min", Err: ErrInvalidMinutes}
		}
	} else {
		p.OpenMin, p.CloseMin = "00:00", "00:00"
	}

	// Check for unique day_of_week
	var found bool
	err := tx.ReadAll(func() model.Model { return new(BusinessHours) }, crud.Query{Conditions: []storage.Condition{storage.Eq("day_of_week", p.DayOfWeek)}}, func(m model.Model) { found = true })
	if err != nil { return 0, err }
	if found { return 0, ErrDuplicateDate }

	m.publish(CalendarKindBusinessHours, false, 0, 0)
	return crud.ApplyAndPublish, nil
}
