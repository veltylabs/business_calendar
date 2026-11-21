package businesscalendar

import (
	"webtyp.com/fmt"
	"webtyp.com/lang"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/time"
	"webtyp.com/view"
)

var weekdayNames = [7]lang.Text{
	"Sunday",
	"Monday",
	"Tuesday",
	"Wednesday",
	"Thursday",
	"Friday",
	"Saturday",
}

func formatUTCDate(unixSec int64) string {
	if unixSec <= 0 {
		return ""
	}
	y, mo, d, _, _, _ := time.CivilUTC(time.Unix(unixSec))
	return fmt.Sprintf("%04d-%02d-%02d", y, mo, d)
}

func (m *BusinessHours) Item() model.Item {
	label := fmt.Convert(m.DayOfWeek).String()
	if m.DayOfWeek >= 0 && m.DayOfWeek < int64(len(weekdayNames)) {
		label = lang.Translate(weekdayNames[m.DayOfWeek]).String()
	}
	desc := lang.Translate("Closed").String()
	if m.IsOpen {
		if m.OpenMin != "" && m.CloseMin != "" {
			desc = m.OpenMin + "–" + m.CloseMin
		} else {
			desc = lang.Translate("Open").String()
		}
	}
	return model.Item{ID: m.Id, Label: label, Description: desc}
}

func (m *Holiday) Item() model.Item {
	return model.Item{ID: m.Id, Label: m.Name, Description: formatUTCDate(m.SpecificDate)}
}

func (m *Closure) Item() model.Item {
	return model.Item{ID: m.Id, Label: m.Reason, Description: formatUTCDate(m.SpecificDate)}
}

func NewBusinessHoursView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(view.NewRecords().Caller(caller),
		view.Ops{Module: ModelName, Page: view.Page{First: view.All}},
		func() model.TableSlice { return &BusinessHoursList{} })
	allOpts := append([]view.Option{view.WithTitle(lang.Translate("Business hours").String())}, opts...)
	return view.New(b, &BusinessHours{}, allOpts...)
}

func NewHolidaysView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(view.NewRecords().Caller(caller),
		view.Ops{Module: ModelName, Page: view.Page{First: view.All}},
		func() model.TableSlice { return &HolidayList{} })
	allOpts := append([]view.Option{view.WithTitle(lang.Translate("Holidays").String())}, opts...)
	return view.New(b, &Holiday{}, allOpts...)
}

func NewClosuresView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(view.NewRecords().Caller(caller),
		view.Ops{Module: ModelName, Page: view.Page{First: view.All}},
		func() model.TableSlice { return &ClosureList{} })
	allOpts := append([]view.Option{view.WithTitle(lang.Translate("Closures").String())}, opts...)
	return view.New(b, &Closure{}, allOpts...)
}
