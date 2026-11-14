# NFR notes (Phase 4 / ТЗ §5)

Виміри виконуються у контрольованому середовищі (локальний CI-скрипт / dist smoke / `go test`), не на прод-машині користувача.

## Цілі ТЗ

| Метрика | Ціль | Політика v0.1 |
|---------|------|---------------|
| Cold start | ≤ 1 s | Записати факт на ELF; host-dependent (див. Ubuntu smoke / dist-notes) |
| FTS search (~3000 programs) | ≤ 50 ms | **done** — gate `TestFTSLatencyAt3000` |
| Idle RAM | ≤ 60–80 MB | NFR-01 / D9 accepted-deviation (не блокує тег; див. TODO / DECISIONS) |

## Методика

1. **Cold start (GUI):** час до першого кадру / RSS webview; оркестратор `./scripts/ubuntu-dist-smoke.sh` (якщо є в гілці) або `/usr/bin/time` + `ps`.
2. **FTS latency @~3000:** `go test ./internal/db -run '^TestFTSLatencyAt3000$' -count=1 -v`  
   Бенч: `go test ./internal/db -bench=BenchmarkSearchAt3000 -benchtime=2s`.  
   Фікстура: 3000 catalog programs + man/snippet на needle `ffmpegbench` (`internal/db/fts_bench_test.go`).  
   Поріг: avg ≤ **50 ms** (override: `FTS_LATENCY_BUDGET_MS`). Входить у `./scripts/ci-local.sh`.
3. **Idle RSS:** після старту — `ps -o rss= -p $(pgrep -n SnippetShow)` (КіБ).

## Результати

### Core / FTS (агент, 2026-11-14) — FIX-FTS-3k

| Метрика | Результат | Відхилення |
|---------|-----------|------------|
| `TestFTSLatencyAt3000` | avg ≈ **94 µs**, max ≈ **116 µs** @ 3000 programs | ≪ 50 ms — **ціль виконана** |
| `BenchmarkSearchAt3000` | ≈ **90 µs/op**, ~1.7 KiB/op | — |
| Core unit suite (`./scripts/ci-local.sh`) | green (incl. FTS gate) | — |

### Історичні / host-dependent

| Метрика | Результат | Відхилення |
|---------|-----------|------------|
| FTS unit на малих фікстурах | ≪ 50 ms | — |
| Cold start ELF | host / webkit dependent | dist-notes |
| Idle RSS webview | часто > 80 МБ (webkit2gtk) | **NFR-01** accepted для v0.1 |

## NFR-01 (Idle RSS)

- Відоме обмеження Wails + webkit2gtk; **не** gate для реліз-тегу v0.1 (D9, якщо зафіксовано в `DECISIONS.md`).
- POST-06 — post-MVP.

## Мітигації вже в продукті (не гарантують ≤80 МБ)

- Vanilla JS без фреймворку (D2).
- Лінивий man parse (P2-14).
- Мінімальний bash highlight без editor-lib.
- Один ELF; без фонових мережевих сервісів.

## Catalog sync latency (H4)

- `SyncRepoCatalog` пише пакети чанками по `DefaultCatalogChunkSize` (500) у транзакції.
- FTS оновлюється лише після commit чанка (не на кожен `Upsert` mid-write).
- `cache_empty` (D5) без змін поведінки.
- На дуже великих apt-кешах UI показує `index:progress` phase `catalog`; cancel — `CancelIndex` (M3).
