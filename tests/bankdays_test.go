package tests

import (
	"testing"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
	tinytime "webtyp.com/time"
	"webtyp.com/unixid"

	"github.com/veltylabs/business_calendar"
)

func TestBankDayAfter(t *testing.T) {
	conn := mem.New()
	db := orm.New(conn)
	ids, _ := unixid.NewUnixID()

	cal, err := businesscalendar.New(db, businesscalendar.Deps{IDs: ids})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Helpers
	d := func(y, m, day int) tinytime.Unix {
		return tinytime.Date(y, m, day, 0, 0, 0)
	}

	setupHoliday := func(date tinytime.Unix, name string) {
		err := cal.AddHoliday(businesscalendar.Holiday{
			SpecificDate: int64(date),
			Name:         name,
		})
		if err != nil {
			t.Fatalf("setup holiday failed: %v", err)
		}
	}
	setupClosure := func(date tinytime.Unix, reason string) {
		err := cal.AddClosure(businesscalendar.Closure{
			SpecificDate: int64(date),
			Reason:       reason,
		})
		if err != nil {
			t.Fatalf("setup closure failed: %v", err)
		}
	}

	tests := []struct {
		name     string
		setup    func()
		from     tinytime.Unix
		days     int
		expected tinytime.Unix
		err      error
	}{
		{
			name:     "B1: Mon + 2 -> Wed",
			setup:    func() {},
			from:     d(2026, 10, 5),
			days:     2,
			expected: d(2026, 10, 7),
		},
		{
			name:     "B2: Fri + 1 -> Mon",
			setup:    func() {},
			from:     d(2026, 10, 9),
			days:     1,
			expected: d(2026, 10, 12),
		},
		{
			name:     "B3: Fri + 2 -> Tue",
			setup:    func() {},
			from:     d(2026, 10, 9),
			days:     2,
			expected: d(2026, 10, 13),
		},
		{
			name: "B4: Fri + 1 (with holiday Mon) -> Tue",
			setup: func() {
				setupHoliday(d(2026, 10, 12), "Holiday")
			},
			from:     d(2026, 10, 9),
			days:     1,
			expected: d(2026, 10, 13),
		},
		{
			name:     "B5: Sat + 0 -> Mon",
			setup:    func() {},
			from:     d(2026, 10, 10),
			days:     0,
			expected: d(2026, 10, 12),
		},
		{
			name:     "B6: Wed 14:30 + 0 -> Wed 00:00",
			setup:    func() {},
			from:     d(2026, 10, 7) + 14*3600 + 1800, // 14:30
			days:     0,
			expected: d(2026, 10, 7),
		},
		{
			name: "B7: Thu + 1 (with holidays Fri and Sat) -> Mon",
			setup: func() {
				setupHoliday(d(2026, 9, 18), "Holiday 1")
				setupHoliday(d(2026, 9, 19), "Holiday 2")
			},
			from:     d(2026, 9, 17),
			days:     1,
			expected: d(2026, 9, 21),
		},
		{
			name: "B8: Mon + 1 (with closure Tue) -> Tue (closures don't count)",
			setup: func() {
				setupClosure(d(2026, 10, 13), "Closure")
			},
			from:     d(2026, 10, 12),
			days:     1,
			expected: d(2026, 10, 13),
		},
		{
			name:     "B9: negative days",
			setup:    func() {},
			from:     d(2026, 10, 12),
			days:     -1,
			err:      businesscalendar.ErrNegativeDays,
		},
		{
			name:     "B10: Mon + 10 -> Mon two weeks later",
			setup:    func() {},
			from:     d(2026, 10, 5),
			days:     10,
			expected: d(2026, 10, 19),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// clean DB for each run
			conn = mem.New()
			db = orm.New(conn)
			cal, _ = businesscalendar.New(db, businesscalendar.Deps{IDs: ids})
			tt.setup()

			got, err := cal.BankDayAfter(tt.from, tt.days)
			if tt.err != nil {
				if err != tt.err {
					t.Fatalf("expected error %v, got %v", tt.err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("expected %v (%s), got %v (%s)",
					tt.expected, tinytime.FormatISO8601(tt.expected),
					got, tinytime.FormatISO8601(got))
			}
		})
	}
}
