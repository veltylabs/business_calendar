package businesscalendar

import (
	"webtyp.com/input"
	"webtyp.com/model"
	"webtyp.com/time"
)

var BusinessHoursModel = model.Definition{
	Name: "business_hours",
	Allow: model.Create | model.Read | model.Update,
	Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.HaveNoBusinessRule, Update: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), OmitEmpty: true, ServerOnly: true, DB: &model.FieldDB{PK: true}},
		{Name: "day_of_week", Type: input.Number(), Label: "Day of week", DB: &model.FieldDB{Unique: true}, Item: model.ItemLabel},
		{Name: "open_min", Type: input.Hour(), Label: "Opens at", NotNull: true, Item: model.ItemDescription},
		{Name: "close_min", Type: input.Hour(), Label: "Closes at", NotNull: true, Item: model.ItemDescription},
		{Name: "is_open", Type: input.Checkbox(), Label: "Open", Item: model.ItemDescription},
		{Name: "notes", Type: input.Textarea(), Label: "Notes", Item: model.ItemDescription},
		{Name: "updated_at", Type: model.NanoTime(), NotNull: true, ServerOnly: true},
	},
}

var HolidayModel = model.Definition{
	Name: "holiday",
	Allow: model.Create | model.Read | model.Delete,
	Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.HaveNoBusinessRule, Delete: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), OmitEmpty: true, ServerOnly: true, DB: &model.FieldDB{PK: true}},
		{Name: "specific_date", Type: input.Number(), Label: "Date", NotNull: true, Item: model.ItemDescription}, // midnight UTC, seconds
		{Name: "name", Type: input.Text(), Label: "Name", NotNull: true, Item: model.ItemLabel},
		{Name: "notes", Type: input.Textarea(), Label: "Notes"},
		{Name: "updated_at", Type: model.NanoTime(), NotNull: true, ServerOnly: true},
	},
}

var ClosureModel = model.Definition{
	Name: "closure",
	Allow: model.Create | model.Read | model.Delete,
	Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.HaveNoBusinessRule, Delete: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), OmitEmpty: true, ServerOnly: true, DB: &model.FieldDB{PK: true}},
		{Name: "specific_date", Type: input.Number(), Label: "Date", NotNull: true, Item: model.ItemDescription},
		{Name: "reason", Type: input.Text(), Label: "Reason", NotNull: true, Item: model.ItemLabel},
		{Name: "updated_at", Type: model.NanoTime(), NotNull: true, ServerOnly: true},
	},
}

var DayStatusModel = model.Definition{
	Name:  "day_detail",
	Wire:  true,
	Allow: model.Read,
	Rules: model.Rules{Read: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "date", Type: model.Int(), OmitEmpty: true},
		{Name: "open", Type: model.Bool(), OmitEmpty: true},
		{Name: "open_min", Type: model.Int(), OmitEmpty: true},
		{Name: "close_min", Type: model.Int(), OmitEmpty: true},
		{Name: "closed_by", Type: model.Text(), OmitEmpty: true},
	},
}

var Definition = model.ModuleDefinition{
	Name: "business_calendar",
	Entities: []model.Definition{BusinessHoursModel, HolidayModel, ClosureModel, DayStatusModel},
}

const (
	ErrNotFound       model.NotFound = "business_calendar: record not found"
	ErrInvalidDay     model.Invalid = "business_calendar: day_of_week must be 0..6"
	ErrInvalidWeekday model.Invalid = "business_calendar: day_of_week must be 0..6"
	ErrInvalidMinutes model.Invalid = "business_calendar: minutes must be 0..1439 and open_min < close_min"
	ErrDuplicateDate  model.Conflict = "business_calendar: a record already exists for that date"
	ErrCannotRead     model.Invalid = "business_calendar: cannot read the request"
	ErrFieldNotEditable model.Invalid = "business_calendar: that field cannot be changed"
	ErrDateRequired     model.Invalid = "business_calendar: date is required"
)

const EventCalendarChanged = "business.calendar.changed"

const (
	CalendarKindBusinessHours = "BUSINESS_HOURS"
	CalendarKindHoliday       = "HOLIDAY"
	CalendarKindClosure       = "CLOSURE"
)

type CalendarChangedPayload struct {
	FromDate int64
	ToDate   int64
	Kind string
	Closed bool
}

func (p *CalendarChangedPayload) EncodeFields(w model.FieldWriter) {
	w.Int("from_date", p.FromDate)
	w.Int("to_date", p.ToDate)
	w.String("kind", p.Kind)
	w.Bool("closed", p.Closed)
}

func (p *CalendarChangedPayload) DecodeFields(r model.FieldReader) {
	if v, ok := r.Int("from_date"); ok { p.FromDate = v }
	if v, ok := r.Int("to_date"); ok { p.ToDate = v }
	if v, ok := r.String("kind"); ok { p.Kind = v }
	if v, ok := r.Bool("closed"); ok { p.Closed = v }
}

func (p *CalendarChangedPayload) IsNil() bool { return p == nil }

type ClosedReason string

const (
	ClosedWeekly  ClosedReason = "WEEKLY"
	ClosedHoliday ClosedReason = "HOLIDAY"
	ClosedLocal   ClosedReason = "CLOSURE"
)

type DayDetail struct {
	Limits time.DayLimits
	ClosedBy ClosedReason
}

func (d *DayDetail) EncodeFields(w model.FieldWriter) {
	w.Bool("open", d.Limits.Open)
	w.Int("open_min", int64(d.Limits.OpenMin))
	w.Int("close_min", int64(d.Limits.CloseMin))
	w.String("closed_by", string(d.ClosedBy))
}

func (d *DayDetail) IsNil() bool { return d == nil }
