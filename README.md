# business_calendar
<img src="docs/img/badges.svg">

El **calendario institucional** para el ecosistema Velty: cuándo está abierto el establecimiento y cuándo está cerrado, independientemente del horario de cualquier profesional.

Posee tres cosas:

1. **Horarios de atención (`business_hours`)** — la ventana semanal de apertura, una fila por día de la semana.
2. **Feriados (`holidays`)** — cierres fechados con un origen legal/administrativo, editables en tiempo de ejecución (un nuevo feriado aparece por ley; nadie recompila).
3. **Cierres (`closures`)** — cierres fechados con un origen local (mantenimiento, un evento).

**No** posee la agenda de ningún profesional. `appointment_booking` posee eso y consume este módulo a través del contrato de solo lectura [`Reader`](#el-puerto-reader).

> Reemplaza al archivado `github.com/veltylabs/business_hours`, cuya codificación de texto de hora de reloj ("HH:MM") se corrige aquí a **minutos desde la medianoche** —la misma codificación que usa `appointment_booking` para los bloques de trabajo— y cuyos nombres de días en español prefijados han desaparecido (la traducción vive en la aplicación consumidora).

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

cal.MountOperations(opRegistry) // router.OpRegistry

reader := businesscalendar.Reader(cal) // el puerto que consume un módulo vecino
bounds, err := reader.GetDayBounds(dateMidnightUTC)
```

`New` asume que el esquema de la base de datos (`business_hours`, `holiday`, `closure`) ya existe. La aplicación que despliega llama a `migrate.Migrate(conn, compiler)` una vez, en tiempo de despliegue.

## Operaciones (Ops)

| Op | Recurso | Acción | Args |
|----|----------|--------|------|
| `list_business_hours` | `business_hours` | read | — |
| `upsert_business_hours` | `business_hours` | create \| update | `day_of_week`, `open_min`, `close_min`, `is_open`, `notes` |
| `get_day_bounds` | `business_hours` | read | `date` (medianoche UTC, segundos) |
| `list_holidays` | `holiday` | read | — |
| `add_holiday` | `holiday` | create | `specific_date`, `name`, `notes` |
| `remove_holiday` | `holiday` | delete | `id` |
| `list_closures` | `closure` | read | — |
| `add_closure` | `closure` | create | `specific_date`, `reason` |
| `remove_closure` | `closure` | delete | `id` |

Los tiempos son **minutos desde la medianoche** (`0..1439`); `get_day_bounds` responde "si el establecimiento está abierto esta fecha y entre qué minutos".

## El puerto Reader

`interfaces.go` declara el puerto de solo lectura del que depende un vecino:

```go
type Reader interface {
    GetDayBounds(date int64) (DayBounds, error)
}
```

`*Module` lo satisface estructuralmente. `DayBounds` lleva `Open`, `OpenMin`, `CloseMin` y `ClosedBy` —un `ClosedReason` que nombra el origen (`WEEKLY` | `HOLIDAY` | `CLOSURE`). Precedencia de resolución: **Cierre local > Feriado > Regla semanal**.

## Eventos

Cada escritura exitosa publica `EventCalendarChanged` (`business.calendar.changed`) con un `CalendarChangedPayload` tipado que indica *qué* cambió, su rango de fechas afectado y `Closed` —la dirección. `Closed` es `true` solo cuando el cambio cierra el tiempo (un feriado agregado, horarios reducidos), que es la única dirección que puede invalidar una reserva existente.

## Claves de traducción

Este módulo no renderiza ningún lenguaje humano codificado de forma rígida. Introduce estas claves canónicas en inglés, traducidas por la aplicación consumidora a través de `webtyp.com/fmt/lang`: `Business hours`, `Holidays`, `Closures`, `Open`, `Closed`, más los siete nombres de días de la semana de `webtyp.com/date` (`date.WeekdayName`).

## View and demo

The module exports its own UI view, seed demo data, and executable WASM demo shell:

- `ui/` — exports `ui.ID`, `ui.Label`, and `ui.Browser(caller, ids, tenantID)`.
- `seed/` — `seed.Load(module)` writes initial weekly business hours and Chilean holidays through domain service methods.
- `web/` — runnable in-browser demo (`client.go`). Run `webtyp` at the repository root to launch the demo (in-browser, in-memory, no login).

## Archivos clave

| Archivo | Rol |
|------|------|
| `model.go` | Definiciones (registros + args de transporte), errores, tópico/payload de eventos |
| `model_orm.go` | Generado por `ormc` — **no editar** |
| `module.go` | `Module`, `Deps`, `New`, métodos de servicio, publicación de eventos |
| `migrate/` | Migración de esquema en tiempo de despliegue (`Migrate`) |
| `interfaces.go` | El puerto `Reader` + precedencia de `GetDayBounds` |
| `ops.go` | Constantes de Op, `MountOperations`, manejadores |
| `view.go` | Tres presentadores (`NewBusinessHoursView`, `NewHolidaysView`, `NewClosuresView`) |
| `ui/` | Vista UI del módulo (`ui.Browser`, `ui.ID`, `ui.Label`) |
| `seed/` | Datos semilla de demostración (`seed.Load`) |
| `web/` | Demo ejecutable en el navegador (`web/client.go`) |
| `tests/` | Pruebas de casos de uso, pruebas de ops, conformidad (sobre `storage/mem` + `router/mock`/`loopback`) |

## Documentación

- [ARCHITECTURE.md](docs/ARCHITECTURE.md) — alcance del dominio, patrones, ops, ejemplo de raíz de composición.
- [Diagrama de base de datos](docs/diagrams/database.md) — Mermaid ERD.

---

*Este módulo forma parte de la colección de módulos de Velty Labs.*
