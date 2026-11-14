# SnippetShow

**SnippetShow** — Linux Command Wiki & Script Builder: локальна легковагова десктопна утиліта для індексації CLI-програм, man-прапорців, користувацьких сніпетів і конструктора bash-скриптів.

Канонічне ТЗ: [`TOR_base.md`](TOR_base.md). Рішення: [`docs/DECISIONS.md`](docs/DECISIONS.md) (D1–D12). Матриця ТЗ→модуль→тести: [`docs/LIVING_SPEC.md`](docs/LIVING_SPEC.md). NFR: [`docs/NFR_NOTES.md`](docs/NFR_NOTES.md) (NFR-01 / Idle RSS — accepted-deviation для v0.1, D9).

## Стек

| Шар | Технологія |
|-----|------------|
| Backend | Go ≥ 1.22 (Wails v2.17.1 → `go.mod` 1.26.0) |
| GUI | Wails v2 + Vanilla JS (Vite) |
| DB | SQLite (`modernc.org/sqlite`, FTS5) → `~/.local/share/linux-cmd-wiki/wiki.db` |
| Ліцензія | MIT |
| Ціль | **Linux-only** (Debian/Ubuntu, Arch, Fedora); без Windows/macOS артефактів у репо |

## Вимоги середовища

- Go 1.22+
- Wails CLI v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- Node.js + npm (збірка frontend)
- `gcc`, `pkg-config`, GTK3, webkit2gtk

### Залежності webkit2gtk за сімействами

**Debian / Ubuntu** — див. також [`build/dist-notes-ubuntu.md`](build/dist-notes-ubuntu.md)

```bash
sudo apt update
sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev
# Старіші релізи з 4.0:
# sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev
```

На Ubuntu 24.04+ збірка потребує tag `webkit2_41` (див. `build/build.sh`).

**Arch / Manjaro / EndeavourOS** — [`build/dist-notes-arch.md`](build/dist-notes-arch.md)

```bash
sudo pacman -S base-devel gtk3 webkit2gtk
```

**Fedora / RHEL-подібні** — [`build/dist-notes-fedora.md`](build/dist-notes-fedora.md)

```bash
sudo dnf install gtk3-devel webkit2gtk4.1-devel
# або webkit2gtk3-devel / webkit2gtk4.0-devel залежно від релізу
```

## Розробка

```bash
wails doctor
wails dev
# або:
./scripts/ci-local.sh
```

## Збірка ELF

```bash
./build/build.sh
# артефакт: build/bin/SnippetShow
```

## Ubuntu dist smoke (v0.1 / D12)

```bash
./scripts/ubuntu-dist-smoke.sh
# чекліст: build/dist-notes-ubuntu.md (P4-16)
```

Arch/Fedora — deferred після базової готовності (best-effort нотатки в `build/dist-notes-*.md`).

## Тести

Проєкт тестується у контрольованому середовищі (локальний скрипт / CI), не «на око» на прод-машині:

```bash
./scripts/ci-local.sh
```

## JSON backup (фаза 4)

Toolbar: **Експорт JSON** / **Імпорт JSON** — лише `user_snippets` + `custom_scripts` (`version: 1`).

**Merge-політика імпорту:**

- Сніпети: ключ `(programName, title, command)`; дублікати пропускаються.
- Якщо програми немає — створюється stub (`is_installed=0`); `programs`/`man_commands` не затираються.
- Скрипти: ключ `(name, scriptBody)`; дублікати пропускаються.
- Повторний імпорт ідемпотентний для ідентичних записів.

## Структура

`internal/{app,db,scanner,manparser,index,clipboard,terminal,safety,backup,scriptbuilder}`, `frontend/` (Vanilla), `docs/`, `scripts/ci-local.sh`, `.github/workflows/ci.yml`.

Збірка **лише Linux** (`build/build.sh` + `build/appicon.png` + dist-notes). Каталоги `build/windows` / `build/darwin` навмисно відсутні.

## Статус

**MVP фази 0–4** — feature-complete на `main`. Рішення v0.1 (D9–D12): NFR-01 прийнято; M11 = розділення моделі; шлях БД без міграції до публічного релізу; **Ubuntu dist smoke checked**. Черга: `docs/ROADMAP_FIXES.md`.

## Wiki + Builder (фази 2–4)

1. Стартовий `EnsureIndexed` сканує встановлені бінарники.
2. Toolbar: режими, пошук, алфавіт, «Оновити індекс», каталог, Export/Import JSON.
3. Ліворуч — список; у центрі — man / install + «Мої команди».
4. Дії: **Копіювати**, **Запустити**, **+ Script** (і чекбокси man → draft).
5. Праворуч **Script Builder**: reorder (↑↓ / DnD), bash editor + shebang, Copy / Run / Export `.sh` (+x) / Save у БД.
6. Run: D7 + confirm для `rm -rf` / `dd` / `mkfs` / `sudo`.
