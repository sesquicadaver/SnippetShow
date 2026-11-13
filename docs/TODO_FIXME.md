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

## Відкриті заглушки (наступні фази)

| ID | Модуль | Що | Цільова фаза |
|----|--------|----|--------------|
| STUB-05 | `internal/clipboard` | `Copy` | 3 |
| STUB-06 | `internal/terminal` | `Detect`, `Run` | 3 |
| STUB-07 | `internal/safety` | `Analyze` | 3 |
| STUB-08 | `internal/backup` | `Export`, `Import` | 4 |
| STUB-09 | `internal/scriptbuilder` | `Draft.Generate` | 4 |
| STUB-11 | `frontend` Wiki actions | Copy / Run / + Script у таблиці man — заглушка «Фаза 3» | 3 |
| STUB-12 | `frontend` «Мої команди» | CRUD сніпетів — placeholder | 3 |
| STUB-13 | `frontend` Script Builder | права панель — placeholder фази 4 | 4 |

## Відомі обмеження Core / Wiki

| ID | Опис | Нотатка |
|----|------|---------|
| CORE-01 | Man regex покриває типові OPTIONS/FLAGS; складні man — через Raw Man | ітеративно покращувати corpus |
| CORE-02 | Repo catalog apt/pacman/dnf — soft-fail при порожньому кеші (D5) | OK; UI показує hint |
| CORE-03 | PM metadata lookup може не знайти пакет для `/usr/local` бінарників | summary з whatis / порожній package |
| UI-01 | Повний sync apt-каталогу може бути довгим на великих дзеркалах | progress + окрема кнопка «Підвантажити каталог» |

## Відкриті дефекти середовища

| ID | Опис | Нотатка |
|----|------|---------|
| ENV-01 | `wails doctor` на Ubuntu 24.04 шукає `libwebkit2gtk-4.0`, тоді як доступний 4.1 | Збірка через `-tags webkit2_41` у `build/build.sh` |
