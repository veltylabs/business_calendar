package businesscalendar
import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/time"
)
func (p *DayStatus) Read(a crud.CurrentUser, filter *DayStatus, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	if filter == nil || filter.Date == 0 {
		return nil, &model.FieldError{Field: "date", Err: ErrDateRequired}
	}
	detail, err := m.GetDayDetail(time.Unix(filter.Date))
	if err != nil { return nil, err }
	list := DayStatusList{&DayStatus{
		Date:     filter.Date,
		Open:     detail.Limits.Open,
		OpenMin:  int64(detail.Limits.OpenMin),
		CloseMin: int64(detail.Limits.CloseMin),
		ClosedBy: string(detail.ClosedBy),
	}}
	return &list, nil
}
