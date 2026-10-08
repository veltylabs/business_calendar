package tests

import (
	"testing"
	businesscalendar "github.com/veltylabs/business_calendar"
)

func TestSentinelErrorMessages(t *testing.T) {
	cases := []struct {
		err error
		msg string
	}{
		{businesscalendar.ErrNotFound, "business_calendar: record not found"},
		{businesscalendar.ErrInvalidDay, "business_calendar: day_of_week must be 0..6"},
		{businesscalendar.ErrInvalidMinutes, "business_calendar: minutes must be 0..1439 and open_min < close_min"},
		{businesscalendar.ErrDuplicateDate, "business_calendar: a record already exists for that date"},
	}

	for _, c := range cases {
		if c.err.Error() != c.msg {
			t.Errorf("expected error message %q, got %q", c.msg, c.err.Error())
		}
	}
}
