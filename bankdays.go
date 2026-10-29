package businesscalendar

import (
	"webtyp.com/model"
	tinytime "webtyp.com/time"
)

// ErrNegativeDays: BankDayAfter con días negativos.
const ErrNegativeDays = model.Invalid("business_calendar: days must be zero or more")

// BankDayAfter devuelve la medianoche UTC del día hábil bancario que está days días hábiles después de from.
// Día hábil bancario: lunes a viernes que no es feriado legal (Holiday). El horario y los cierres propios del
// establecimiento no cuentan. days == 0 devuelve el mismo día si es hábil, si no el siguiente hábil.
// Ejemplo: deposito, err := cal.BankDayAfter(cobradoEl, 2) // cobro del viernes con abono a 2 días → martes
func (m *Module) BankDayAfter(from tinytime.Unix, days int) (tinytime.Unix, error) {
	if days < 0 {
		return 0, ErrNegativeDays
	}
	day := tinytime.MidnightUTC(from)
	// Feriados de un margen amplio: days hábiles nunca ocupan más de days*2+14 días corridos.
	holidays, err := m.ListHolidays(int64(day), int64(day+tinytime.Unix(days*2+14)*tinytime.Day))
	if err != nil {
		return 0, err
	}
	isBankDay := func(d tinytime.Unix) bool {
		wd := tinytime.Weekday(d)
		if wd == 0 || wd == 6 {
			return false
		}
		for _, h := range holidays {
			if tinytime.Unix(h.SpecificDate) == d {
				return false
			}
		}
		return true
	}
	for !isBankDay(day) {
		day += tinytime.Day
	}
	for left := days; left > 0; {
		day += tinytime.Day
		if isBankDay(day) {
			left--
		}
	}
	return day, nil
}
