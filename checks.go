package businesscalendar

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

func normalizeHours(h BusinessHours) (BusinessHours, error) {
	if h.DayOfWeek < 0 || h.DayOfWeek > 6 {
		return h, &model.FieldError{Field: "day_of_week", Err: ErrInvalidDay}
	}
	if !h.IsOpen {
		h.OpenMin, h.CloseMin = "00:00", "00:00"
		return h, nil
	}
	openMin, openErr := parseClock(h.OpenMin)
	closeMin, closeErr := parseClock(h.CloseMin)
	if openErr != nil || closeErr != nil || openMin >= closeMin {
		return h, &model.FieldError{Field: "open_min", Err: ErrInvalidMinutes}
	}
	return h, nil
}

func checkNewHoliday(tx crud.Tx, h *Holiday) error {
	_, found, err := firstRow(tx, func() model.Model { return &Holiday{} }, storage.Eq("specific_date", h.SpecificDate))
	if err != nil { return err }
	if found { return ErrDuplicateDate }
	return nil
}

func checkNewClosure(tx crud.Tx, c *Closure) error {
	_, found, err := firstRow(tx, func() model.Model { return &Closure{} }, storage.Eq("specific_date", c.SpecificDate))
	if err != nil { return err }
	if found { return ErrDuplicateDate }
	return nil
}

func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value { return true }
	}
	return false
}

func hoursCloseTime(prev, next BusinessHours) bool {
	if !next.IsOpen { return prev.IsOpen }
	if !prev.IsOpen { return false }
	nextOpen, _ := parseClock(next.OpenMin)
	nextClose, _ := parseClock(next.CloseMin)
	prevOpen, _ := parseClock(prev.OpenMin)
	prevClose, _ := parseClock(prev.CloseMin)
	return nextOpen > prevOpen || nextClose < prevClose
}
