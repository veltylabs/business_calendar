package businesscalendar
import "webtyp.com/crud"
func (p *Closure) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if err := checkNewClosure(tx, p); err != nil { return 0, err }
	m.publish(CalendarKindClosure, true, p.SpecificDate, p.SpecificDate)
	return crud.ApplyAndPublish, nil
}
