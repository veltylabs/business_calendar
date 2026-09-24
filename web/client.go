//go:build wasm

package main

import (
	. "webtyp.com/dom"
	"webtyp.com/events/mock"
	"webtyp.com/layout/platformd"
	"webtyp.com/orm"
	"webtyp.com/router/loopback"
	"webtyp.com/storage/mem"
	"webtyp.com/unixid"

	businesscalendar "github.com/veltylabs/business_calendar"
	"github.com/veltylabs/business_calendar/seed"
	"github.com/veltylabs/business_calendar/ui"
)

// demoTenantID is the only tenant of this in-browser demo.
const demoTenantID = "demo"

// demoUser is the fixed identity the demo shell shows: the demo has no login.
type demoUser struct{}

func (demoUser) UserName() string    { return "Demo" }
func (demoUser) UserAvatar() string  { return "" }
func (demoUser) UserRoles() []string { return []string{"Administrador"} }

func main() {
	ids, err := unixid.NewUnixID()
	if err != nil {
		panic(err)
	}
	db := orm.New(mem.New())
	broker := &mock.Broker{}

	bc, err := businesscalendar.New(db, businesscalendar.Deps{
		IDs:       ids,
		Publisher: broker,
	})
	if err != nil {
		panic(err)
	}

	if _, err := seed.Load(bc); err != nil {
		panic(err)
	}

	caller := loopback.WithTenant(demoTenantID, bc)

	v, err := ui.Browser(caller, ids, demoTenantID)
	if err != nil {
		panic(err)
	}

	p := &platformd.Platform{
		AppName:   ui.Label + " — demo",
		User:      demoUser{},
		Modules:   []platformd.UIModule{v},
		DefaultID: ui.ID,
	}
	Append("body", p)
	select {}
}
