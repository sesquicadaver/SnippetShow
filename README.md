# SnippetShow

**SnippetShow** — Linux Command Wiki & Script Builder: локальна легковагова десктопна утиліта для індексації CLI-програм, man-прапорців, користувацьких сніпетів і конструктора bash-скриптів.

Канонічне ТЗ: [`TOR_base.md`](TOR_base.md). Рішення старту: [`docs/DECISIONS.md`](docs/DECISIONS.md) (D1–D8). Матриця ТЗ→модуль→тести: [`docs/LIVING_SPEC.md`](docs/LIVING_SPEC.md).

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

**Debian / Ubuntu**

```bash
sudo apt update
sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev
# Старіші релізи з 4.0:
# sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev
```

На Ubuntu 24.04+ збірка потребує tag `webkit2_41` (див. `build/build.sh`).

**Arch / Manjaro / EndeavourOS**

```bash
sudo pacman -S base-devel gtk3 webkit2gtk
```

**Fedora / RHEL-подібні**

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

## Тести

Проєкт тестується у контрольованому середовищі (локальний скрипт / CI), не «на око» на прод-машині:

```bash
./scripts/ci-local.sh
```

## Структура (Phase 0)

Див. план у Agent Store / `docs/`. Коротко: `internal/{app,db,scanner,manparser,index,clipboard,terminal,safety,backup,scriptbuilder}`, `frontend/` (Vanilla), `docs/`, `scripts/ci-local.sh`, `.github/workflows/ci.yml`.

## Статус

**Фаза 0** — скелет Wails/Go, документація рішень, заглушки модулів без повної Core-логіки.
