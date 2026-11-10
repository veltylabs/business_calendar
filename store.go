package businesscalendar

import "webtyp.com/crud"

// StoreBusinessHours is the store of business_hours.
func (m *Module) StoreBusinessHours() crud.Store { return m.store }

// StoreHoliday is the store of holiday.
func (m *Module) StoreHoliday() crud.Store { return m.store }

// StoreClosure is the store of closure.
func (m *Module) StoreClosure() crud.Store { return m.store }
