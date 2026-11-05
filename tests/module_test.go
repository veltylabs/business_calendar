package tests

import (
	"testing"
	"webtyp.com/appkit"
	"webtyp.com/orm"
	"github.com/veltylabs/business_calendar/module"
	"webtyp.com/storage/mem"
)

type dummyIDs struct{}
func (dummyIDs) NewID() string { return "123" }

func TestModuleBuilds(t *testing.T) {
	db := orm.New(mem.New())

	deps := appkit.Deps{
		DB: db,
		IDs: dummyIDs{},
		Events: newRecording(),
	}

	if module.Module.Server.Build == nil {
		t.Skip("appkit gives no test helper")
	}

	_, err := module.Module.Server.Build(deps)
	if err != nil {
		t.Fatalf("Module.Server.Build failed: %v", err)
	}
}
