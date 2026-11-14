# NFR notes (Phase 4 / ТЗ §5)

Виміри виконуються у контрольованому середовищі (локальний CI-скрипт / dist smoke / dev VM), не на прод-машині користувача.

## Цілі ТЗ

| Метрика | Ціль | Політика v0.1 |
|---------|------|---------------|
| Cold start | ≤ 1 s | Записати факт на ELF; host-dependent |
| FTS search (~3000 programs) | ≤ 50 ms | Довести бенчем (цикл B) або зафіксувати відхилення |
| Idle RAM | ≤ 60–80 MB | **D9 / NFR-01: accepted-deviation** — не блокує реліз-тег |

## Методика

1. **Cold start (GUI):** час від старту процесу до RSS ≥ ~80 MiB (webview піднявся); суб’єктивно до першого кадру залежить від webkit. Оркестратор: `./scripts/ubuntu-dist-smoke.sh`.
2. **FTS latency:** `go test ./internal/db -run TestManCommandsAndFTS -count=1` + бенч на фікстурі ~3000 (цикл B).
3. **Idle RSS:** через ~2 s після старту — `ps -o rss= -p $(pgrep -n SnippetShow)` (КіБ).

## Результати (агентське середовище)

### Core / tooling (2026-11-13+)

| Метрика | Результат | Відхилення |
|---------|-----------|------------|
| Core unit suite (`./scripts/ci-local.sh`) | green | — |
| FTS unit на малих фікстурах | ≪ 50 ms | ціль виконується на unit-рівні; бенч @~3000 ще потрібен |

### Ubuntu ELF smoke (2026-11-14) — FIX-M9-Ubuntu / D12

| Метрика | Результат | Нотатка |
|---------|-----------|---------|
| Host | Ubuntu 24.04.5 LTS | webkit2gtk **2.52.6** (`webkit2_41`) |
| ELF | `build/bin/SnippetShow` ≈ 13 MiB | `./build/build.sh` |
| Cold start → RSS≥80 MiB | **≈ 107 ms** | процес; перший кадр UI — окремо, суб’єктивно OK |
| Idle RSS (~2 s) | **≈ 216 MiB** (221 236 KiB) | **NFR-01 / D9 accepted-deviation** |
| P4-16 checklist | **all checked** | `build/dist-notes-ubuntu.md`; GUI interactive PASS |

Артефакти прогону: `build/smoke-artifacts/` (локально, gitignore); короткий звіт у Project store.

## NFR-01 (Idle RSS) — рішення v0.1

- **Статус:** відоме обмеження стеку Wails + webkit2gtk; **не** gate для реліз-тегу v0.1 (`docs/DECISIONS.md` **D9**).
- **POST-06:** wontfix-for-v0.1; подальші спроби знизити RSS — лише post-MVP за окремим пріоритетом.
- У README / реліз-нотах згадувати Idle RAM як host-dependent / часто > 80 МБ.

## Мітигації вже в продукті (не гарантують ≤80 МБ)

- Vanilla JS без фреймворку (D2).
- Лінивий man parse (P2-14).
- Мінімальний bash highlight без editor-lib.
- Один ELF; без фонових мережевих сервісів.

## Catalog sync latency (H4)

- `SyncRepoCatalog` пише пакети чанками по `DefaultCatalogChunkSize` (500) у транзакції.
- FTS оновлюється лише після commit чанка (не на кожен `Upsert` mid-write).
- `cache_empty` (D5) без змін поведінки.
- На дуже великих apt-кешах (десятки тисяч імен) операція все ще може тривати секунди–десятки секунд; UI показує `index:progress` phase `catalog` до `done`. Cancel — `CancelIndex` / кнопка «Скасувати» (M3).

## Dist smoke (D12)

- v0.1: обов’язковий Ubuntu чекліст — **done** (`build/dist-notes-ubuntu.md`, `scripts/ubuntu-dist-smoke.sh`).
- Arch/Fedora — deferred / best-effort після готовності базової системи, не блокують тег.
