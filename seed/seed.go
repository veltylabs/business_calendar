package seed

import (
	businesscalendar "github.com/veltylabs/business_calendar"
	tinytime "webtyp.com/time"
)

type Data struct {
	Hours    []businesscalendar.BusinessHours
	Holidays []businesscalendar.Holiday
}

type holidaySeed struct {
	dateStr string
	name    string
}

var chileHolidays2026 = []holidaySeed{
	{"2026-01-01", "Año Nuevo"},
	{"2026-05-01", "Día del Trabajador"},
	{"2026-06-29", "San Pedro y San Pablo"},
	{"2026-07-16", "Virgen del Carmen"},
	{"2026-08-15", "Asunción de la Virgen"},
	{"2026-09-18", "Fiestas Patrias"},
	{"2026-09-19", "Glorias del Ejército"},
	{"2026-10-12", "Encuentro de Dos Mundos"},
	{"2026-10-31", "Día de las Iglesias Evangélicas"},
	{"2026-11-01", "Día de Todos los Santos"},
	{"2026-12-08", "Inmaculada Concepción"},
	{"2026-12-25", "Navidad"},
}

func Load(m *businesscalendar.Module) (Data, error) {
	var data Data

	// 1. Business hours seed
	hoursToSeed := []businesscalendar.BusinessHours{
		{DayOfWeek: 1, OpenMin: 480, CloseMin: 1080, IsOpen: true},
		{DayOfWeek: 2, OpenMin: 480, CloseMin: 1080, IsOpen: true},
		{DayOfWeek: 3, OpenMin: 480, CloseMin: 1080, IsOpen: true},
		{DayOfWeek: 4, OpenMin: 480, CloseMin: 1080, IsOpen: true},
		{DayOfWeek: 5, OpenMin: 480, CloseMin: 1080, IsOpen: true},
		{DayOfWeek: 6, OpenMin: 540, CloseMin: 780, IsOpen: true},
		{DayOfWeek: 0, OpenMin: 0, CloseMin: 0, IsOpen: false},
	}

	for _, h := range hoursToSeed {
		if err := m.UpsertBusinessHours(h); err != nil {
			return Data{}, err
		}
	}

	loadedHours, err := m.ListBusinessHours()
	if err != nil {
		return Data{}, err
	}
	data.Hours = loadedHours

	// 2. Holidays seed
	for _, hs := range chileHolidays2026 {
		nano, err := tinytime.ParseDate(hs.dateStr)
		if err != nil {
			return Data{}, err
		}
		sec := nano / 1_000_000_000
		h := businesscalendar.Holiday{
			SpecificDate: sec,
			Name:         hs.name,
		}
		if err := m.AddHoliday(h); err != nil {
			return Data{}, err
		}
	}

	loadedHolidays, err := m.ListHolidays(0, 0)
	if err != nil {
		return Data{}, err
	}
	data.Holidays = loadedHolidays

	return data, nil
}
