package businesscalendar
import "webtyp.com/crud"
func (p *Closure) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	m.publish(CalendarKindClosure, false, p.SpecificDate, p.SpecificDate)
	return crud.ApplyAndPublish, nil
}
