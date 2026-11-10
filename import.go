package businesscalendar

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/time"
	"webtyp.com/storage"
)

// ImportBusinessHours creates or updates a BusinessHours record during seeding or import.
func (m *Module) ImportBusinessHours(p BusinessHours) error {
	if p.Id == "" { p.Id = m.ids.NewID() }
	p.UpdatedAt = time.NowNano()
	if err := model.ValidateFields(model.ActionCreate, &p); err != nil {
		return err
	}

	return m.store.Atomic(func(tx crud.Tx) error {
		var existing BusinessHours
		var found bool
		err := tx.ReadAll(func() model.Model { return new(BusinessHours) }, crud.Query{Conditions: []storage.Condition{storage.Eq("day_of_week", p.DayOfWeek)}}, func(row model.Model) {
			existing = *row.(*BusinessHours)
			found = true
		})
		if err != nil {
			return err
		}

		if found {
			p.Id = existing.Id
			// Not importing fields strictly, we can just UpdateFields
			return tx.UpdateFields(&p, p.Id, []string{"open_min", "close_min", "is_open", "notes", "updated_at"})
		}

		if p.Id == "" {
			p.Id = m.ids.NewID()
		}
		return tx.Create(&p)
	})
}

// ImportHoliday creates or updates a Holiday record during seeding or import.
func (m *Module) ImportHoliday(p Holiday) error {
	if p.Id == "" { p.Id = m.ids.NewID() }
	p.UpdatedAt = time.NowNano()
	if err := model.ValidateFields(model.ActionCreate, &p); err != nil {
		return err
	}

	return m.store.Atomic(func(tx crud.Tx) error {
		var existing Holiday
		var found bool
		err := tx.ReadAll(func() model.Model { return new(Holiday) }, crud.Query{Conditions: []storage.Condition{storage.Eq("specific_date", p.SpecificDate)}}, func(row model.Model) {
			existing = *row.(*Holiday)
			found = true
		})
		if err != nil {
			return err
		}

		if found {
			p.Id = existing.Id
			return tx.UpdateFields(&p, p.Id, []string{"name", "notes", "updated_at"})
		}

		if p.Id == "" {
			p.Id = m.ids.NewID()
		}
		return tx.Create(&p)
	})
}
