# Рішення проєкту SnippetShow (D1–D12)

Зафіксовано на старті Phase 0 (D1–D8) за `docs/work-plan.md` / планом SnippetShow; D9–D12 — продуктові рішення v0.1 (2026-11-14). Зміна будь-якого пункту потребує нового запису тут і оцінки впливу на вже зроблені фази.

| # | Питання | Рішення | Обґрунтування |
|---|---------|---------|---------------|
| **D1** | Бренд / ID пакета / шлях БД | **UI-назва:** SnippetShow; **підзаголовок:** Linux Command Wiki & Script Builder; **шлях БД:** `~/.local/share/linux-cmd-wiki/wiki.db` (як у ТЗ); **module / binary:** `snippetshow` / `SnippetShow` | Не ламати шлях з ТЗ; бренд репо видимий у вікні |
| **D2** | Vanilla JS vs Svelte | **Vanilla JS + мінімальний CSS** (Vite лише як бандлер); Svelte — опційний апгрейд після фази 2 | Менше tooling, швидший cold start, простіша Wails-зв’язка |
| **D3** | Мова UI | **Українська** як основна для першої збірки; рядки в `frontend/src/i18n/uk.js` | Робоча мова команди; словник готує EN пізніше |
| **D4** | Ліцензія | **MIT** (`LICENSE`) | Типово для локальних OSS desktop утиліт |
| **D5** | Порожній/застарілий package cache | Режим «Всі програми»: порожній стан + підказка оновити кеш (`apt update` / `pacman -Sy` / `dnf makecache`); не падати | UX без вигаданих даних |
| **D6** | Man для невстановлених | **Ні** у MVP: summary з репо + install-команда; man — лише для встановлених | Акцент ТЗ; уникає парсу відсутніх man |
| **D7** | Порядок fallback терміналів | 1) `$TERMINAL` → 2) `x-terminal-emulator` → 3) `gnome-terminal` → 4) `konsole` → 5) `xfce4-terminal` → 6) помилка з діалогом | Як у ТЗ, з явною фінальною помилкою |
| **D8** | Версія Wails / Go | **Go ≥ 1.22** (канон ТЗ); **Wails v2.17.1** підняв `go.mod` до **1.26.0** (вимога модуля) | Канон ТЗ + фактична вимога останньої стабільної Wails 2.x |
| **D9** | Idle RSS / DoD v0.1 (NFR-01) | **Прийняти** Idle RSS часто > 80 МБ (webkit2gtk) як **відоме обмеження v0.1**; **не** блокувати реліз-тег до ≤80 МБ | Host-dependent floor webview; мітигація RSS — post-MVP (POST-06 wontfix-for-v0.1) |
| **D10** | M11: пакет ≠ команда в режимі «Всі» | Варіант **«розділення»**: окрема модель `packages` vs `commands` (не лише UX-підказка і не лише фільтр PATH) | Прибирає плутанину каталогу репо vs PATH-бінарники; імплементація — цикл D |
| **D11** | POST-01 / зміна шляху БД | **Не планується** до публічного релізу; D1 шлях лишається; **без** міграції для існуючих інсталяцій до того | Стабільність шляху для early adopters; POST-01 далекий backlog |
| **D12** | Dist smoke для тегу v0.1 | Спочатку **лише Ubuntu** (обов’язковий чекліст); Arch/Fedora — після готовності базової системи; **best-effort** нотатки ок і **не** блокують тег | Реалістичний DoD без очікування трьох GUI-середовищ |

## Примітки середовища

- На Ubuntu 24.04+ для збірки Wails потрібен tag **`webkit2_41`** (пакет `libwebkit2gtk-4.1-dev`), бо `webkit2gtk-4.0` більше не постачається. Обгортка: `build/build.sh`.
- Продукт **Linux-only**: каталоги `build/windows/` та `build/darwin/` не тримаємо в репо (ігнор у `.gitignore`, якщо Wails їх регенерує).
- Ubuntu smoke оркестратор: `scripts/ubuntu-dist-smoke.sh` + чекліст `build/dist-notes-ubuntu.md`.

## Модель загроз Run / Export (M7 / SAFE-02)

- Wails bindings (`RunCommand`, `RunScript`, `Export*`) довіряють локальному UI-процесу: параметр `confirmed: true` і довільний шлях експорту доступні з DevTools.
- Це **очікувана** модель для single-user desktop ELF (не мережевий сервіс): захист — діалог confirm у штатному UI + safety.Analyze перед стартом; sandbox / RPC auth поза MVP.
- Export у штатному UI йде через `SaveFileDialog` / `OpenFileDialog`; прямі `ExportScriptFile`/`ExportBackup(path)` лишаються для тестів і автоматизації.

## Наслідки D9–D12 для черги

- OPEN Autopilot: L4 staticcheck → M11 split (`docs/ROADMAP_FIXES.md`); FTS@3k / Ubuntu dist smoke / Vite tooling (FIX-M10) — **done**.
- POST-01 / POST-06 не входить у шлях до публічного тегу v0.1.
- Arch/Fedora smoke — після базової готовності (D12).
