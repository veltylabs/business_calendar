package businesscalendar

import (
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/storage"
	"webtyp.com/time"
)

func (m *Module) getWeeklyBounds(dayOfWeek int) (DayDetail, error) {
	bh, err := m.businessHoursByDay(dayOfWeek)
	if err != nil {
		if model.KindOf(err) == model.ErrorNotFound {
			return DayDetail{ClosedBy: ClosedWeekly}, nil
		}
		return DayDetail{}, err
	}
	if !bh.IsOpen {
		return DayDetail{ClosedBy: ClosedWeekly}, nil
	}
	open, err := storedClock("open_min", bh.OpenMin)
	if err != nil {
		return DayDetail{}, err
	}
	closeAt, err := storedClock("close_min", bh.CloseMin)
	if err != nil {
		return DayDetail{}, err
	}
	return DayDetail{Limits: time.DayLimits{Open: true, OpenMin: open, CloseMin: closeAt}}, nil
}

func (m *Module) GetDayDetail(date time.Unix) (DayDetail, error) {
	if _, found, err := firstRow(m.store, func() model.Model { return &Closure{} },
		storage.Eq("specific_date", int64(date))); err != nil {
		return DayDetail{}, err
	} else if found {
		return DayDetail{ClosedBy: ClosedLocal}, nil
	}
	if _, found, err := firstRow(m.store, func() model.Model { return &Holiday{} },
		storage.Eq("specific_date", int64(date))); err != nil {
		return DayDetail{}, err
	} else if found {
		return DayDetail{ClosedBy: ClosedHoliday}, nil
	}
	return m.getWeeklyBounds(time.WeekdayUTC(date))
}

func (m *Module) DayLimits(date time.Unix) (time.DayLimits, error) {
	d, err := m.GetDayDetail(date)
	return d.Limits, err
}

func (m *Module) WeekdayLimits(dayOfWeek int) (time.DayLimits, error) {
	if dayOfWeek < 0 || dayOfWeek > 6 {
		return time.DayLimits{}, ErrInvalidWeekday
	}
	d, err := m.getWeeklyBounds(dayOfWeek)
	return d.Limits, err
}

func (m *Module) ClosedDates(from, to time.Unix) (holidays, closures []string, err error) {
	hs, err := m.ListHolidays(int64(from), int64(to), "", "", 0)
	if err != nil {
		return nil, nil, err
	}
	cs, err := m.ListClosures(int64(from), int64(to), "", "", 0)
	if err != nil {
		return nil, nil, err
	}
	for _, h := range hs {
		holidays = append(holidays, dateText(time.Unix(h.SpecificDate)))
	}
	for _, c := range cs {
		closures = append(closures, dateText(time.Unix(c.SpecificDate)))
	}
	return holidays, closures, nil
}

func dateText(day time.Unix) string {
	y, mo, d, _, _, _ := time.CivilUTC(day)
	return fmt.Sprintf("%04d-%02d-%02d", y, mo, d)
}

func (m *Module) businessHoursByDay(day int) (BusinessHours, error) {
	row, found, err := firstRow(m.store, func() model.Model { return &BusinessHours{} },
		storage.Eq("day_of_week", int64(day)))
	if err != nil {
		return BusinessHours{}, err
	}
	if !found {
		return BusinessHours{}, ErrNotFound
	}
	return *row.(*BusinessHours), nil
}
