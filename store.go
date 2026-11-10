package businesscalendar

import "webtyp.com/crud"

func (m *Module) StoreBusinessHours() crud.Store { return m.store }
func (m *Module) StoreHoliday() crud.Store { return m.store }
func (m *Module) StoreClosure() crud.Store { return m.store }
