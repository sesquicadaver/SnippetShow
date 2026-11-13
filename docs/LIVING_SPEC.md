# Living Specification — матриця ТЗ → модуль → тести

Оновлювати після кожної фази/модуля. Статуси: `planned` | `stub` | `partial` | `done`.

| ТЗ§ | Вимога (коротко) | Модуль | Тест | Статус |
|-----|------------------|--------|------|--------|
| §1 | Go 1.22+ / Wails v2 / SQLite modernc / ELF + webkit2gtk | `main.go`, `wails.json`, `go.mod` | `scripts/ci-local.sh`, CI workflow | partial (скелет + Core + Wiki + Snippets) |
| §1 | Платформи Debian/Ubuntu, Arch, Fedora (лише Linux) | README; без `build/windows`/`build/darwin` | smoke Phase 4 | planned |
| §2.1 | System Scanner (PM + PATH + repo catalog) | `internal/scanner` | `scanner_test.go` | done |
| §2.2 | Man Parser (whatis, raw, OPTIONS regex, pool) | `internal/manparser` | `manparser_test.go`, `testdata/sample.man` | done |
| §2.3 | Incremental «Оновити індекс» | `internal/index` | `index_test.go` | done |
| §3 | SQLite schema + FTS5 path | `internal/db` | `db_test.go` | done |
| §2.3 / P1-16 | App API: ScanInstalled, UpdateIndex, GetProgram, Search | `internal/app`, `app.go` | `api_test.go` | done |
| §4.1–4.2 / P2 | Toolbar + лівий навігатор (режими, FTS, A–Z, оновлення) | `frontend/src/ui/{toolbar,navigator,shell}.js`, `ListPrograms` | `api_test.go` (фільтри), UI smoke | done |
| §4.3 / P2–P3 | Center Wiki man + Raw Man + install hint + «Мої команди» CRUD | `frontend/src/ui/wiki.js`, `GetProgram`, snippets API | `api_test.go`, `db_test.go` | done |
| §4.3 | Clipboard X11/Wayland (+ Wails) | `internal/clipboard`, `App.CopyText` | `clipboard_test.go` | done |
| §4.3 / §5 | Terminal Run + safety confirm | `internal/terminal`, `internal/safety`, `App.RunCommand` | `terminal_test.go`, `safety_test.go`, `api_test.go` | done |
| §4.3 P3-11 | + Script → in-memory draft | `internal/scriptbuilder`, `AddToScript` | `scriptbuilder_test.go`, `api_test.go` | partial (draft; UI фаза 4) |
| §4.4 | Script Builder UI | `internal/scriptbuilder` + UI placeholder | `scriptbuilder_test.go` | stub (лічильник draft у фазі 3) |
| §5 | JSON backup snippets/scripts | `internal/backup` | `backup_test.go` | stub |
| D1–D8 | Рішення старту | `docs/DECISIONS.md`, `internal/app` | `api_test.go` | done |

Посилання на цей файл обов’язкове в MR після змін модулів.

## Контракт JS ↔ Go (фаза 3)

| Метод | Призначення |
|-------|-------------|
| `AppInfo` / `PhaseStatus` | бренд / маркер фази |
| `EnsureIndexed` | first-run scan якщо БД порожня |
| `UpdateIndex` | інкрементальне оновлення + подія `index:progress` |
| `SyncRepoCatalog` | каталог репо → `programs` (режим «Всі», D5) |
| `ListPrograms(ListQuery)` | режим / літера / FTS → список навігатора |
| `Search` / `SearchPrograms` | FTS5 |
| `GetProgram(name, parseMan)` | деталі + lazy man + snippets + install-команда |
| `GetRawMan(name)` | сирий man (лише встановлені, D6) |
| `DetectPackageManager` / `InstallCommandFor` | підказки встановлення |
| `ListSnippets` / `CreateSnippet` / `UpdateSnippet` / `DeleteSnippet` | CRUD «Мої команди» |
| `CopyText` | буфер обміну (Wails → wl-copy/xclip/xsel) |
| `AnalyzeCommand` / `RunCommand` | safety + запуск у терміналі (D7) |
| `AddToScript` / `ScriptDraft` | in-memory draft для Script Builder |
