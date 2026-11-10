| Test | Rule |
|---|---|
| [TestBankDayAfter](../tests/bankdays_test.go) | |
| [TestAdminDefinesWeeklyBusinessHours](../tests/business_hours_test.go) | |
| [TestWideningHoursIsVisibleToReaderImmediately](../tests/business_hours_test.go) | |
| [TestUpsertBusinessHours_NarrowingCloses](../tests/business_hours_test.go) | |
| [TestInvalidMinutesRejected](../tests/business_hours_test.go) | |
| [TestInvalidDayRejected](../tests/business_hours_test.go) | |
| [TestListBusinessHours_EmptyIsNotError](../tests/business_hours_test.go) | |
| [TestClosedDates](../tests/closedates_test.go) | |
| [TestClosedDatesEmpty](../tests/closedates_test.go) | |
| [TestLocalClosureIsDistinguishableFromHoliday](../tests/closures_test.go) | |
| [TestDayLimitsPrecedenceClosureOverHolidayOverWeekly](../tests/closures_test.go) | |
| [TestDuplicateClosureOnSameDateIsRejected](../tests/closures_test.go) | |
| [TestRemoveClosure_NotFound](../tests/closures_test.go) | |
| [TestListClosures_RangeFilters](../tests/closures_test.go) | |
| [TestNewBusinessHoursView_ListsWeek](../tests/conformance_test.go) | |
| [TestNewHolidaysView_CRUD](../tests/conformance_test.go) | |
| [TestNewClosuresView_CRUD](../tests/conformance_test.go) | |
| [TestHolidayItem_DescriptionIsTheRealDate](../tests/conformance_test.go) | |
| [TestItemsDeclaredOnFields](../tests/conformance_test.go) | |
| [TestSentinelErrorMessages](../tests/errors_test.go) | |
| [TestEventCalendarChanged_EveryMutation](../tests/events_test.go) | |
| [TestEventCalendarChanged_RemovalOpens](../tests/events_test.go) | |
| [TestNewHolidayClosesTheDayWithoutRecompile](../tests/holidays_test.go) | |
| [TestRemovingHolidayReopensTheDay](../tests/holidays_test.go) | |
| [TestMovingHolidayFlipsBothDays](../tests/holidays_test.go) | |
| [TestDuplicateHolidayOnSameDateIsRejected](../tests/holidays_test.go) | |
| [TestRemoveHoliday_NotFound](../tests/holidays_test.go) | |
| [TestMigrate_UpToDate_OnlyEnsuresTables](../tests/migrate_test.go) | |
| [TestMigrate_NewField_AddsOnlyThatColumn](../tests/migrate_test.go) | |
| [TestMigrate_IntrospectionFails_ReturnsErrorAndStops](../tests/migrate_test.go) | |
| [TestScreenSaveAndDelete](../tests/screen_save_test.go) | |
| [TestSeed_Load](../tests/seed_test.go) | |
| [TestWeekdayLimits_RegularHours](../tests/weekday_bounds_test.go) | |
| [TestWeekdayLimits_ClosedDay](../tests/weekday_bounds_test.go) | |
| [TestWeekdayLimits_InvalidWeekday](../tests/weekday_bounds_test.go) | |
| [TestWeekdayLimits_IsolatedFromHolidaysAndClosures](../tests/weekday_bounds_test.go) | |
| [TestWeekdayLimits_StoredHourNotAClock](../tests/weekday_bounds_test.go) | |
| [TestWASM_BusinessCalendar_Screens](../tests/ui_view_wasm_test.go) | |
| [TestBusinessHours_SundayAndClosedAreAllowed](../tests/crud_test.go) | un día cerrado (`DayOfWeek 0`, sin horas) se crea sin error y guarda 00:00–00:00 |
| [TestBusinessHours_OneRowPerDay](../tests/crud_test.go) | crear dos veces el mismo día de la semana → conflicto |
| [TestBusinessHours_OpenBeforeClose](../tests/crud_test.go) | abrir a las 18:00 y cerrar a las 08:00 → rechazo sobre `open_min` |
| [TestBusinessHours_InvalidDay](../tests/crud_test.go) | `DayOfWeek 7` → rechazo sobre `day_of_week` |
| [TestBusinessHours_UpdateOnlyListedFields](../tests/crud_test.go) | actualizar solo `close_min` deja `open_min` como estaba |
| [TestBusinessHours_UpdateRejectsOtherFields](../tests/crud_test.go) | actualizar `id` lo rechaza `crud` (campo inmutable) |
| [TestBusinessHours_CloseDayNormalizesMinutes](../tests/crud_test.go) | cerrar un día (`is_open` falso) deja sus horas en 00:00–00:00, escritas por la regla |
| [TestHoliday_DuplicateDateRejected](../tests/crud_test.go) | dos feriados en la misma fecha → conflicto |
| [TestHoliday_NoUpdateOperation](../tests/crud_test.go) | `holiday.update` no existe: «unknown operation» |
| [TestHoliday_DeleteReopensAndPublishes](../tests/crud_test.go) | crear y borrar un feriado publica dos `EventCalendarChanged`: cerrado y luego abierto |
| [TestClosure_DuplicateDateRejected](../tests/crud_test.go) | dos cierres en la misma fecha → conflicto |
| [TestDayDetail_ClosedOnHoliday](../tests/crud_test.go) | leer `day_detail` de un día feriado → cerrado, `ClosedBy == HOLIDAY` |
| [TestDayDetail_NeedsDate](../tests/crud_test.go) | leer `day_detail` sin fecha → rechazo sobre `date` |
| [TestMount_MatchesDefinition](../tests/crud_test.go) | `crud.Mount` acepta la definición del módulo |
| [TestCreate_PublishesChange](../tests/crud_test.go) | crear un feriado publica un `*crud.Change` al publicador del loopback |
| [TestDemo_Valid](../tests/demo_test.go) | `seed.Demo()` es válida y trae al menos una pantalla (destino wasm) |
