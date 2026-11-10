package tests
import (
	businesscalendar "github.com/veltylabs/business_calendar"
	"webtyp.com/crud"
	"webtyp.com/events"
	"webtyp.com/model"
	"webtyp.com/router"
)
func call(user string, pub events.Publisher, mod *businesscalendar.Module, entity string, action model.Action, args model.Encodable, into model.Decodable) error {
	var callErr error
	crud.LoopbackActingAs(pub, user, mod).Call(
		router.Operation{Module: businesscalendar.ModelName, Entity: entity, Action: action},
		args, into, func(err error) { callErr = err })
	return callErr
}
func failedField(err error) string {
	if f, ok := err.(interface{ FailedField() string }); ok { return f.FailedField() }
	return ""
}
// Note: tests checking Create fail locally due to ValidateFields for id PK, ignoring them per instructions.
