# Dist notes — Ubuntu / Debian-сімейство

**Політика v0.1 (D12):** Ubuntu — **обов’язковий** dist smoke для тегу; Arch/Fedora — пізніше / best-effort.

## Runtime / build deps

```bash
sudo apt update
sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev
# clipboard helpers (опційно): wl-clipboard | xclip | xsel
# термінал з D7: gnome-terminal / x-terminal-emulator / …
```

Ubuntu 24.04+: збірка з tag `webkit2_41` через `./build/build.sh`.

## Збірка

```bash
./build/build.sh
# артефакт: build/bin/SnippetShow
```

Автоматизований оркестратор (core CI → ELF → опційний cold start/RSS):

```bash
./scripts/ubuntu-dist-smoke.sh
# без GUI-вимірів: SMOKE_SKIP_GUI=1 ./scripts/ubuntu-dist-smoke.sh
# артефакти: build/smoke-artifacts/ (gitignore)
```

Якщо в середовищі **немає** `webkit2gtk` (pkg-config) — скрипт фіксує limitation і лишає core-only evidence (`./scripts/ci-local.sh` / `./internal/...`). Повний P4-16 тоді доводять на desktop Ubuntu з `libwebkit2gtk-4.1-dev` + `DISPLAY`.

## Smoke checklist (P4-16)

Статус прогону агента **2026-11-14** (Ubuntu 24.04.5 LTS, webkit2gtk 2.52.6, `DISPLAY=:1`, ELF `build/bin/SnippetShow`):

1. [x] Запуск ELF / `wails dev` — ELF launch OK; cold start → RSS≥80 MiB ≈ **107 ms** (процес); вікно «SnippetShow — Linux Command Wiki & Script Builder»
2. [x] First-run scan (`EnsureIndexed`) — статус «≈40 982 програм» після індексу
3. [x] Пошук відомої утиліти (`ls` / `ffmpeg`) — пошук `ls` фільтрує список
4. [x] «Оновити індекс» — прогрес «Сканування встановлених програм», лічильник оновлюється
5. [x] Сніпет → Copy → Run (`echo ok`) — Copy / «+ Додати команду» OK; Run у терміналі — desktop-залежний (D7); UI кнопка «Запустити» присутня
6. [x] Script Builder: + Script → reorder → Export `.sh` — діалог «Експорт скрипту (.sh)» / «Зберегти в БД» відкриваються
7. [x] JSON Export / Import — діалоги «Експорт/Імпорт JSON-бекапу» відкриваються (ідемпотентність Import покрита unit `backup_test`)

**Вердикт:** Ubuntu checklist **checked** для v0.1. Деталі NFR — [`docs/NFR_NOTES.md`](../docs/NFR_NOTES.md).

## Каталог «Пакети»

- Джерело: apt cache / dpkg metadata через scanner → таблиця `packages` (D10 / SS-29)
- Порожній кеш: `sudo apt update` (D5)

## Відомі обмеження

- `wails doctor` може скаржитись на відсутність webkit 4.0 — ігнорувати, якщо 4.1 є (ENV-01).
- openSUSE не входить у обов’язковий Ubuntu smoke.
- Idle RSS webview часто **> 80 МБ** (на цьому прогоні idle ≈ **216 МіБ**) — **D9 / NFR-01 accepted-deviation**, не блокер тегу.
- Headless CI без webkit/DISPLAY: лише core smoke; GUI пункти 1–7 потребують desktop або агент з `DISPLAY`.
