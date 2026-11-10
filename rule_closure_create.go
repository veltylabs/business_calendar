package businesscalendar

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

// Create es la regla de negocio de crear closure: valida lo que solo tu dominio sabe y responde si se guarda y si se avisa.
// crud ya validó el esquema, los campos solo del servidor, los límites y el permiso antes de llamarte.
func (p *Closure) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	var found bool
	err := tx.ReadAll(func() model.Model { return new(Closure) }, crud.Query{Conditions: []storage.Condition{storage.Eq("specific_date", p.SpecificDate)}}, func(m model.Model) { found = true })
	if err != nil { return 0, err }
	if found { return 0, ErrDuplicateDate }

	m.publish(CalendarKindClosure, true, p.SpecificDate, p.SpecificDate)
	return crud.ApplyAndPublish, nil
}
