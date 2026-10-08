---
PLAN: "fix: detect sentinel errors without == between interfaces (no reflection in wasm)"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 11604534656909899668
---

# Plan — `business_calendar`: errores centinela sin `==` entre interfaces

> Master: `webtyp/docs/NO_REFLECTION_MASTER_PLAN.md` (ola 3). Doctrina: skill `api-design`.
> **Prerrequisito:** `go get webtyp.com/orm@latest` y confirmar que existe `orm.IsNotFound`. Si falta alguna, parar y reportarlo: no implementar un sustituto local.

## 1. El problema

En TinyGo, `==`, `!=` y `switch` entre valores de interfaz compilan a `runtime.interfaceEqual`, que
llama a `reflectValueEqual(reflectlite.ValueOf(x), reflectlite.ValueOf(y))`. `error` es una interfaz:
cada `err == ErrX` mete `internal/reflectlite` (~9 KB) en el binario wasm. La regla del dueño es que
el código que compila a wasm no use reflexión nunca. `errors.Is`/`errors.As` tampoco sirven: también
usan reflectlite.

## 2. La corrección — dos patrones, ninguno más

**A. Centinelas de otros paquetes** — usar su función de consulta:

| Antes | Después |
|---|---|
| `err == orm.ErrNotFound` | `orm.IsNotFound(err)` |
| `err != orm.ErrNotFound` | `!orm.IsNotFound(err)` |
| `err == storage.ErrNoRows` | `storage.IsNoRows(err)` |

**B. Centinelas propios de este paquete** — un tipo string no exportado; se afirma una vez y se
compara el valor concreto (comparación de strings, sin reflexión):

```go
// domainError is the concrete type of this package's sentinel errors. Code
// compares them by asserting this type and comparing the value: == between two
// error values compiles, under TinyGo, to runtime.interfaceEqual, which pulls
// internal/reflectlite into the wasm binary.
type domainError string

func (e domainError) Error() string { return string(e) }

const (
	ErrNotFound domainError = "<texto actual>"
	// … uno por centinela, con su texto actual
)
```

- `<texto actual>`: el string exacto que devuelve hoy el centinela (`fmt.Err("a", "b")` une las
  palabras con un espacio: `"a b"`). Un test fija cada texto: los mensajes no cambian.
- Uso, por ejemplo al traducir errores a códigos:

```go
if e, ok := err.(domainError); ok {
	switch e {
	case ErrFloorInUse, ErrRoomOverlap:
		return conflict
	case ErrNotFound:
		return notFound
	}
}
if orm.IsNotFound(err) {
	return notFound
}
```

- Un `switch err { case ErrA: … }` pasa a `if e, ok := err.(domainError); ok { switch e { … } }`.
- Si un centinela propio se envuelve antes de compararlo (`fmt.Errf("…%v", ErrX)`), la comparación
  con `==` ya no funcionaba: dejarlo igual y anotarlo en el PR, no inventar otra detección.

## 3. Sitios a cambiar (inventario del 2026-10-08)

### Código de producción

- `ops.go:79` — `switch err {`
- `ops.go:80` — `case ErrNotFound:`
- `ops.go:82` — `case ErrDuplicateDate:`
- `ops.go:84` — `case ErrInvalidDay, ErrInvalidMinutes:`
- `interfaces.go:80` — `if err != orm.ErrNotFound {`
- `interfaces.go:89` — `if err != orm.ErrNotFound {`
- `interfaces.go:95` — `if err == ErrNotFound {`
- `module.go:63` — `if err == orm.ErrNotFound {`
- `module.go:104` — `if err != nil && err != ErrNotFound {`
- `module.go:107` — `if err == ErrNotFound {`
- `module.go:153` — `if err != orm.ErrNotFound {`
- `module.go:170` — `if err == orm.ErrNotFound {`
- `module.go:216` — `if err != orm.ErrNotFound {`
- `module.go:233` — `if err == orm.ErrNotFound {`

### Centinelas propios de este repo (patrón B)

- `model.go:64` — `ErrNotFound       = fmt.Err("business_calendar: record not found")`
- `model.go:65` — `ErrInvalidDay     = fmt.Err("business_calendar: day_of_week must be 0..6")`
- `model.go:66` — `ErrInvalidMinutes = fmt.Err("business_calendar: minutes must be 0..1439 and open_min < close_min")`
- `model.go:67` — `ErrDuplicateDate  = fmt.Err("business_calendar: a record already exists for that date")`

### Tests (se migran igual: un solo camino también en los tests)

- `tests/closures_test.go:99` — `if err != businesscalendar.ErrDuplicateDate {`
- `tests/closures_test.go:106` — `if err := m.RemoveClosure("does-not-exist"); err != businesscalendar.ErrNotFound {`
- `tests/holidays_test.go:120` — `if err != businesscalendar.ErrDuplicateDate {`
- `tests/holidays_test.go:127` — `if err := m.RemoveHoliday("does-not-exist"); err != businesscalendar.ErrNotFound {`
- `tests/business_hours_test.go:110` — `if err := m.UpsertBusinessHours(c); err != businesscalendar.ErrInvalidMinutes {`
- `tests/business_hours_test.go:121` — `if err != businesscalendar.ErrInvalidDay {`

Si encuentras otro `==`/`!=`/`switch` entre valores de interfaz con operandos no nil que no esté en la
lista, se migra igual. `x == nil` y `x != nil` están bien.

## 4. Tests

- Todos los tests existentes siguen verdes sin cambiar su intención.
- Un test que fija el `Error()` de cada centinela propio convertido (patrón B) contra su texto anterior.
- Si el paquete traduce errores a códigos/respuestas (por ejemplo en `ops.go`), un test por rama
  cambiada: el mismo error produce el mismo código que antes.
- `gotest` verde (vet, race, tests, wasm).

## 5. Criterios de aceptación

- `grep -rnE '(==|!=) *[A-Za-z_.]*Err[A-Za-z]*' --include=*.go . | grep -v '_temp/'` → vacío.
- `grep -rn 'switch err {' --include=*.go .` → vacío.
- `grep -rn 'errors.Is\|errors.As' --include=*.go .` → vacío.
- Ningún símbolo exportado nuevo: `git diff | grep '^+func [A-Z]'`.
- `gotest` verde.

## 6. Restricciones

Las de `AGENTS.md`, más: nada de `reflect`, `unsafe`, `errors.Is`/`errors.As`, ni `==`/`!=`/`switch`
entre valores de interfaz con operandos no nil. No tocar otros repos.
