package businesscalendar

import (
	"webtyp.com/fmt"
	"webtyp.com/model"
)

// BusinessHoursModel: one row per day of week. day_of_week is unique — this
// module never writes more than seven rows.
//
// Times are MINUTES FROM MIDNIGHT (0..1439), the same encoding
// appointment_booking uses for work blocks. The archived business_hours module
// stored clock-time text and forced a conversion at the one boundary that has
// to compare the two; the encoding is fixed here instead.
var BusinessHoursModel = model.Definition{
	Name: "business_hours",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "day_of_week", Type: model.Int(), NotNull: true, DB: &model.FieldDB{Unique: true}},
		{Name: "open_min", Type: model.Int(), NotNull: true},
		{Name: "close_min", Type: model.Int(), NotNull: true},
		{Name: "is_open", Type: model.Bool(), NotNull: true},
		{Name: "notes", Type: model.Text()},
		{Name: "updated_at", Type: model.Int(), NotNull: true},
	},
}

// HolidayModel: a dated closure with a legal/administrative origin. Editable at
// runtime — a holiday that appears by law must never require a recompile.
var HolidayModel = model.Definition{
	Name: "holiday",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "specific_date", Type: model.Int(), NotNull: true}, // midnight UTC, seconds
		{Name: "name", Type: model.Text(), NotNull: true},
		{Name: "notes", Type: model.Text()},
		{Name: "updated_at", Type: model.Int(), NotNull: true},
	},
}

// ClosureModel: a dated closure the establishment itself decided (maintenance,
// an event). Separate from Holiday because the ORIGIN is what the UI shows and
// what an administrator filters by — collapsing both into one table with a
// "type" column would make the origin a magic string.
var ClosureModel = model.Definition{
	Name: "closure",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "specific_date", Type: model.Int(), NotNull: true},
		{Name: "reason", Type: model.Text(), NotNull: true},
		{Name: "updated_at", Type: model.Int(), NotNull: true},
	},
}

var (
	ErrNotFound       = fmt.Err("business_calendar: record not found")
	ErrInvalidDay     = fmt.Err("business_calendar: day_of_week must be 0..6")
	ErrInvalidMinutes = fmt.Err("business_calendar: minutes must be 0..1439 and open_min < close_min")
	ErrDuplicateDate  = fmt.Err("business_calendar: a record already exists for that date")
)

// EventCalendarChanged fires on every write. appointment_booking subscribes so
// a professional's editor reflects widened hours without a recompile (CU-03),
// a new holiday closes the day for everyone (CU-04), and reservations already
// booked on a day that just became closed are recomputed into conflict.
const EventCalendarChanged = "business.calendar.changed"

// Payload kinds — the ONLY place these literals live. CalendarChangedPayload.Kind
// carries one of them.
const (
	CalendarKindBusinessHours = "BUSINESS_HOURS"
	CalendarKindHoliday       = "HOLIDAY"
	CalendarKindClosure       = "CLOSURE"
)

// CalendarChangedPayload says WHAT changed, not merely that something did.
// A subscriber that only learns "the calendar changed" has to recompute every
// reservation of every professional; with the range it recomputes a window.
//
// Implements model.Encodable, which events.Event.Payload requires.
type CalendarChangedPayload struct {
	// FromDate/ToDate bound the affected dates (midnight UTC seconds).
	// For a holiday or closure both equal that single date. For a business
	// hours change they are 0/0 — a weekly rule has no bounded range, and the
	// subscriber must recompute its whole horizon.
	FromDate int64
	ToDate   int64
	// Kind names what moved, so a subscriber can skip work it does not care
	// about. One of CalendarKindBusinessHours | CalendarKindHoliday |
	// CalendarKindClosure.
	Kind string
	// Closed reports the direction. true when the change CLOSES time (a
	// holiday added, hours narrowed) — the only direction that can invalidate
	// an existing reservation. false when it opens time, which never does.
	Closed bool
}

func (p *CalendarChangedPayload) EncodeFields(w model.FieldWriter) {
	w.Int("from_date", p.FromDate)
	w.Int("to_date", p.ToDate)
	w.String("kind", p.Kind)
	w.Bool("closed", p.Closed)
}

func (p *CalendarChangedPayload) DecodeFields(r model.FieldReader) {
	if v, ok := r.Int("from_date"); ok {
		p.FromDate = v
	}
	if v, ok := r.Int("to_date"); ok {
		p.ToDate = v
	}
	if v, ok := r.String("kind"); ok {
		p.Kind = v
	}
	if v, ok := r.Bool("closed"); ok {
		p.Closed = v
	}
}

func (p *CalendarChangedPayload) IsNil() bool { return p == nil }

// Transport-only Args Definitions.
//
// Constraints follow the harness's NotNull semantics: on a base int/bool kind,
// NotNull rejects the ZERO value (ValidateFields: NotNull && IsZeroPtr), so
// day_of_week 0 (Sunday), is_open false, or midnight (0 minutes) would be
// unrepresentable if NotNull were set here. Required ints are therefore
// enforced as DOMAIN rules in the service (ErrInvalidDay/ErrInvalidMinutes),
// the single fail-closed chokepoint every write path crosses; NotNull stays
// only on the text fields where empty is never legal.
var UpsertBusinessHoursArgsModel = model.Definition{
	Name: "upsert_business_hours_args",
	Fields: model.Fields{
		{Name: "day_of_week", Type: model.Int()},
		{Name: "open_min", Type: model.Int()},
		{Name: "close_min", Type: model.Int()},
		{Name: "is_open", Type: model.Bool()},
		{Name: "notes", Type: model.Text()},
	},
}

var GetDayBoundsArgsModel = model.Definition{
	Name: "get_day_bounds_args",
	Fields: model.Fields{
		{Name: "date", Type: model.Int()},
	},
}

var ListHolidaysArgsModel = model.Definition{
	Name: "list_holidays_args",
	Fields: model.Fields{
		{Name: "from", Type: model.Int()},
		{Name: "to", Type: model.Int()},
	},
}

var AddHolidayArgsModel = model.Definition{
	Name: "add_holiday_args",
	Fields: model.Fields{
		{Name: "specific_date", Type: model.Int()},
		{Name: "name", Type: model.Text(), NotNull: true},
		{Name: "notes", Type: model.Text()},
	},
}

var RemoveHolidayArgsModel = model.Definition{
	Name: "remove_holiday_args",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), NotNull: true},
	},
}

var ListClosuresArgsModel = model.Definition{
	Name: "list_closures_args",
	Fields: model.Fields{
		{Name: "from", Type: model.Int()},
		{Name: "to", Type: model.Int()},
	},
}

var AddClosureArgsModel = model.Definition{
	Name: "add_closure_args",
	Fields: model.Fields{
		{Name: "specific_date", Type: model.Int()},
		{Name: "reason", Type: model.Text(), NotNull: true},
	},
}

var RemoveClosureArgsModel = model.Definition{
	Name: "remove_closure_args",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), NotNull: true},
	},
}
