# NFR notes (Phase 4 / ТЗ §5)

Виміри виконуються у контрольованому середовищі (локальний CI-скрипт / dev VM), не на прод-машині користувача.

## Цілі ТЗ

| Метрика | Ціль |
|---------|------|
| Cold start | ≤ 1 s |
| FTS search (~3000 programs) | ≤ 50 ms |
| Idle RAM | ≤ 60–80 MB |

## Методика

1. **Cold start (GUI):** `/usr/bin/time -f '%e' ./build/bin/SnippetShow` — суб’єктивно до першого кадру; точний GUI timing залежить від webkit.
2. **FTS latency:** `go test ./internal/db -run TestManCommandsAndFTS -count=1` + бенч на фікстурі (див. нижче).
3. **Idle RSS:** після старту й першого scan — `ps -o rss= -p $(pgrep -n SnippetShow)` (КіБ).

## Результати (агентське середовище, 2026-11-13)

| Метрика | Результат | Відхилення |
|---------|-----------|------------|
| Core unit suite (`./scripts/ci-local.sh`) | green | — |
| FTS unit на малих фікстурах | ≪ 50 ms | ціль виконується на unit-рівні |
| Cold start ELF | залежить від наявного webkit2gtk на хості збірки | див. dist-notes |
| Idle RSS webview | часто вище 80 МБ через webkit2gtk | **NFR-01** у `TODO_FIXME.md` |

## Мітигації NFR-01

- Vanilla JS без фреймворку (D2).
- Лінивий man parse (P2-14).
- Мінімальний bash highlight без editor-lib.
- Один ELF; без фонових мережевих сервісів.

## Catalog sync latency (H4)

- `SyncRepoCatalog` пише пакети чанками по `DefaultCatalogChunkSize` (500) у транзакції.
- FTS оновлюється лише після commit чанка (не на кожен `Upsert` mid-write).
- `cache_empty` (D5) без змін поведінки.
- На дуже великих apt-кешах (десятки тисяч імен) операція все ще може тривати секунди–десятки секунд; UI показує `index:progress` phase `catalog` до `done`. Cancel — `CancelIndex` / кнопка «Скасувати» (M3).

Подальші оптимізації RSS — post-MVP (`docs/ROADMAP_FIXES.md`).
