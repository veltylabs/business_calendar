//go:build browser

package tests

import (
	"testing"

	"webtyp.com/model"

	"github.com/veltylabs/business_calendar/ui"
)

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

func TestWASM_BusinessCalendar_Screens(t *testing.T) {
	mock := &mockCaller{}

	screens := ui.Screens(mock)
	if len(screens) != 1 {
		t.Fatalf("se esperaba 1 pantalla raíz, pero devolvió %d", len(screens))
	}

	root := screens[0]
	if err := root.Validate(); err != nil {
		t.Fatalf("la pantalla raíz falló la validación: %v", err)
	}
	if root.ID != ui.ID {
		t.Errorf("se esperaba ID %q, pero se obtuvo %q", ui.ID, root.ID)
	}

	children := root.Children
	if len(children) != 3 {
		t.Fatalf("se esperaban 3 pantallas hijas, pero hay %d", len(children))
	}

	expectedIDs := []string{
		ui.ID + "." + ui.TabHoursID,
		ui.ID + "." + ui.TabHolidaysID,
		ui.ID + "." + ui.TabClosuresID,
	}

	for i, child := range children {
		if err := child.Validate(); err != nil {
			t.Errorf("el hijo %d falló la validación: %v", i, err)
		}
		if child.Permission == "" {
			t.Errorf("el hijo %d no tiene Permiso definido", i)
		}
		if child.ID != expectedIDs[i] {
			t.Errorf("el hijo %d debería tener ID %q, pero tiene %q", i, expectedIDs[i], child.ID)
		}
		if child.Presenter == nil {
			t.Errorf("el hijo %d no tiene Presenter definido", i)
		}
	}
}