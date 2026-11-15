# Living Specification — матриця ТЗ → модуль → тести

Оновлювати після кожної фази/модуля. Статуси: `planned` | `stub` | `partial` | `done`.

| ТЗ§ | Вимога (коротко) | Модуль | Тест | Статус |
|-----|------------------|--------|------|--------|
| §1 | Go 1.22+ / Wails v2 / SQLite modernc / ELF + webkit2gtk; staticcheck gate (SS-28); без артефактів збірки в git (SS-35) | `main.go`, `wails.json`, `go.mod`, `build/build.sh`, `.gitignore` | `scripts/ci-local.sh` (required staticcheck), `.github/workflows/ci.yml` | done |
| §1 | Платформи Debian/Ubuntu, Arch, Fedora (лише Linux) | README; `build/dist-notes-*.md`; `scripts/ubuntu-dist-smoke.sh` | Ubuntu P4-16 checked (D12); Arch/Fedora deferred | done (Ubuntu); deferred (Arch/Fedora) |
| §2.1 | System Scanner (PM + PATH + repo catalog batch upsert/FTS) | `internal/scanner`, `internal/db`, `internal/index` | `scanner_test.go`, `db_test.go`, `index_test.go` | done |
| §2.2 | Man Parser (whatis, raw, OPTIONS regex, pool 4–8 у UpdateIndex); Run argv без ком (SS-20); `[`/`test` → `/usr/bin/*` (SS-25); brace-опції `{a\|s\|…}` (SS-26) | `internal/manparser`, `internal/scriptbuilder`, `frontend/src/ui/wiki.js` | `braces_test.go`, `builtins_test.go`, `wiki_flag_test.mjs` | done |
| §2.3 | Incremental «Оновити індекс» + reparse man при зміні версії | `internal/index` (`ParseOnUpdate`) | `index_test.go` | done |
| §3 | SQLite schema + FTS5 path | `internal/db` | `db_test.go` | done |
| §2.3 / P1-16 | App API: ScanInstalled, UpdateIndex(+cancel), GetProgram(+parseWarning), Search | `internal/app`, `app.go` | `api_test.go` | done |
| §4.1–4.2 / P2 | Toolbar + навігатор (Встановлені/Обрані/Пакети, FTS, A–Z, Section, «Без man»); scroll SS-23; select SS-27; пошук SS-30; man-gate D13/SS-32; явний «Обрати»/«В обраному» (SS-34) | `frontend/src/ui/{toolbar,navigator,shell,wiki}.js`, `ListPrograms`, `SetFavorite` | `api_test.go`; `navigator_test.mjs` (`shouldShowFavoriteControl`); `shell_select_test.mjs`; `ui_input_lag_test.mjs`; `index` SetFavorite tests | done |
| §4.3 / P2–P3 | Center Wiki man + Raw Man + install hint + checkbox↔draft + CRUD | `frontend/src/ui/wiki.js`, `GetProgram`, snippets API | `api_test.go`, `db_test.go` | done |
| §4.3 | Clipboard X11/Wayland (+ Wails) | `internal/clipboard`, `App.CopyText` | `clipboard_test.go` | done |
| §4.3 / §5 | Terminal Run + safety confirm (command-word + path/`env` wrappers) | `internal/terminal`, `internal/safety`, `App.RunCommand` | `terminal_test.go`, `safety_test.go`, `api_test.go` | done |
| §4.3 P3-11 / §4.4 | + Script → draft + Script Builder UI (reorder, editor blanks round-trip, export, save); Run з редактора без втрати правок (SS-21); ввід без lag через rAF highlight + incremental paint (SS-30) | `internal/scriptbuilder`, `frontend/src/ui/{builder,shell}.js` | `scriptbuilder_test.go`, `api_test.go`; UI keep-focus + SetScriptBody before Run; `ui_input_lag_test.mjs` | done |
| §3 / P4-7 | `custom_scripts` CRUD | `internal/db/scripts.go`, `SaveCustomScript` | `db_test.go`, `api_test.go` | done |
| §5 | JSON backup snippets/scripts (`version: 1`) | `internal/backup`, toolbar Export/Import | `backup_test.go`, `api_test.go` | done |
| §5 | NFR cold start / search / RSS | `docs/NFR_NOTES.md`, `internal/db/fts_bench_test.go`, `scripts/ubuntu-dist-smoke.sh` | FTS@3000 **done** (≤50 ms); Ubuntu ELF cold ≈107 ms / idle ≈216 MiB (**D9**) | partial (cold/RSS notes) |
| D1–D12 | Рішення старту + v0.1; M11 packages≠commands (SS-29) | `docs/DECISIONS.md`, `internal/db` (`packages`), `internal/app` | `db_test.go`, `api_test.go`; schema v2 | done |

Посилання на цей файл обов’язкове в MR після змін модулів.

## Контракт JS ↔ Go (фаза 4)

| Метод | Призначення |
|-------|-------------|
| `AppInfo` / `PhaseStatus` | бренд / маркер фази |
| `EnsureIndexed` | first-run scan якщо БД порожня |
| `UpdateIndex` | інкрементальне оновлення + подія `index:progress` |
| `CancelIndex` / `IndexBusy` | скасування довгого UpdateIndex / SyncRepoCatalog (M3) |
| `SyncRepoCatalog` | каталог репо → `packages` (режим «Пакети», D5 / SS-29) |
| `ListPrograms(ListQuery)` | installed → `programs`; all → `packages` (+ FTS); без N+1 |
| `Search` / `SearchPrograms` | FTS5 |
| `GetProgram(name, parseMan)` | деталі + lazy man + `parseWarning` + snippets + install-команда |
| `GetRawMan(name)` | сирий man (лише встановлені, D6) |
| `DetectPackageManager` / `InstallCommandFor` | підказки встановлення |
| `ListSnippets` / `CreateSnippet` / `UpdateSnippet` / `DeleteSnippet` | CRUD «Мої команди» |
| `CopyText` | буфер обміну (Wails → wl-copy/xclip/xsel) |
| `AnalyzeCommand` / `RunCommand` | safety + запуск у терміналі (D7) |
| `AddToScript` / `RemoveFromScript` / `ScriptDraft` / `MoveScriptLine` / `RemoveScriptLine` / `ClearScriptDraft` / `SetScriptBody` / `GenerateScript` | in-memory Script Builder |
| `ExportScriptFile` / `ExportScriptDialog` / `RunScript` | експорт `.sh` (+x) і Run скрипту (temp scrub M5) |
| `ListCustomScripts` / `SaveCustomScript` / `LoadCustomScript` / `DeleteCustomScript` | CRUD `custom_scripts` |
| `ExportBackup` / `ImportBackup` / `ExportBackupDialog` / `ImportBackupDialog` | JSON backup `version: 1` |
