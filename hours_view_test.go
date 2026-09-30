package businesscalendar

import "testing"

func TestHHMM(t *testing.T) {
	cases := []struct {
		min  int64
		want string
	}{
		{0, "00:00"},
		{480, "08:00"},
		{1080, "18:00"},
		{1439, "23:59"},
		{1440, "24:00"},
	}
	for _, tc := range cases {
		got := hhmm(tc.min)
		if got != tc.want {
			t.Errorf("hhmm(%d) = %q, want %q", tc.min, got, tc.want)
		}
	}
}
