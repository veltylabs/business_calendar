# Arquitectura de business_calendar

## 1. Alcance del dominio

El **calendario institucional**: cuándo está abierto el establecimiento y cuándo está cerrado, independientemente del horario de cualquier profesional. Una preocupación delimitada —*cuándo está disponible el establecimiento*— propiedad de este módulo, consumida en modo de solo lectura por `appointment_booking` (que posee *cuándo está disponible un profesional*: un propietario diferente, un editor diferente, un permiso diferente).

Posee tres entidades:

- **BusinessHours** — una fila por día de la semana (0=Domingo … 6=Sábado), `day_of_week` ÚNICO. Los tiempos son **minutos desde la medianoche** (`0..1439`), la misma codificación que utiliza `appointment_booking` para los bloques de trabajo.
- **Holiday** — un cierre fechado con un origen legal/administrativo (`name`), editable en tiempo de ejecución (un nuevo feriado aparece por ley; nadie recompila).
- **Closure** — un cierre fechado decidido por el propio establecimiento (`reason`): mantenimiento, un evento.

### Por qué los feriados y los cierres son tablas separadas

El **origen** es lo que muestra la interfaz de usuario y por lo que filtra un administrador. Colapsar ambos en una tabla con una columna `type` convertiría el origen en una cadena mágica; dos tablas lo convierten en un `ClosedReason` de primera clase y buscable. Ambos están fechados, pero un cierre local **supera** a un feriado en la resolución (a continuación): la decisión más específica y reciente gana, y un feriado en el que el establecimiento decidió trabajar no debe reabrirse eliminando el cierre.

## 2. No multitenant (decisión aprobada)

A diferencia de la mayoría de los módulos de este ecosistema, `business_calendar` **no tiene un campo `tenant_id`** en ninguna tabla: el calendario institucional es un esquema global único por aplicación de raíz de composición, no delimitado por inquilino (tenant). Esta es una decisión deliberada y explícita (según `AGENTS.md` "Notas específicas del dominio"); no agregue `tenant_id` como un cambio casual —eso sería un cambio de dominio, no un cambio de adopción del arnés.

Debido a que no hay columna de inquilino, las condiciones `UPDATE`/`DELETE` se basan en el `id` de la fila (y `day_of_week` para los horarios), lo cual es seguro para un singleton de un solo inquilino. La preocupación de escritura entre inquilinos que protege `tenant_id` no existe aquí.

## 3. Patrones aplicados

Acoplado únicamente a los puertos publicados en `webtyp.com/*`, nunca a infraestructura concreta —consulte `AGENTS.md` (raíz del repositorio) para la lista blanca/lista negra:

- **`orm.DB` para almacenamiento** — independiente del backend sobre cualquier `storage.Conn` que inyecte la raíz de composición (`storage/mem` en las pruebas de este módulo).
- **`ddl`** para la migración de esquema del propio módulo a través de `migrate.Migrate(conn, ddlCompiler)` en tiempo de despliegue —separado deliberadamente de `New()` e aislado en el subpaquete `migrate`.
- **`router.OperationModule`** (`ModelName()` + `MountOperations`) para transporte —el módulo nunca ve un servidor concreto o `net/http`.
- **`model.IDGenerator`** para identidad (`Deps.IDs`, requerido —el módulo nunca construye uno).
- **`events.Publisher`** (`Deps.Publisher`, opcional —nil lo deshabilita silenciosamente) para `EventCalendarChanged` después de cada escritura exitosa.
- **`view.Presenter`** (`NewBusinessHoursView`/`NewHolidaysView`/`NewClosuresView(caller router.Caller)`) construido únicamente con `view`+`model`+`router` (+ `date` y `fmt/lang` para traducción) —la aplicación elige el renderizador.

### Semántica de NotNull y dónde vive la validación

