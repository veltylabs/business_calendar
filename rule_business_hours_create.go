package businesscalendar
import "webtyp.com/crud"
func (p *BusinessHours) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	normalized, err := normalizeHours(*p)
	if err != nil { return 0, err }
	p.OpenMin, p.CloseMin = normalized.OpenMin, normalized.CloseMin
	m.publish(CalendarKindBusinessHours, false, 0, 0)
	return crud.ApplyAndPublish, nil
}
