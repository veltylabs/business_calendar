package businesscalendar

import (
	"webtyp.com/fmt"
	"webtyp.com/time"
)

func parseClock(hhmm string) (int, error) {
	m, err := time.ParseTime(hhmm)
	if err != nil {
		return 0, ErrInvalidMinutes
	}
	return int(m), nil
}

// storedClock lee una hora guardada ("HH:MM"). Un valor que no lo es (una fila escrita cuando la columna guardaba
// minutos: "540") es un error de los datos que nombra el valor — nunca un panic, que botaría el servidor.
func storedClock(field, hhmm string) (int, error) {
	m, err := parseClock(hhmm)
	if err != nil {
		return 0, fmt.Errf("business_calendar: stored business_hours.%s is not an HH:MM clock: %q", field, hhmm)
	}
	return m, nil
}

func formatClock(minutes int) string { return time.FormatMinutes(minutes) }
