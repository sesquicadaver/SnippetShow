# SnippetShow

**SnippetShow** — локальна Linux-only десктопна утиліта (Wails v2): інтерактивна wiki CLI-команд, man-прапорців, користувацьких сніпетів і конструктор bash-скриптів.

| | |
|---|---|
| Стек | Go ≥ 1.26 (`go.mod` 1.26.0 / Wails v2.17.1) · Wails v2 · Vanilla JS (Vite) · SQLite FTS5 (`modernc.org/sqlite`) |
| БД | `~/.local/share/linux-cmd-wiki/wiki.db` (**D1** / **D11** — без міграції шляху до публічного релізу) |
| Платформи | Debian/Ubuntu, Arch, Fedora — **лише Linux** |
| Ліцензія | [MIT](LICENSE) |

Канонічні docs: [`docs/DECISIONS.md`](docs/DECISIONS.md) (D1–D13) · [`docs/LIVING_SPEC.md`](docs/LIVING_SPEC.md) · [`docs/ROADMAP_FIXES.md`](docs/ROADMAP_FIXES.md) · [`docs/TODO_FIXME.md`](docs/TODO_FIXME.md) · [`docs/NFR_NOTES.md`](docs/NFR_NOTES.md).

## Можливості (стан `main`)

- Режими **Встановлені** / **Обрані** / **Пакети** + фільтр **«Без man»**; алфавіт і **Section** — custom dark toolbar-select (WebKitGTK).
- **Names-first** індекс: швидкий список імен PATH; Section/PM metadata збагачуються у фоні (також на warm bootstrap, якщо категорії порожні).
- **Man-gate (D13):** man парситься й зберігається лише для **Обраних**; «Обрати» / «В обраному» у навігаторі; зняття з обраного purge `man_commands`.
- Навігатор **name-first** (Обране + Назва); lead-опис у wiki для Обраних.
- Wiki: man-таблиця, Raw Man, install-hint для пакетів, завжди рядок **«Базова команда»**, CRUD «Мої команди», Copy / Run / + Script.
- **Script Builder:** reorder (↑↓ / DnD), bash editor, Copy / Run / Export `.sh` (+x) / Save у БД.
- JSON backup (`version: 1`) — лише `user_snippets` + `custom_scripts` (merge, ідемпотентний повторний імпорт).
- Safety confirm перед Run для `rm -rf` / `dd` / `mkfs` / `sudo` (і path/`env` wrappers).

<p align="center">
  <img src="Screenshots/Scr1.png" alt="SnippetShow — список і wiki" width="90%" />
</p>

## Вимоги

Два різні набори залежностей: **toolchain для збірки з джерела** і **shared libs для запуску вже зібраного ELF**. Деталі по дистрибутивах — у [`build/dist-notes-ubuntu.md`](build/dist-notes-ubuntu.md), [`build/dist-notes-arch.md`](build/dist-notes-arch.md), [`build/dist-notes-fedora.md`](build/dist-notes-fedora.md).

### Для збірки

Потрібні: **Go ≥ 1.26** (`go.mod` / Wails v2.17.1), **Node.js + npm** (CI: Node 20), **gcc**, **pkg-config**, **GTK3** + **webkit2gtk 4.1** (headers/`-dev`; або 4.0 на старіших релізах), **Wails CLI v2**.

Після встановлення Go:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
export PATH="$PATH:$(go env GOPATH)/bin"   # додайте у ~/.bashrc / ~/.zshrc
wails doctor
```

На Ubuntu 24.04+ / Fedora 40+ / сучасному Arch збірка йде з tag `webkit2_41` — це робить `./build/build.sh`. Якщо `wails doctor` скаржиться на відсутність webkit 4.0 при наявному 4.1 — ігнорувати (**ENV-01**).

**Debian / Ubuntu**

```bash
sudo apt update
sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev \
  nodejs npm
