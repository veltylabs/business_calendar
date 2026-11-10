package businesscalendar

import (
	"webtyp.com/crud"
	"webtyp.com/time"
	"webtyp.com/model"
)

// Read responde la lectura de day_detail, que no es una tabla: tú produces las filas.
// crud ya validó los argumentos, aplicó el tope de página y comprobó el permiso antes de llamarte.
func (p *DayDetail) Read(a crud.CurrentUser, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	detail, err := m.GetDayDetail(time.Unix(p.Date))
	if err != nil { return nil, err }
	return &DayDetailList{&DayDetail{Date: p.Date, Open: detail.Open, OpenMin: detail.OpenMin, CloseMin: detail.CloseMin, ClosedBy: detail.ClosedBy}}, nil
}
