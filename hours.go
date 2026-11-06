package businesscalendar

import (
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/time"
)

// getWeeklyBounds representa los pasos 3–4 de GetDayDetail: solo la plantilla semanal. La
// única implementación compartida por GetDayDetail y WeekdayLimits.
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

// GetDayDetail resuelve el estado del establecimiento para una fecha, INCLUYENDO
// por qué un día cerrado está cerrado. Esta es la única ruta de resolución — DayLimits
// la envuelve y simplemente descarta ClosedBy para un consumidor a nivel Go que nunca
// lo pidió.
//
// Orden de resolución, en esta precedencia exacta:
//
//  1. Un Closure en esa fecha → {Open: false, ClosedBy: ClosedLocal}.
//  2. Un Holiday en esa fecha → {Open: false, ClosedBy: ClosedHoliday}.
//  3. BusinessHours para ese día de la semana con is_open == false →
//     {Open: false, ClosedBy: ClosedWeekly}.
//  4. De lo contrario {Open: true, OpenMin, CloseMin}.
//
// Un cierre local supera a un feriado porque es la decisión más específica, más
// reciente, y porque un feriado en el que el establecimiento eligió trabajar
// no debe reabrirse al eliminar el cierre. Un día de la semana sin fila
// business_hours en absoluto está cerrado (cerrado por defecto — la ausencia de un
// horario nunca significa abierto).
// Ejemplo: detail, err := cal.GetDayDetail(date)
func (m *Module) GetDayDetail(date time.Unix) (DayDetail, error) {
	var c Closure
	_, err := ReadOneClosure(m.db.Query(&c).Where(Closure_.SpecificDate).Eq(int64(date)), &c)
	if err == nil {
		return DayDetail{ClosedBy: ClosedLocal}, nil
	}
	if !orm.IsNotFound(err) {
		return DayDetail{}, err
	}

	var h Holiday
	_, err = ReadOneHoliday(m.db.Query(&h).Where(Holiday_.SpecificDate).Eq(int64(date)), &h)
	if err == nil {
		return DayDetail{ClosedBy: ClosedHoliday}, nil
	}
	if !orm.IsNotFound(err) {
		return DayDetail{}, err
	}

	return m.getWeeklyBounds(time.WeekdayUTC(date))
}

// DayLimits es el puerto de lectura neutral: solo los límites neutrales, sin ninguna razón
// adjunta. Satisface estructuralmente cualquier interfaz propia con forma de OpeningHours
// de un hermano.
// Ejemplo: limits, err := cal.DayLimits(date)
func (m *Module) DayLimits(date time.Unix) (time.DayLimits, error) {
	d, err := m.GetDayDetail(date)
	return d.Limits, err
}

// WeekdayLimits devuelve las horas regulares de un día de la semana (0 = domingo … 6 =
// sábado) — la plantilla semanal, sin feriados o cierres locales.
// Ejemplo: limits, err := cal.WeekdayLimits(1)
func (m *Module) WeekdayLimits(dayOfWeek int) (time.DayLimits, error) {
	if dayOfWeek < 0 || dayOfWeek > 6 {
		return time.DayLimits{}, ErrInvalidWeekday
	}
	d, err := m.getWeeklyBounds(dayOfWeek)
	return d.Limits, err
}

// ClosedDates lista los feriados y cierres entre from y to (medianoches UTC, ambos incluidos), como fechas
// "YYYY-MM-DD", en orden ascendente. Satisface OpeningHours de appointment_booking.
// Ejemplo: holidays, closures, err := m.ClosedDates(from, to)
func (m *Module) ClosedDates(from, to time.Unix) (holidays, closures []string, err error) {
	hs, err := m.ListHolidays(int64(from), int64(to))
	if err != nil {
		return nil, nil, err
	}
	cs, err := m.ListClosures(int64(from), int64(to))
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

// dateText es la fecha "YYYY-MM-DD" de una clave de día (medianoche UTC).
func dateText(day time.Unix) string {
	y, mo, d, _, _, _ := time.CivilUTC(day)
	return fmt.Sprintf("%04d-%02d-%02d", y, mo, d)
}
