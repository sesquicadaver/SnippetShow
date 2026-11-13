# TODO / FIXME — виявлені заглушки та дефекти

Правило anti-stub: заглушки дозволені лише як тимчасові Phase-маркери з записом тут. У release path MVP їх не лишати.

## Phase 0 (свідомо)

| ID | Модуль | Що | Цільова фаза |
|----|--------|----|--------------|
| STUB-01 | `internal/db` | `Open` → `ErrNotImplemented` | 1 |
| STUB-02 | `internal/scanner` | `DetectPM`, `ScanInstalled` | 1 |
| STUB-03 | `internal/manparser` | `Summary`, `ParseOptions` | 1 |
| STUB-04 | `internal/index` | `UpdateIndex` | 1 |
| STUB-05 | `internal/clipboard` | `Copy` | 3 |
| STUB-06 | `internal/terminal` | `Detect`, `Run` | 3 |
| STUB-07 | `internal/safety` | `Analyze` | 3 |
| STUB-08 | `internal/backup` | `Export`, `Import` | 4 |
| STUB-09 | `internal/scriptbuilder` | `Draft.Generate` | 4 |
| STUB-10 | `frontend/src/ui` | Лише shell Phase 0; немає Wiki layout | 2 |

## Відкриті дефекти середовища

| ID | Опис | Нотатка |
|----|------|---------|
| ENV-01 | `wails doctor` на Ubuntu 24.04 шукає `libwebkit2gtk-4.0`, тоді як доступний 4.1 | Збірка через `-tags webkit2_41` у `build/build.sh` |