# Go 1.26 (Ubuntu 24.04: дефолтний golang-go ≈ 1.22 — недостатньо для go.mod)
sudo apt install golang-1.26-go
export PATH="/usr/lib/go-1.26/bin:$PATH"
# альтернатива: офіційний tarball з https://go.dev/dl → /usr/local/go
```

**Arch / Manjaro / EndeavourOS**

```bash
sudo pacman -S base-devel go npm gtk3 webkit2gtk-4.1
```

На rolling Arch unversioned `webkit2gtk` застарілий — ставте **`webkit2gtk-4.1`**.

**Fedora**

```bash
sudo dnf install golang npm gcc pkgconf-pkg-config \
  gtk3-devel webkit2gtk4.1-devel
```

Перевірте `go version` ≥ 1.26. На старіших Fedora можливі fallback: `webkit2gtk4.0-devel` / `webkit2gtk3-devel`.

### Для роботи

Щоб **запустити** уже зібраний `build/bin/SnippetShow`, достатньо runtime shared libs GTK3 і webkit2gtk (без повного `-dev` / `base-devel` toolchain, без Go/Node/Wails). Опційно: clipboard (`wl-clipboard` / `xclip`) і термінал для Run (D7).

**Debian / Ubuntu**

```bash
sudo apt update
sudo apt install libgtk-3-0 libwebkit2gtk-4.1-0
# Ubuntu 24.04+ (t64): інколи libgtk-3-0t64 / libwebkit2gtk-4.1-0t64 — apt підставить transitional
```

**Arch / Manjaro / EndeavourOS**

```bash
sudo pacman -S gtk3 webkit2gtk-4.1
```

**Fedora**

```bash
sudo dnf install gtk3 webkit2gtk4.1
```

На старіших Fedora: `webkit2gtk4.0` / `webkit2gtk3` замість `webkit2gtk4.1`.

## Розробка / тести

```bash
wails doctor
wails dev
./scripts/ci-local.sh   # unit + staticcheck (обов’язковий, pin 2026.2.2)
```

Проєкт тестується у контрольованому середовищі (venv / CI), не «на око» на прод-машині.

## Збірка

```bash
./build/build.sh
# → build/bin/SnippetShow
```

## Запуск

```bash
./build/bin/SnippetShow
```

## Dist smoke (v0.1 / D12)

```bash
./scripts/ubuntu-dist-smoke.sh
```

Ubuntu — обов’язковий чекліст (**checked**). Arch/Fedora — deferred / best-effort після базової готовності.

## Структура

```
internal/{app,db,scanner,manparser,index,clipboard,terminal,safety,backup,scriptbuilder}
frontend/src/ui/{toolbar,navigator,shell,wiki,builder}.js
docs/   scripts/ci-local.sh   build/
```

## JSON backup

Toolbar **Експорт / Імпорт JSON**:

- Сніпети: ключ `(programName, title, command)`; дублікати пропускаються; відсутня програма → stub (`is_installed=0`).
- Скрипти: ключ `(name, scriptBody)`; дублікати пропускаються.
- Повторний імпорт ідемпотентний для ідентичних записів.

## Підтримати проєкт

Якщо цей проєкт вам корисний, ви можете підтримати його розвиток добровільним донатом у USDT.

Донати є необов’язковими і не надають права власності, частки в капіталі, токенів, прав управління, платної підтримки, пріоритетного обслуговування чи будь-якої інвестиційної віддачі.

| Мережа | Адреса |
|---|---|
| USDT ERC-20 / Ethereum | `0xfa9821efd142228d53e1418fe335bb1cd8ff3c39` |
| USDT TRC-20 / Tron | `TNnhueeGqujf6AAUhcgissoEkL7tdzmqQv` |

**Важливо:** переконайтеся, що обрана мережа відповідає типу адреси. Надсилайте USDT ERC-20 лише на адресу Ethereum. Надсилайте USDT TRC-20 лише на адресу Tron. Транзакції, надіслані не в ту мережу, можуть бути втрачені назавжди.

Дякуємо за підтримку проєкту!
