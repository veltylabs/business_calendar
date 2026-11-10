package businesscalendar
import "webtyp.com/crud"
func (p *Holiday) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if err := checkNewHoliday(tx, p); err != nil { return 0, err }
	m.publish(CalendarKindHoliday, true, p.SpecificDate, p.SpecificDate)
	return crud.ApplyAndPublish, nil
}
