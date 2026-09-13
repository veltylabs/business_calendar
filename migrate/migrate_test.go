package migrate_test

import (
	"testing"

	"webtyp.com/ddl"
	"webtyp.com/model"

	"github.com/veltylabs/business_calendar/migrate"
)

type dummyExecer struct {
	calls []string
}

func (d *dummyExecer) Exec(query string, args ...any) error {
	d.calls = append(d.calls, query)
	return nil
}

type dummyCompiler struct{}

func (d *dummyCompiler) CompileDDL(stmt ddl.Stmt, m model.Model) (string, []any, error) {
	return stmt.Table, nil, nil
}

func TestMigrate_CreatesThreeTables(t *testing.T) {
	execer := &dummyExecer{}
	compiler := &dummyCompiler{}

	if err := migrate.Migrate(execer, compiler); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	if len(execer.calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(execer.calls))
	}
}

func TestMigrate_TableOrder(t *testing.T) {
	execer := &dummyExecer{}
	compiler := &dummyCompiler{}

	if err := migrate.Migrate(execer, compiler); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	expected := []string{"business_hours", "holiday", "closure"}
	if len(execer.calls) != len(expected) {
		t.Fatalf("expected calls %v, got %v", expected, execer.calls)
	}
	for i, name := range expected {
		if execer.calls[i] != name {
			t.Fatalf("call %d: expected %s, got %s", i, name, execer.calls[i])
		}
	}
}