En un tipo base `model.Int()`/`model.Bool()`, `NotNull` rechaza el **valor cero** (`ValidateFields`: `NotNull && IsZeroPtr`), por lo que `day_of_week` 0 (Domingo), `is_open == false` y la medianoche (`0` minutos) son todos valores legítimos que no deben rechazarse. Por lo tanto, los registros del dominio llevan `NotNull` solo para impulsar el DDL `NOT NULL`; los invariantes *semánticos* viven en los métodos de servicio —el único punto de control a prueba de fallos que cruza cada ruta de escritura:

| Invariante | Centinela |
|-----------|----------|
| `day_of_week` en 0..6 | `ErrInvalidDay` |
| ventana abierta dentro de 0..1439, `open_min < close_min` | `ErrInvalidMinutes` |
| segundo feriado/cierre en la misma fecha | `ErrDuplicateDate` |
| fila faltante | `ErrNotFound` |

Los argumentos de transporte repiten los campos de **texto** requeridos con `NotNull` (nombre de feriado `name`, razón de cierre `reason`, `id` de eliminación) para que el límite de la operación rechace la entrada vacía con un 400 antes de llegar al servicio; cada operación que muta se canaliza a través de las verificaciones de dominio del servicio.

## 4. `GetDayBounds`/`GetDayDetail` y precedencia de resolución

> **Enmendado el 2026-09-08.** El puerto originalmente devolvía un `DayBounds` local del módulo que llevaba `ClosedBy`. Eso forzaba a cualquier consumidor a importar este módulo solo para nombrar el tipo —exactamente el acoplamiento que `veltylabs/appointment_booking` no debe tener. Consulte `webtyp/docs/AGENDA_DOMAIN_MASTER_PLAN.md` §3-bis para ver la justificación completa.

`interfaces.go` divide la respuesta en dos, según la audiencia:

```go
// El valor que consume un módulo de dominio vecino (por ejemplo, appointment_booking).
// Declara su PROPIA interfaz devolviendo tinytime.DayBounds —un tipo neutral en
// webtyp/time que ambas partes ya importan— y este Module la satisface
// estructuralmente. Sin importación en ninguna dirección, sin adaptador.
func (m *Module) GetDayBounds(date int64) (tinytime.DayBounds, error)

// La respuesta más rica de este módulo: los mismos límites MÁS por qué un día cerrado
// está cerrado. Utilizado por la operación get_day_bounds y la propia interfaz de usuario de este módulo.
func (m *Module) GetDayDetail(date int64) (DayDetail, error)

type DayDetail struct {
    Bounds   tinytime.DayBounds
    ClosedBy ClosedReason // "" cuando Bounds.Open es true
}
```

`Reader` (declarado aquí) es un **alias de conveniencia para los consumidores de este propio módulo**, no el contrato que nombra un módulo vecino; un vecino declara su propia interfaz con una forma idéntica a la firma de `GetDayBounds` y este `Module` la satisface estructuralmente:

```go
type Reader interface {
    GetDayBounds(date int64) (tinytime.DayBounds, error)
}
```

`GetDayBounds` es un envoltorio ligero: `GetDayDetail` es la única ruta de resolución; `GetDayBounds` lo llama y descarta `ClosedBy`. Hay exactamente un lugar donde se decide la precedencia.

Resolución, en este orden exacto (sin cambios por la enmienda):

1. Un `Closure` en esa fecha → `{Open: false, ClosedBy: ClosedLocal}`.
2. Un `Holiday` en esa fecha → `{Open: false, ClosedBy: ClosedHoliday}`.
3. `BusinessHours` para ese día de la semana con `is_open == false` (o sin fila alguna —la ausencia de un horario equivale a cerrado) → `{Open: false, ClosedBy: ClosedWeekly}`.
4. De lo contrario, `{Open: true, OpenMin, CloseMin}`.

## 5. Eventos

Cada mutación exitosa publica `EventCalendarChanged` con un `CalendarChangedPayload` completamente poblado (implementa `model.Encodable`):

