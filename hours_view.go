package businesscalendar

import (
	"webtyp.com/model"
)

var weekdays = [7]string{
	"Sunday",
	"Monday",
	"Tuesday",
	"Wednesday",
	"Thursday",
	"Friday",
	"Saturday",
}

type businessHoursView struct {
	row *BusinessHours
}

func (v *businessHoursView) Schema() []model.Field { return nil }
func (v *businessHoursView) Pointers() []any       { return nil }
func (v *businessHoursView) IsNil() bool           { return v == nil || v.row == nil }

func (v *businessHoursView) EncodeFields(w model.FieldWriter) {
	if v.row == nil {
		return
	}
	v.row.EncodeFields(w)

	if v.row.DayOfWeek >= 0 && v.row.DayOfWeek < 7 {
		w.String("day", weekdays[v.row.DayOfWeek])
	}
	if v.row.IsOpen {
		w.String("opens", v.row.OpenMin)
		w.String("closes", v.row.CloseMin)
	}
}

func (v *businessHoursView) DecodeFields(_ model.FieldReader) {}

type businessHoursViewList []*businessHoursView

func (s *businessHoursViewList) Schema() []model.Field  { return nil }
func (s *businessHoursViewList) Pointers() []any        { return nil }
func (s *businessHoursViewList) Len() int               { return len(*s) }
func (s *businessHoursViewList) At(i int) model.Fielder { return (*s)[i] }
func (s *businessHoursViewList) Append() model.Fielder {
	v := &businessHoursView{}
	*s = append(*s, v)
	return v
}
func (s *businessHoursViewList) IsNil() bool                      { return s == nil }
func (s *businessHoursViewList) EncodeFields(_ model.FieldWriter) {}
func (s *businessHoursViewList) DecodeFields(_ model.FieldReader) {}
