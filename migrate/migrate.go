package migrate

import (
	"webtyp.com/ddl"

	businesscalendar "github.com/veltylabs/business_calendar"
)

// Migrate reconcilia el esquema de base de datos que posee business_calendar: BusinessHours, Holiday, Closure.
//
// Deliberadamente NO es llamado por New, y vive deliberadamente en su propio
// paquete en lugar de un nuevo archivo en el paquete raíz: nada en la ruta de
// construcción WASM de una aplicación consumidora (su view.go, que importa el
// paquete raíz businesscalendar para businesscalendar.NewView) importa jamás
// "github.com/veltylabs/business_calendar/migrate", por lo que webtyp.com/ddl nunca
// entra en ese grafo de construcción, independientemente de los build tags en el lado del consumidor.
//
// conn es un ddl.Execer, no un *orm.DB, por lo que un transporte en tiempo de despliegue que solo
// puede ejecutar DDL lo satisface. El RawConn() de un *orm.DB también lo satisface,
// para llamadas locales/de prueba:
//
//	conn, _ := postgres.Open(dsn)
//	compiler, _ := conn.(ddl.Compiler)
//	err := migrate.Migrate(conn, compiler)
//
// Ejemplo: err := migrate.Migrate(conn, compiler)
func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error {
	// Sync y no CreateTable: crea la tabla que falta y además agrega las columnas que un modelo
	// ganó, así un campo nuevo llega a una base ya instalada. Flujo: https://github.com/webtyp/ddl/blob/main/docs/diagrams/SYNC_FLOW.md
	return ddl.New(conn, ddlCompiler).Sync(
		&businesscalendar.BusinessHours{},
		&businesscalendar.Holiday{},
		&businesscalendar.Closure{},
	)
}
