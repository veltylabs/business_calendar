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
func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error {
	if err := ddl.New(conn, ddlCompiler).CreateTable(&businesscalendar.BusinessHours{}); err != nil {
		return err
	}
	if err := ddl.New(conn, ddlCompiler).CreateTable(&businesscalendar.Holiday{}); err != nil {
		return err
	}
	return ddl.New(conn, ddlCompiler).CreateTable(&businesscalendar.Closure{})
}
