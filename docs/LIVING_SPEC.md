# Living Specification — матриця ТЗ → модуль → тести

Оновлювати після кожної фази/модуля. Статуси: `planned` | `stub` | `partial` | `done`.

| ТЗ§ | Вимога (коротко) | Модуль | Тест | Статус |
|-----|------------------|--------|------|--------|
| §1 | Go 1.22+ / Wails v2 / SQLite modernc / ELF + webkit2gtk | `main.go`, `wails.json`, `go.mod` | `scripts/ci-local.sh`, CI workflow | partial (скелет + Core + Wiki UI) |
| §1 | Платформи Debian/Ubuntu, Arch, Fedora (лише Linux) | README; без `build/windows`/`build/darwin` | smoke Phase 4 | planned |
| §2.1 | System Scanner (PM + PATH + repo catalog) | `internal/scanner` | `scanner_test.go` | done |
| §2.2 | Man Parser (whatis, raw, OPTIONS regex, pool) | `internal/manparser` | `manparser_test.go`, `testdata/sample.man` | done |
| §2.3 | Incremental «Оновити індекс» | `internal/index` | `index_test.go` | done |
| §3 | SQLite schema + FTS5 path | `internal/db` | `db_test.go` | done |
| §2.3 / P1-16 | App API: ScanInstalled, UpdateIndex, GetProgram, Search | `internal/app`, `app.go` | `api_test.go` | done |
| §4.1–4.2 / P2 | Toolbar + лівий навігатор (режими, FTS, A–Z, оновлення) | `frontend/src/ui/{toolbar,navigator,shell}.js`, `ListPrograms` | `api_test.go` (фільтри), UI smoke | done |
| §4.3 / P2 read-only | Center Wiki man + Raw Man + install hint (без CRUD/actions) | `frontend/src/ui/wiki.js`, `GetRawMan`, `GetProgram` | `api_test.go` | done |
| §4.3 | Clipboard X11/Wayland | `internal/clipboard` | `clipboard_test.go` | stub |
| §4.3 / §5 | Terminal Run + safety confirm | `internal/terminal`, `internal/safety` | `terminal_test.go`, `safety_test.go` | stub (порядок D7 зафіксовано) |
| §4.4 | Script Builder | `internal/scriptbuilder` + UI placeholder | `scriptbuilder_test.go` | stub (UI placeholder фаза 2) |
| §5 | JSON backup snippets/scripts | `internal/backup` | `backup_test.go` | stub |
| D1–D8 | Рішення старту | `docs/DECISIONS.md`, `internal/app` | `api_test.go` | done |

Посилання на цей файл обов’язкове в MR після змін модулів.

## Контракт JS ↔ Go (фаза 2)

| Метод | Призначення |
|-------|-------------|
| `AppInfo` / `PhaseStatus` | бренд / маркер фази |
| `EnsureIndexed` | first-run scan якщо БД порожня |
| `UpdateIndex` | інкрементальне оновлення + подія `index:progress` |
| `SyncRepoCatalog` | каталог репо → `programs` (режим «Всі», D5) |
| `ListPrograms(ListQuery)` | режим / літера / FTS → список навігатора |
| `Search` / `SearchPrograms` | FTS5 |
| `GetProgram(name, parseMan)` | деталі + lazy man + install-команда |
| `GetRawMan(name)` | сирий man (лише встановлені, D6) |
| `DetectPackageManager` / `InstallCommandFor` | підказки встановлення |
