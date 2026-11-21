# Bugs de UI y mejoras — `business_calendar`

> Auditoría en vivo sobre `#business_calendar.hours`, `#business_calendar.holidays`, `#business_calendar.closures` (2026-11-19).

## Bugs encontrados

| # | Sev | Pantalla | Qué se ve en pantalla | Causa raíz verificada |
|---|---|---|---|---|
| BC-1 | 🔴 | `#business_calendar.hours` | **Ambos campos de hora muestran el mismo rótulo `Formato de hora: HH:MM`** — no dice cuál es apertura y cuál es cierre. | `business_calendar/definition.go:16-17` declara `Label: "Opens at"` y `Label: "Closes at"`, pero `webtyp/form/form.go:375` prioriza `inp.GetTitle()` y `webtyp/input/hour.go:20` fija `SetTitle("time format: HH:MM")` pisando el `Label` del modelo (bug de framework F2). |
| BC-2 | 🔴 | `#business_calendar.hours` | **`Día de la semana` es un campo numérico crudo (`0`..`6`)** y la lista muestra `0`, `1`, `2`, `3`, `4`, `5`, `6` en lugar de los nombres de los días (`Domingo`..`Sábado`). | `BusinessHoursModel` (`business_calendar/definition.go:15`) define `{Name: "day_of_week", Type: input.Number(), Item: model.ItemLabel}` en lugar de opciones nombradas o un selector/etiqueta legible. |
| BC-3 | 🔴 | `#business_calendar.holidays`, `#business_calendar.closures` | **El campo `Fecha` es un número entero (`0`) y la lista de feriados muestra timestamps Unix crudos** (`1767225600`, `1775174400`, `1775260800`). | `HolidayModel` y `ClosureModel` (`business_calendar/definition.go:30,43`) declaran `{Name: "specific_date", Type: input.Number(), Item: model.ItemDescription}` en vez de `input.Date()` o un tipo fecha formateado para humanos. |
| BC-4 | 🟡 | `#business_calendar.hours` | **Badges de horario truncados en la lista** (`00:00 · 00:0...`, `08:00 · 17:3...`). | El texto concatenado en `ItemDescription` excede el ancho del badge en `targetlist`. |
| BC-5 | ⚪ | `#business_calendar.closures` | **Lista vacía `0 / 0` sin guía ni explicación** de para qué sirven los cierres vs feriados. | Falta estado vacío descriptivo en `crudview`. |

## Sugerencias de mejora justificadas

1. **Reemplazar `input.Number()` de `specific_date` por `input.Date()` (y formatear `YYYY-MM-DD` / `DD/MM/YYYY` en la lista de Feriados y Cierres):** *Justificación:* ningún usuario administrativo puede calcular ni leer epoch Unix (`1767225600`); es inutilizable en clínica hoy.
2. **Mostrar nombres de días (`Domingo`..`Sábado`) y volver `day_of_week` un selector o ficha de solo lectura con 7 días fijos (ordenados Lunes→Domingo):** *Justificación:* los 7 días de la semana son fijos; no debería existir botón `+` para crear un día 7 u 8, ni mostrar `0..6`.
3. **Unificar `Feriados` y `Cierres` o mejorar el badge para que muestre `08:00–17:30` o `Cerrado` sin truncar:** *Justificación:* reduce la altura de `08:00 · 17:3...` y hace legible el estado «Abierto/Cerrado» directamente en la lista.
