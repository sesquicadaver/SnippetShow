# TODO / FIXME — виявлені заглушки та дефекти

Правило anti-stub: заглушки дозволені лише як тимчасові Phase-маркери з записом тут. У release path MVP їх не лишати.

## Закрито у Phase 1

| ID | Модуль | Що | Статус |
|----|--------|----|--------|
| STUB-01 | `internal/db` | Open/schema/FTS/CRUD | done |
| STUB-02 | `internal/scanner` | DetectPM, ScanInstalled, catalog | done |
| STUB-03 | `internal/manparser` | Summary, RawMan, ParseOptions, pool | done |
| STUB-04 | `internal/index` | FirstRun / UpdateIndex / ParseProgram | done |

## Закрито у Phase 2

| ID | Модуль | Що | Статус |
|----|--------|----|--------|
| STUB-10 | `frontend/src/ui` | Read-only Wiki: toolbar / navigator / man / Raw Man | done |

## Закрито у Phase 3

| ID | Модуль | Що | Статус |
|----|--------|----|--------|
| STUB-05 | `internal/clipboard` | `Copy` (wl-copy / xclip / xsel + Wails ClipboardSetText) | done |
| STUB-06 | `internal/terminal` | `Detect`, `Run` (D7 + argv templates) | done |
| STUB-07 | `internal/safety` | `Analyze` (rm -rf / dd / mkfs / sudo) | done |
| STUB-11 | `frontend` Wiki actions | Copy / Run / + Script | done |
| STUB-12 | `frontend` «Мої команди» | CRUD сніпетів + FTS sync | done |

## Закрито у Phase 4

| ID | Модуль | Що | Статус |
|----|--------|----|--------|
| STUB-08 | `internal/backup` | `Export` / `Import` (`version: 1`, merge) | done |
| STUB-09 | `internal/scriptbuilder` | reorder / Generate / WriteExecutable / ApplyBody | done |
| STUB-13 | `frontend` Script Builder | права панель, editor, save/export/run, DnD | done |

## Hotfix (після code review)

| ID | Опис | Статус |
|----|------|--------|
| H2 | Safety bypass path-prefix / `env` wrappers (`/usr/bin/sudo`, `/bin/rm -rf`, …) | **done** — command-word + wrap/path у `internal/safety` |
| H1 | `ParseOnUpdate` не вмикається → немає reparse man при зміні версії | **done** — `index.New` вмикає прапор; FirstRun без mass parse |
| M1 | `ParseMany` pool не підключений в index | **done** — `parseMany` у UpdateIndex |
| H3 | Script Builder стирає порожні рядки на commit | **done** — `ApplyBody` vs dense `SetLines`; L7 inner `#!` kept |
| H4 | Catalog sync: per-package upsert + FTS → зависання | **done** — `UpsertCatalogProgramsBatch` (chunk 500, deferred FTS) |

## Medium cycle (після hotfix)

| ID | Опис | Статус |
|----|------|--------|
| M5 | `RunScript` не прибирає temp `snippetshow-script-*` | **done** — scrub on error + grace/stale |
| M2 | `GetProgram` мовчки ігнорує помилку `ParseProgram` | **done** — `parseWarning` у відповіді + UI |
| M3 | Немає cancel для UpdateIndex / catalog | **done** — `CancelIndex` + кнопка в toolbar |
| M4 | `ListPrograms` FTS робить N+1 `GetProgramByID` | **done** — колонки в `SearchHit` |
| L1 | Uncheck man → не прибирає з draft | **done** — `RemoveFromScript` |
| M7 | Confirm gate trust-UI (DevTools `confirmed`) | **documented** — SAFE-02 / DECISIONS |
| SS-20 | Run argv з man: кома в прапорці (`--audible,` / `-a, --all`) | **done** — `NormalizeFlagList` + `normalizeManFlag` у Wiki Run/Copy/Script |

## Відомі обмеження Core / Wiki / Builder

| ID | Опис | Нотатка |
|----|------|---------|
| CORE-01 | Man regex покриває типові OPTIONS/FLAGS; складні man — через Raw Man | ітеративно покращувати corpus |
| SAFE-01 | Safety не парсить shell AST; обгортки поза `env/command/nice/nohup` і `$CMD` — відоме обмеження | ROADMAP backlog |
| SAFE-02 | Confirm gate на стороні UI: `confirmed: true` з DevTools обходить діалог (Wails trust-UI) | by design для локального desktop; див. DECISIONS |
| CORE-02 | Repo catalog apt/pacman/dnf — soft-fail при порожньому кеші (D5) | OK; UI показує hint |
| UI-05 | ExportScriptFile/ExportBackup приймають шлях з JS (не лише dialog) | локальний trust boundary; UI використовує dialog |
| CORE-03 | PM metadata lookup може не знайти пакет для `/usr/local` бінарників | summary з whatis / порожній package |
| UI-01 | Повний sync apt-каталогу може бути довгим на дуже великих дзеркалах | progress + batch FTS (H4) + cancel (M3); див. `NFR_NOTES` catalog |
| UI-02 | Copy залежить від Wails clipboard або наявності wl-copy/xclip/xsel | toast з помилкою якщо жоден шлях недоступний |
| UI-03 | Run потребує встановленого терміналу з D7 | діалог/toast при ErrNotFound |
| UI-04 | Bash highlight у Builder — мінімальний (shebang/comments/strings/keywords) | без важкої editor-lib (NFR RAM) |
| DIST-01 | openSUSE — best-effort через rpm/dnf-подібні шляхи; не цільовий smoke | задокументовано в dist-notes |
| NFR-01 | Idle RSS webview часто > 80 МБ на сучасних webkit2gtk | **D9 accepted-deviation v0.1** — не блокує реліз-тег; idle ≈216 МіБ на Ubuntu smoke 2026-11-14; `docs/NFR_NOTES.md`; POST-06 wontfix-for-v0.1 |

## Відкриті дефекти середовища

| ID | Опис | Нотатка |
|----|------|---------|
| ENV-01 | `wails doctor` на Ubuntu 24.04 шукає `libwebkit2gtk-4.0`, тоді як доступний 4.1 | Збірка через `-tags webkit2_41` у `build/build.sh` |
