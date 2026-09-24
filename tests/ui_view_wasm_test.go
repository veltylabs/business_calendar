//go:build wasm

package tests

import (
	"strings"
	"testing"

	"webtyp.com/dom"
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

func TestWASM_BusinessCalendar_ViewListsItsThreeTabsOnInit(t *testing.T) {
	mock := &mockCaller{
		onCall: func(op string, args model.Encodable, into model.Decodable) error {
			return nil
		},
	}

	m, err := ui.Browser(mock, &testIDGen{}, testTenantID)
	if err != nil {
		t.Fatalf("ui.Browser: %v", err)
	}
	html := initView(m)

	for _, label := range []string{"Horario", "Feriados", "Cierres"} {
		if !strings.Contains(html, label) {
			t.Errorf("falta la pestaña %q en la pantalla de Calendario", label)
		}
	}
	_ = businesscalendar.ModelName
}