| Método | Tipo (Kind) | Cerrado | Rango |
|--------|------|--------|-------|
| `AddHoliday` | `HOLIDAY` | true | `[date, date]` |
| `RemoveHoliday` | `HOLIDAY` | false | `[date, date]` |
| `AddClosure` | `CLOSURE` | true | `[date, date]` |
| `RemoveClosure` | `CLOSURE` | false | `[date, date]` |
| `UpsertBusinessHours` (nueva fila) | `BUSINESS_HOURS` | false | `0,0` |
| `UpsertBusinessHours` (reducción) | `BUSINESS_HOURS` | true | `0,0` |
| `UpsertBusinessHours` (ampliación) | `BUSINESS_HOURS` | false | `0,0` |

`Closed` es el campo en el que confía un suscriptor: `true` solo cuando el cambio cierra el tiempo (un feriado agregado, horarios reducidos), la única dirección que puede invalidar una reserva existente. En caso de duda, el módulo publica `Closed: true` (un recálculo es económico; omitir un cierre real deja desamparado a un paciente en un día cerrado). Una regla semanal no tiene un rango acotado, por lo que los eventos de horarios de atención llevan `0,0` y un suscriptor recalcula todo su horizonte.

## 6. Operaciones (Ops)

| Op | Acción | Recurso | Descripción |
|----|--------|----------|--------------|
| `list_business_hours` | `r` | `business_hours` | Las 7 filas de días de la semana, por `day_of_week` |
| `upsert_business_hours` | `cr` | `business_hours` | Crear o actualizar una ventana de día de la semana |
| `get_day_bounds` | `r` | `business_hours` | `DayDetail` para una fecha (límites + `ClosedBy`) |
| `list_holidays` | `r` | `holiday` | Todos los feriados |
| `add_holiday` | `c` | `holiday` | Registrar un feriado (fecha duplicada → 409) |
| `remove_holiday` | `d` | `holiday` | Eliminar por id (faltante → 404) |
| `list_closures` | `r` | `closure` | Todos los cierres |
| `add_closure` | `c` | `closure` | Registrar un cierre (fecha duplicada → 409) |
| `remove_closure` | `d` | `closure` | Eliminar por id (faltante → 404) |

Mapeo de estados: `400` decodificación/validación (`ErrInvalidDay`/`ErrInvalidMinutes`) · `404` no encontrado (`ErrNotFound`) · `409` conflicto (`ErrDuplicateDate`) · solo errores internos reales `500`.

La operación de upsert declara `model.Create|model.Update` (una máscara de bits) porque realmente puede hacer ambas cosas; declarar solo `Update` permitiría que un principal con permisos solo de actualización cree filas. Tres recursos (`business_hours`, `holiday`, `closure`) porque un administrador que puede agregar un feriado no es necesariamente el que puede reescribir los horarios de apertura.

El filtrado por rango (`ListHolidays(from, to)`, `ListClosures(from, to)`) vive en los métodos de **servicio** para consumidores dentro del proceso; las **operaciones** de lista no toman argumentos y devuelven todo el horizonte (una vista lista todo).

## 7. Ejemplo de raíz de composición

```go
cal, err := businesscalendar.New(db, businesscalendar.Deps{
    IDs:      unixid.NewUnixID(), // inyectado —el módulo nunca construye uno
    Publisher: broker,            // events.Broker, o nil
})
if err != nil { /* ... */ }

cal.MountOperations(opRegistry) // router.OperationRegistry

// appointment_booking nunca importa este paquete —declara su propia
// interfaz con la forma de la firma de GetDayBounds, y *cal la satisface
// estructuralmente. El alias Reader de este módulo es para los usuarios de ESTE módulo:
var reader businesscalendar.Reader = cal
bounds, err := reader.GetDayBounds(midnightUTC) // tinytime.DayBounds —sin ClosedBy

detail, err := cal.GetDayDetail(midnightUTC) // DayDetail —CON ClosedBy, interfaz propia del módulo

hoursView := businesscalendar.NewBusinessHoursView(caller) // router.Caller
holidaysView := businesscalendar.NewHolidaysView(caller)
closuresView := businesscalendar.NewClosuresView(caller)
```

## 8. Esquema

Consulte [`docs/diagrams/database.md`](diagrams/database.md).
