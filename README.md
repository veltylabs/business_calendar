# business_calendar
<img src="docs/img/badges.svg">

El **calendario institucional** para el ecosistema Velty: cuándo está abierto el establecimiento y cuándo está cerrado, independientemente del horario de cualquier profesional.

Posee tres cosas:

1. **Horarios de atención (`business_hours`)** — la ventana semanal de apertura, una fila por día de la semana.
2. **Feriados (`holidays`)** — cierres fechados con un origen legal/administrativo, editables en tiempo de ejecución (un nuevo feriado aparece por ley; nadie recompila).
3. **Cierres (`closures`)** — cierres fechados con un origen local (mantenimiento, un evento).

**No** posee la agenda de ningún profesional. `appointment_booking` posee eso y consume este módulo a través del contrato de solo lectura `OpeningHours`.

> Reemplaza al archivado `github.com/veltylabs/business_hours`, cuya codificación de texto de hora de reloj ("HH:MM") se corrige aquí a **24h clock string (HH:MM)** —la misma codificación que usa `appointment_booking` para los bloques de trabajo— y cuyos nombres de días en español prefijados han desaparecido (la traducción vive en la aplicación consumidora).

## Inicio rápido

```go
// En tiempo de despliegue, ejecutar las migraciones de esquema:
err := migrate.Migrate(conn, ddlCompiler)
if err != nil { /* ... */ }

// Al iniciar el proceso:
cal, err := businesscalendar.New(db, businesscalendar.Deps{
    IDs: idGenerator, // model.IDGenerator — requerido
    Publisher: broker, // events.Publisher — opcional; nil deshabilita la publicación
})
if err != nil { /* ... */ }

crud.Mount(reg, nil, nil, cal) // monta las operaciones que declara definition.go

bounds, err := cal.DayLimits(dateMidnightUTC)
```

`New` asume que el esquema de la base de datos (`business_hours`, `holiday`, `closure`) ya existe. La aplicación que despliega llama a `migrate.Migrate(conn, compiler)` una vez, en tiempo de despliegue. Esta migración utiliza `ddl.Sync` para crear las tablas que faltan y agregar nuevas columnas si el modelo fue actualizado, ver flujo: https://github.com/webtyp/ddl/blob/main/docs/diagrams/SYNC_FLOW.md.

## Operaciones

Las declara `definition.go` (`Allow` y `Rules`); `crud` las monta. Una operación es `router.Operation{Module: ModelName, Entity: <entidad>, Action: <acción>}`.

| Entidad | Acciones | Regla del módulo |
|----|----|----|
| `business_hours` | create, read, update | un día de la semana por fila; `open_min` < `close_min`; un día cerrado guarda 00:00–00:00 |
| `holiday` | create, read, delete | una fecha por feriado (repetida → conflicto) |
| `closure` | create, read, delete | una fecha por cierre (repetida → conflicto) |
| `day_detail` (lectura) | read con filtro `{date}` | si el establecimiento está abierto esa fecha, entre qué minutos y por qué está cerrado |

API: `BankDayAfter(desde, días)`: día hábil bancario N días después (lunes a viernes sin feriados; el horario de la clínica no cuenta).

Nota: los listados cargan según declara su `view.Page` y buscan en el servidor.

Los tiempos se almacenan como **24h clock string (HH:MM)**; `day_detail` responde "si el establecimiento está abierto esta fecha y entre qué minutos".


## OpeningHours

Un módulo vecino (como `appointment_booking` o `room_layout`) declara en su propio `interfaces.go` el contrato que necesita:

```go
type OpeningHours interface {
    DayLimits(date time.Unix) (time.DayLimits, error)
    WeekdayLimits(dayOfWeek int) (time.DayLimits, error)
    ClosedDates(from, to time.Unix) (holidays, closures []string, err error)
}
```

`*Module` lo satisface estructuralmente. `time.DayLimits` lleva `Open`, `OpenMin` y `CloseMin`. Precedencia de resolución: **Cierre local > Feriado > Regla semanal**.

## Eventos

Cada escritura exitosa publica `EventCalendarChanged` (`business.calendar.changed`) con un `CalendarChangedPayload` tipado que indica *qué* cambió, su rango de fechas afectado y `Closed` —la dirección. `Closed` es `true` solo cuando el cambio cierra el tiempo (un feriado agregado, horarios reducidos), que es la única dirección que puede invalidar una reserva existente.

## Claves de traducción

Este módulo no renderiza ningún lenguaje humano codificado de forma rígida. Introduce estas claves canónicas en inglés, traducidas por la aplicación consumidora a través de `webtyp.com/fmt/lang`: `Business hours`, `Holidays`, `Closures`, `Open`, `Closed`, más los siete nombres de días de la semana de `webtyp.com/date` (`date.WeekdayName`).

## View and demo

The module exports its own UI view, seed demo data, and executable WASM demo shell:

- `ui/` — exports `ui.ID`, `ui.Label`, and `ui.Screens(caller router.Caller) []view.Screen`.
- `seed/` — `seed.Load(module)` writes initial weekly business hours and Chilean holidays through domain service methods.
- `seed/demo.go` — the runnable demo (`seed.Demo()`). Run `webtyp` at the repository root to launch it (in-browser, in-memory, no login).

## Archivos clave

| Archivo | Rol |
|------|------|
| `definition.go` | Definiciones de las entidades (`Allow`, `Rules`), errores, tópico/payload de eventos |
| `definition_gen.go` | Generado por `definitionc` — **no editar** |
| `checks.go`, `rows.go`, `store.go` | Reglas compartidas, lectura de filas y repositorios |
| `rule_*.go` | Una regla de negocio por acción declarada |
| `module.go` | `Module`, `Deps`, `New`, métodos de servicio, publicación de eventos |
| `migrate/` | Migración de esquema en tiempo de despliegue (`Migrate`) |

| `view.go` | Tres presentadores (`NewBusinessHoursView`, `NewHolidaysView`, `NewClosuresView`) |
| `ui/` | Vista UI del módulo (`ui.Screens(caller router.Caller) []view.Screen`, `ui.ID`, `ui.Label`) |
| `seed/` | Datos semilla de demostración (`seed.Load`) |
| `tests/` | Pruebas de casos de uso, pruebas de ops, conformidad (sobre `storage/mem` + `crud.LoopbackActingAs`) |

## Documentación

- [ARCHITECTURE.md](docs/ARCHITECTURE.md) — alcance del dominio, patrones, ops, ejemplo de raíz de composición.
- [Diagrama de base de datos](docs/diagrams/database.md) — Mermaid ERD.

---

*Este módulo forma parte de la colección de módulos de Velty Labs.*

errores tipados (`model.NotFound`, `model.Conflict`, `model.Invalid`) que `crud` traduce al estado que corresponde; permisos por entidad (`model.ResourceOf`); claves foráneas declaradas.
