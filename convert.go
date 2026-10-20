package businesscalendar

import tinytime "webtyp.com/time"

func parseClock(hhmm string) (int, error) {
	m, err := tinytime.ParseTime(hhmm)
	if err != nil {
		return 0, ErrInvalidMinutes
	}
	return int(m), nil
}

func mustParseClock(hhmm string) int {
	m, err := parseClock(hhmm)
	if err != nil {
		panic("businesscalendar: invalid stored clock " + hhmm)
	}
	return m
}

func formatClock(minutes int) string { return tinytime.FormatMinutes(minutes) }
