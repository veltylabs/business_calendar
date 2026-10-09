---
PLAN: "feat(business_calendar): GetWeekdayBounds — the regular hours of a weekday, without holidays or closures"
EXECUTOR: jules
REVIEWER: none
---

# Plan — `GetWeekdayBounds(dayOfWeek)`

> Master: `webtyp/docs/NO_REFLECTION_MASTER_PLAN.md` (bloqueo de la ola 3 en `appointment_booking`).
> `veltylabs/appointment_booking` espera el tag de este plan para su propio plan.

## 1. El problema (bug de producto, no de test)

`appointment_booking.SaveDayBlocks` valida la plantilla **semanal** de horario de un funcionario
contra una sola fecha: la próxima ocurrencia de ese día de la semana, con `GetDayBounds(date)`.
`GetDayBounds` aplica feriados y cierres (pasos 1 y 2 de `GetDayDetail`). Consecuencia: el
2026-10-08 (jueves) el próximo lunes era el 2026-10-12, feriado → **toda esa semana fue imposible
guardar el horario de los lunes**, aunque los lunes normales se trabaje. Lo mismo ocurre la semana
previa a cualquier feriado o cierre local. Lo destapó `appointment_booking/tests/seed_test.go`.

Un feriado es una excepción de una fecha, no la forma de la semana. Una plantilla semanal se valida
contra el horario semanal (paso 3–4 de `GetDayDetail`); los feriados siguen bloqueando las reservas
de esa fecha concreta, que es donde corresponden.

## 2. Design gate (api-design)

1. **Antecedentes.** Google Business Profile separa `regularHours` (semana tipo) de
   `specialHours` (fechas excepcionales); Schema.org `OpeningHoursSpecification` modela lo mismo con
   `dayOfWeek` vs `validFrom/validThrough`; las agendas de Calendly y Cal.com validan la
   disponibilidad semanal contra horas regulares y aplican las fechas especiales como overrides. Este
   módulo ya tiene las dos capas guardadas (`BusinessHours` vs `Holiday`/`Closure`) pero solo expone
   la combinada.
2. **Nombre.** `GetWeekdayBounds(dayOfWeek int)`: par de `GetDayBounds(date)`, misma forma de
   respuesta (`tinytime.DayBounds`), distinto argumento. Se lee "límites del día de la semana".
3. **Balance.** Conceptos +1 · archivos que el consumidor toca: su interfaz local (+1 método) ·
   formas de obtener las horas regulares de un día: hoy 0 vía el contrato neutral, después 1.
4. **Dónde va.** `business_calendar`, dueño de `BusinessHours`. Respeta AGENDA_DOMAIN §3-bis: el
   vecino declara su propia interfaz con `tinytime.DayBounds` y este `Module` la satisface
   estructuralmente, sin importarse.
5. **Qué borra.** Nada aquí. En `appointment_booking` (su plan): `nextWeekday` y la validación por
   fecha representativa.

## 3. La corrección

En `interfaces.go`:

```go
// GetWeekdayBounds answers "between which minutes does the establishment
// normally work on this weekday" (0 = Sunday … 6 = Saturday) — the weekly
// template only: holidays and local closures are date exceptions and are NOT
// applied here (GetDayBounds applies them). A consumer validating a weekly
// schedule uses this; one validating a concrete date uses GetDayBounds.
//
// A weekday with no business_hours row, or with is_open == false, is closed
// (closed by default, same rule as GetDayDetail step 3).
func (m *Module) GetWeekdayBounds(dayOfWeek int) (tinytime.DayBounds, error)
```

- Implementación: exactamente los pasos 3–4 de `GetDayDetail`, reusando `businessHoursByDay`. Para
  no duplicar, extraer esos pasos a una función privada que usen los dos (`GetDayDetail` llama a la
  nueva para su tramo final). Mismos campos (`Open`, `OpenMin`, `CloseMin`) que hoy produce el paso 4.
- `dayOfWeek` fuera de 0..6 → error con un centinela nuevo del patrón `domainError` del paquete
  (`ErrInvalidWeekday`, texto `"business_calendar: day of week must be 0..6"`).
- Agregar el método a la interfaz `Reader` (con su comentario), junto a `GetDayBounds`.

## 4. Tests (rojo primero, en `tests/`)

- Lunes con `BusinessHours{DayOfWeek: 1, IsOpen: true, 480–1080}` **y** un feriado el próximo lunes:
  `GetWeekdayBounds(1)` → `{Open: true, 480, 1080}`; `GetDayBounds(<ese lunes>)` → cerrado (sin
  cambios de comportamiento).
- Domingo con `IsOpen: false` → `GetWeekdayBounds(0)` cerrado. Día sin fila → cerrado.
- `GetWeekdayBounds(7)` y `(-1)` → `ErrInvalidWeekday`.
- Un cierre local (`Closure`) en una fecha no afecta `GetWeekdayBounds` de ese día de la semana.
- Los tests existentes de `GetDayDetail`/`GetDayBounds` siguen verdes (la extracción no cambia nada).
- `gotest` verde (vet, race, tests, wasm).

## 5. Criterios de aceptación

- Exportados nuevos: `GetWeekdayBounds`, `ErrInvalidWeekday`.
- `GetDayDetail` y `GetWeekdayBounds` comparten una sola implementación del paso semanal.
- `docs/ARCHITECTURE.md` describe las dos preguntas (fecha vs día de la semana) y cuándo usar cada una.
- `gotest` verde.

## 6. Restricciones

Las de `AGENTS.md`. Además: nada de `reflect`, `errors.Is`/`errors.As`, ni `==`/`!=`/`switch` entre
valores de interfaz con operandos no nil (detectar centinelas con aserción de tipo, como ya hace este
paquete con `domainError`). No tocar otros repos.
