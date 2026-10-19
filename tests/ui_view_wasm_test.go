//go:build wasm

package tests

import (
	"testing"

	"webtyp.com/dom"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"

	businesscalendar "github.com/veltylabs/business_calendar"
	"github.com/veltylabs/business_calendar/ui"
)

const testTenantID = "demo"

type testIDGen struct{ n int }

func (g *testIDGen) NewID() string {
	g.n++
	return "test-id-1"
}

type mockCaller struct {
	lastOp   string
	lastArgs model.Encodable
	onCall   func(op string, args model.Encodable, into model.Decodable) error
}

func (m *mockCaller) Call(op string, args model.Encodable, into model.Decodable, done func(err error)) {
	m.lastOp = op
	m.lastArgs = args
	if m.onCall != nil {
		err := m.onCall(op, args, into)
		done(err)
	} else {
		done(nil)
	}
}

func (m *mockCaller) Dispatch(op string, args model.Encodable) {
	m.lastOp = op
	m.lastArgs = args
}

func initView(m interface{ View() dom.Component }) string {
	comp := m.View()
	if initer, ok := comp.(interface{ Init(ctx dom.Ctx) }); ok {
		initer.Init(nil)
	}
	if r, ok := comp.(dom.ViewRenderer); ok {
		return r.Render().String()
	}
	return comp.String()
}

func TestWASM_BusinessCalendar_BrowserIsAGroupOfItsThreeScreens(t *testing.T) {
	mock := &mockCaller{
		onCall: func(op string, args model.Encodable, into model.Decodable) error {
			return nil
		},
	}

	m, err := ui.Browser(mock, &testIDGen{}, testTenantID)
	if err != nil {
		t.Fatalf("ui.Browser: %v", err)
	}

	gm, ok := m.(platformd.GroupModule)
	if !ok {
		t.Fatalf("se esperaba que ui.Browser devolviera un platformd.GroupModule, pero devolvió %T", m)
	}

	children := gm.Children()
	if len(children) != 3 {
		t.Fatalf("se esperaban 3 submódulos, pero hay %d", len(children))
	}

	expectedModels := []string{
		"business_calendar.hours",
		"business_calendar.holidays",
		"business_calendar.closures",
	}

	for i, child := range children {
		if modelName := child.ModelName(); modelName != expectedModels[i] {
			t.Errorf("el hijo %d debería tener ModelName %q, pero tiene %q", i, expectedModels[i], modelName)
		}
	}

	for i, label := range []string{"Horario", "Feriados", "Cierres"} {
		if got := children[i].Label(); got != label {
			t.Errorf("el hijo %d debería llamarse %q, pero se llama %q", i, label, got)
		}
		if html := initView(children[i]); html == "" {
			t.Errorf("la pantalla del hijo %d (%s) no renderiza nada", i, label)
		}
	}

	_ = businesscalendar.ModelName
}
