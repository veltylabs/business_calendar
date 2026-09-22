package businesscalendar

import (
	"webtyp.com/date"
	"webtyp.com/fmt"
	"webtyp.com/fmt/lang"
	"webtyp.com/model"
	"webtyp.com/router"
	tinytime "webtyp.com/time"
	"webtyp.com/view"
)

// Item projects a BusinessHours row as a view.Item (view.Itemizer). The weekday
// label is rendered from the canonical English name (date.WeekdayName) through
// lang.Translate — this module hardcodes no human language; the consuming app
// registers the dictionary.
func (r *BusinessHours) Item() view.Item {
	return view.Item{
		ID:          r.Id,
		Label:       lang.Translate(date.WeekdayName(int(r.DayOfWeek))).String(),
		Description: hoursLabel(r),
	}
}

// specificDateLabel renders a SpecificDate (SECONDS, midnight UTC — see
// model.go's own doc comment) as its calendar-date string. Deliberately
// tinytime.FormatISO8601, never tinytime.FormatDate: FormatDate applies the
// LOCAL timezone offset, which is correct for a genuine point in time but
// wrong for a value that already encodes a calendar day at UTC midnight —
// subtracting hours from UTC midnight crosses into the previous day in any
// negative offset (Chile, UTC-3/-4). This is the same trap
// mjosefa-cms/modules/appointment_booking's unixToDay already documents and
// avoids for the identical reason. FormatISO8601 takes UnixNANO, matching
// tinytime.Now()/ParseDate's own convention — hence *1e9 from the stored
// seconds — and its first 10 characters are exactly "YYYY-MM-DD".
func specificDateLabel(seconds int64) string {
	return tinytime.FormatISO8601(seconds * 1000000000)[:10]
}

// Item projects a Holiday as a view.Item.
func (h *Holiday) Item() view.Item {
	return view.Item{
		ID:          h.Id,
		Label:       lang.Translate(h.Name).String(),
		Description: specificDateLabel(h.SpecificDate),
	}
}

// Item projects a Closure as a view.Item.
func (c *Closure) Item() view.Item {
	return view.Item{
		ID:          c.Id,
		Label:       lang.Translate(c.Reason).String(),
		Description: specificDateLabel(c.SpecificDate),
	}
}

// hoursLabel renders one weekly row's window for display — a minutes range when
// open, the notes or a translated "Closed" when not. Presentation only; the
// ops never apply this formatting.
func hoursLabel(r *BusinessHours) string {
	if !r.IsOpen {
		if r.Notes != "" {
			return lang.Translate(r.Notes).String()
		}
		return lang.Translate("Closed").String()
	}
	return minutesHHMM(int(r.OpenMin)) + "–" + minutesHHMM(int(r.CloseMin))
}

// minutesHHMM renders minutes-from-midnight as a two-digit clock pair.
func minutesHHMM(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}

// NewBusinessHoursView builds the weekly-hours Presenter — read + save
// (upsert). The app decides which renderer draws it.
func NewBusinessHoursView(caller router.Caller) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{Module: ModelName, List: OpListBusinessHours, Save: OpUpsertBusinessHours},
		func() model.ModelSlice { return &BusinessHoursList{} })
	return view.New(b, &BusinessHours{}, view.WithTitle(lang.Translate("Business hours").String()))
}

// NewHolidaysView builds the holidays Presenter — full CRUD (list, add, remove).
func NewHolidaysView(caller router.Caller) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{Module: ModelName, List: OpListHolidays, Save: OpAddHoliday, Delete: OpRemoveHoliday},
		func() model.ModelSlice { return &HolidayList{} })
	return view.New(b, &Holiday{}, view.WithTitle(lang.Translate("Holidays").String()))
}

// NewClosuresView builds the closures Presenter — full CRUD (list, add, remove).
func NewClosuresView(caller router.Caller) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{Module: ModelName, List: OpListClosures, Save: OpAddClosure, Delete: OpRemoveClosure},
		func() model.ModelSlice { return &ClosureList{} })
	return view.New(b, &Closure{}, view.WithTitle(lang.Translate("Closures").String()))
}
