package tests

import (
	"testing"

	businesscalendar "github.com/veltylabs/business_calendar"
	"webtyp.com/json"
	"webtyp.com/model"
	"webtyp.com/router/mock"
)

type decodedItem struct {
	DayOfWeek int64
	OpenMin   int64
	CloseMin  int64
	IsOpen    bool
	Day       string
	Opens     string
	Closes    string
}

func (d *decodedItem) Schema() []model.Field { return nil }
func (d *decodedItem) Pointers() []any       { return nil }
func (d *decodedItem) IsNil() bool           { return d == nil }
func (d *decodedItem) EncodeFields(_ model.FieldWriter) {}

func (d *decodedItem) DecodeFields(r model.FieldReader) {
	if v, ok := r.Int("day_of_week"); ok {
		d.DayOfWeek = v
	}
	if v, ok := r.Int("open_min"); ok {
		d.OpenMin = v
	}
	if v, ok := r.Int("close_min"); ok {
		d.CloseMin = v
	}
	if v, ok := r.Bool("is_open"); ok {
		d.IsOpen = v
	}
	if v, ok := r.String("day"); ok {
		d.Day = v
	}
	if v, ok := r.String("opens"); ok {
		d.Opens = v
	}
	if v, ok := r.String("closes"); ok {
		d.Closes = v
	}
}

type decodedItemList []*decodedItem

func (s *decodedItemList) Schema() []model.Field  { return nil }
func (s *decodedItemList) Pointers() []any        { return nil }
func (s *decodedItemList) Len() int               { return len(*s) }
func (s *decodedItemList) At(i int) model.Fielder { return (*s)[i] }
func (s *decodedItemList) Append() model.Fielder {
	v := &decodedItem{}
	*s = append(*s, v)
	return v
}
func (s *decodedItemList) IsNil() bool                      { return s == nil }
func (s *decodedItemList) EncodeFields(_ model.FieldWriter) {}
func (s *decodedItemList) DecodeFields(_ model.FieldReader) {}

func TestOpListBusinessHours_ReadableFieldsAndHHMM(t *testing.T) {
	m, _ := newModule(t, nil)

	// Insert Tuesday (day 2): 08:00 (480) - 18:00 (1080), open
	err := m.UpsertBusinessHours(businesscalendar.BusinessHours{
		DayOfWeek: 2,
		OpenMin:   480,
		CloseMin:  1080,
		IsOpen:    true,
	})
	if err != nil {
		t.Fatalf("upsert Tuesday: %v", err)
	}

	// Insert Sunday (day 0): 00:00 (0) - 00:00 (0), closed
	err = m.UpsertBusinessHours(businesscalendar.BusinessHours{
		DayOfWeek: 0,
		OpenMin:   0,
		CloseMin:  0,
		IsOpen:    false,
	})
	if err != nil {
		t.Fatalf("upsert Sunday: %v", err)
	}

	reg := mounted(t, m)
	ctx := &mock.Context{}
	ctx.SetUserID("u1")
	reg.Invoke("OP", "/"+businesscalendar.OpListBusinessHours, ctx)
	if ctx.Status != 0 {
		t.Fatalf("expected status 0 (success), got %d body=%s", ctx.Status, ctx.ResponseBody())
	}

	body := ctx.ResponseBody()

	var list decodedItemList
	if err := json.Decode(body, &list); err != nil {
		t.Fatalf("failed to parse response JSON: %v (body=%s)", err, body)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 items in array, got %d", len(list))
	}

	var tueItem, sunItem *decodedItem
	for _, item := range list {
		if item.DayOfWeek == 2 {
			tueItem = item
		} else if item.DayOfWeek == 0 {
			sunItem = item
		}
	}

	if tueItem == nil {
		t.Fatal("Tuesday item not found in response")
	}
	if sunItem == nil {
		t.Fatal("Sunday item not found in response")
	}

	// Tuesday check
	if tueItem.Day != "Tuesday" {
		t.Errorf("Tuesday day = %q, want %q", tueItem.Day, "Tuesday")
	}
	if tueItem.Opens != "08:00" {
		t.Errorf("Tuesday opens = %q, want %q", tueItem.Opens, "08:00")
	}
	if tueItem.Closes != "18:00" {
		t.Errorf("Tuesday closes = %q, want %q", tueItem.Closes, "18:00")
	}
	if tueItem.OpenMin != 480 {
		t.Errorf("Tuesday open_min = %d, want 480", tueItem.OpenMin)
	}
	if tueItem.CloseMin != 1080 {
		t.Errorf("Tuesday close_min = %d, want 1080", tueItem.CloseMin)
	}

	// Sunday check
	if sunItem.Day != "Sunday" {
		t.Errorf("Sunday day = %q, want %q", sunItem.Day, "Sunday")
	}
	if sunItem.Opens != "" {
		t.Errorf("Sunday opens present when closed: %q", sunItem.Opens)
	}
	if sunItem.Closes != "" {
		t.Errorf("Sunday closes present when closed: %q", sunItem.Closes)
	}
}

func TestHHMM_Boundaries(t *testing.T) {
	m, _ := newModule(t, nil)

	// Test minutes 0, 480, 1080, 1439
	cases := []struct {
		dayMin    int64
		openMin   int64
		closeMin  int64
		wantOpen  string
		wantClose string
	}{
		{1, 0, 1439, "00:00", "23:59"},
		{3, 480, 1080, "08:00", "18:00"},
	}

	for _, tc := range cases {
		err := m.UpsertBusinessHours(businesscalendar.BusinessHours{
			DayOfWeek: tc.dayMin,
			OpenMin:   tc.openMin,
			CloseMin:  tc.closeMin,
			IsOpen:    true,
		})
		if err != nil {
			t.Fatalf("upsert day %d: %v", tc.dayMin, err)
		}
	}

	reg := mounted(t, m)
	ctx := &mock.Context{}
	ctx.SetUserID("u1")
	reg.Invoke("OP", "/"+businesscalendar.OpListBusinessHours, ctx)
	if ctx.Status != 0 {
		t.Fatalf("expected status 0, got %d", ctx.Status)
	}

	var list decodedItemList
	if err := json.Decode(ctx.ResponseBody(), &list); err != nil {
		t.Fatalf("json decode error: %v", err)
	}

	for _, tc := range cases {
		var found *decodedItem
		for _, item := range list {
			if item.DayOfWeek == tc.dayMin {
				found = item
				break
			}
		}
		if found == nil {
			t.Fatalf("day %d not found in response", tc.dayMin)
		}
		if found.Opens != tc.wantOpen {
			t.Errorf("day %d opens = %q, want %q", tc.dayMin, found.Opens, tc.wantOpen)
		}
		if found.Closes != tc.wantClose {
			t.Errorf("day %d closes = %q, want %q", tc.dayMin, found.Closes, tc.wantClose)
		}
	}
}
