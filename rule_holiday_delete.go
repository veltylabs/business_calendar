package businesscalendar
import "webtyp.com/crud"
func (p *Holiday) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	m.publish(CalendarKindHoliday, false, p.SpecificDate, p.SpecificDate)
	return crud.ApplyAndPublish, nil
}
