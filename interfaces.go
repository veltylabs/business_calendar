package businesscalendar

import (
	"webtyp.com/model"
	"webtyp.com/orm"
	tinytime "webtyp.com/time"
)

// ClosedReason nombra el origen de un cierre. Los literales de cadena viven SOLAMENTE en
// estas constantes — nunca en línea en el lugar de llamada.
// Ejemplo: var reason businesscalendar.ClosedReason = businesscalendar.ClosedHoliday
type ClosedReason string

const (
	ClosedWeekly  ClosedReason = "WEEKLY"  // business_hours says closed that weekday
	ClosedHoliday ClosedReason = "HOLIDAY" // a Holiday row falls on that date
	ClosedLocal   ClosedReason = "CLOSURE" // a Closure row falls on that date
)

// DayDetail es la propia respuesta de este módulo para una fecha: los límites neutrales MÁS
// el por qué un día cerrado está cerrado. ClosedBy es vocabulario de este módulo y nunca
// cruza a un vecino — vea Reader a continuación.
//
// Es un modelo de resultado solo de salida: implementa model.Encodable para que la
// operación get_day_bounds pueda responder con él, y nunca es una tabla de BD.
// Ejemplo: detail, err := cal.GetDayDetail(date)
type DayDetail struct {
	Bounds tinytime.DayBounds
	// ClosedBy nombra por qué el día está cerrado. "" cuando Bounds.Open es true.
	ClosedBy ClosedReason
}

func (d *DayDetail) EncodeFields(w model.FieldWriter) {
	w.Bool("open", d.Bounds.Open)
	w.Int("open_min", int64(d.Bounds.OpenMin))
	w.Int("close_min", int64(d.Bounds.CloseMin))
	w.String("closed_by", string(d.ClosedBy))
}

func (d *DayDetail) IsNil() bool { return d == nil }

// Reader es el puerto DE SOLO LECTURA del que dependen los propios consumidores de este
// módulo — un alias de conveniencia, no el contrato que nombra un módulo de dominio hermano.
//
// Un vecino que solo necesita saber "si esta fecha es utilizable y entre qué
// minutos" (ej. veltylabs/appointment_booking) declara su PROPIA interfaz
// devolviendo tinytime.DayBounds — una hoja neutral que ambos lados ya importan —
// y este Module la satisface estructuralmente, sin adaptador y sin importación en
// ninguna dirección. Vea webtyp/docs/AGENDA_DOMAIN_MASTER_PLAN.md §3-bis para
// entender por qué: un módulo genérico de reservas no debe depender de ningún calendario
// institucional específico.
// Ejemplo: var reader businesscalendar.Reader = cal
type Reader interface {
	// GetDayBounds responde "si el establecimiento está abierto en esta fecha, y
	// entre qué minutos". date es la medianoche UTC en segundos.
	GetDayBounds(date tinytime.Unix) (tinytime.DayBounds, error)

	// GetWeekdayBounds responde "entre qué minutos trabaja normalmente el establecimiento
	// en este día de la semana" (0 = domingo … 6 = sábado) — solo la plantilla
	// semanal: los feriados y cierres locales son excepciones de fecha y NO se
	// aplican aquí (GetDayBounds los aplica). Un consumidor que valida un horario
	// semanal usa esto; uno que valida una fecha concreta usa GetDayBounds.
	//
	// Un día de la semana sin fila business_hours, o con is_open == false, está cerrado
	// (cerrado por defecto, misma regla que el paso 3 de GetDayDetail).
	GetWeekdayBounds(dayOfWeek int) (tinytime.DayBounds, error)
}

// GetDayDetail resuelve el estado del establecimiento para una fecha, INCLUYENDO
// por qué un día cerrado está cerrado. Esta es la única ruta de resolución — GetDayBounds
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
// getWeeklyBounds representa los pasos 3–4 de GetDayDetail: solo la plantilla semanal. La
// única implementación compartida por GetDayDetail y GetWeekdayBounds.
func (m *Module) getWeeklyBounds(dayOfWeek int) (DayDetail, error) {
	bh, err := m.businessHoursByDay(dayOfWeek)
	if err != nil {
		if e, ok := err.(domainError); ok && e == ErrNotFound {
			return DayDetail{ClosedBy: ClosedWeekly}, nil
		}
		return DayDetail{}, err
	}
	if !bh.IsOpen {
		return DayDetail{ClosedBy: ClosedWeekly}, nil
	}
	return DayDetail{Bounds: tinytime.DayBounds{Open: true, OpenMin: mustParseClock(bh.OpenMin), CloseMin: mustParseClock(bh.CloseMin)}}, nil
}

func (m *Module) GetDayDetail(date tinytime.Unix) (DayDetail, error) {
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

	return m.getWeeklyBounds(tinytime.Weekday(date))
}

// GetDayBounds es el puerto Reader: solo los límites neutrales, sin ninguna razón
// adjunta. Satisface estructuralmente cualquier interfaz propia con forma de BoundsReader
// de un hermano.
// Ejemplo: bounds, err := cal.GetDayBounds(date)
func (m *Module) GetDayBounds(date tinytime.Unix) (tinytime.DayBounds, error) {
	d, err := m.GetDayDetail(date)
	return d.Bounds, err
}

// GetWeekdayBounds devuelve las horas regulares de un día de la semana (0 = domingo … 6 =
// sábado) — la plantilla semanal, sin feriados o cierres locales. Vea
// Reader para saber cuándo usarlo en lugar de GetDayBounds.
// Ejemplo: bounds, err := cal.GetWeekdayBounds(1)
func (m *Module) GetWeekdayBounds(dayOfWeek int) (tinytime.DayBounds, error) {
	if dayOfWeek < 0 || dayOfWeek > 6 {
		return tinytime.DayBounds{}, ErrInvalidWeekday
	}
	d, err := m.getWeeklyBounds(dayOfWeek)
	return d.Bounds, err
}

var _ Reader = (*Module)(nil)
