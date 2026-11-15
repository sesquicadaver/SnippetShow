# Черга виправлень / post-MVP

Канонічна черга після MVP. Рішення v0.1 (D9–D12) — у `docs/DECISIONS.md`.

## OPEN

| ID | Тема | Нотатка |
|----|------|---------|
| *(черга порожня)* | — | наступне — backlog / post-v0.1 |

Порядок Autopilot: зверху вниз у таблиці OPEN.

## DONE (hotfix)

| ID | Тема | Нотатка |
|----|------|---------|
| FIX-H2 | Safety: path/env/command-word bypass | `/usr/bin/sudo`, `/bin/rm -rf`, `env sudo`, … → confirm |
| FIX-H1 | ParseOnUpdate + reparse on version change | `index.New` → `ParseOnUpdate: true` |
| FIX-M1 | `ParseMany` pool у index UpdateIndex | Workers 4–8; progress phase `parse` |
| FIX-H3 | Script Builder blank lines / editor fidelity | `ApplyBody` зберігає blanks; strip лише leading shebang |
| FIX-H4 | Catalog batch upsert / deferred FTS | chunks 500; FTS після commit; D5 без змін |

## DONE (medium cycle)

| ID | Тема | Нотатка |
|----|------|---------|
| FIX-M5 | `RunScript` temp scrub | immediate cleanup on error; grace + stale glob |
| FIX-M2 | `GetProgram` parse warning | `parseWarning` у `ProgramDetail` + UI toast |
| FIX-M3 | Cancel UpdateIndex / catalog | `CancelIndex` + toolbar «Скасувати» |
| FIX-M4 | FTS ListPrograms без N+1 | `SearchHit` несе колонки programs |
| FIX-L1 | Wiki uncheck → remove from draft | `RemoveFromScript` / `RemoveCommand` |

## DONE (рішення / dist smoke / NFR / bugfix)

| ID | Тема | Нотатка |
|----|------|---------|
| DEC-D9 | NFR-01 accepted-deviation для v0.1 | не блокує реліз-тег |
| DEC-D10 | M11 = розділення моделі | **done** → FIX-M11-split |
| FIX-M11-split | `packages` ≠ `programs` (команди) | schema v2; mode all → packages; SS-29 |
| DEC-D11 | POST-01 не до публічного релізу | D1 шлях стабільний |
| DEC-D12 | Dist smoke Ubuntu-first | Arch/Fedora deferred / best-effort |
| FIX-M9-Ubuntu | Dist smoke Ubuntu + checked `build/dist-notes-ubuntu.md` | `scripts/ubuntu-dist-smoke.sh`; NFR факти 2026-11-14 |
| FIX-FTS-3k | Бенч/gate FTS @ ~3000 programs ≤ 50 ms | `TestFTSLatencyAt3000` + `BenchmarkSearchAt3000`; avg ≪ 50 ms; `ci-local.sh` |
| FIX-SS-20 | Run argv: зайва кома з man-синтаксису | `NormalizeFlagList`; Wiki `commandFromFlag` |
| FIX-SS-21 | Run з редактора BASH (втрачені правки / usage-only) | keep-focus на actions; sync SetScriptBody перед Run |
| FIX-M10 | Vite 6.4.6; `npm audit` 0 vulns (було high на 5.4.x) | SS-22; `frontend/package.json` + lock |
| FIX-SS-23 | Navigator scroll: список зникав через stale closure | `host._navState` + `navigator_test.mjs` |
| FIX-SS-27 | UI lag на вибір програми (~2 с) | без `reloadList` у select; parse лише якщо !manParsed |
| FIX-SS-25 | Run `[ --help`: bash builtin → `/usr/bin/[` | `ResolveBuiltinShadows` + Wiki `shellSafeProgram` |
| FIX-SS-26 | Brace-опції man (`-ao{a\|s\|t\|u}`) | expand у manparser + Wiki; опис «(варіант: …)» |
| FIX-L4 | `staticcheck` обов’язковий у CI / ci-local | SS-28; pin `2026.2.2`; fail якщо відсутній |
| FIX-SS-30 | Повільний ввід у пошук / BASH editor (фокус, lag) | incremental toolbar+builder; soft `reloadList`; rAF highlight |
| DEC-D13 | Обрані + man-gate + Section + «Без man» | **D13.1–D13.11** затверджено (2026-11-14); сирі Section; category bulk у фоні; фільтр «Без man» |
| FIX-SS-31 | Модель Обрані / man-gate (рішення) | **done** (docs D13.1–D13.11) |
| FIX-SS-32 | Імплементація Обрані / man-gate | schema v3; names-only scan; SetFavorite+purge; Section enrich; UI |
| FIX-SS-34 | Явний «Обрати» у навігаторі (не лише ★) | labeled checkbox + col «Обране»; `shouldShowFavoriteControl` |
| FIX-SS-35 | Артефакти збірки не в git | untrack wailsjs + md5; ignore AppImage/coverage/.wails |
| FIX-SS-36a | Toolbar densify (≤2 смуги) | dual-row CSS areas; compact alphabet; thin progress |
| FIX-SS-36d | Без subtitle; літера = select | brand name only; letter dropdown replaces A–Z strip |
| FIX-SS-36b | UpdateIndex names-first | names batch + background EnrichMetadata (D13.9) |
| FIX-SS-40 | List paint before enrich; toolbar composition | WAL/readers; queueEnrich after reloadList; segmented modes |
| FIX-SS-41 | Nav columns: drop Status/Summary; wiki lead desc | name-first list; Favorites description above man |
| FIX-SS-42 | Unify button primary chrome (variant B) | one `btn-primary` gold; kill `btn-accent`; `.toolbar .btn.btn-primary` wins |

## Backlog (після v0.1 / відкладено)

| ID | Тема | Нотатка |
|----|------|---------|
| FIX-M9-Arch | Dist smoke Arch | **D12** — після готовності базової системи |
| FIX-M9-Fedora | Dist smoke Fedora | **D12** — після готовності базової системи |
| POST-01 | Міграція шляху БД з `linux-cmd-wiki` на бренд SnippetShow | **D11** — не до публічного релізу |
| POST-02 | Опційний Svelte UI | Лише якщо Vanilla розростається (D2) |
| POST-03 | EN i18n словник | Паралельно до `uk.js` |
| POST-04 | openSUSE first-class PM adapter | Зараз best-effort (DIST-01) |
| POST-05 | GUI CI smoke з webkit | Core-тести без GUI лишаються обов’язковими; Ubuntu desktop smoke — окремо |
| POST-06 | Idle RSS ≤ 80 МБ на webkit2gtk | **D9** — wontfix-for-v0.1; див. `docs/NFR_NOTES.md` |
| POST-07 | Багатший bash highlight / редактор | UI-04 — без важкої lib у MVP |
| FIX-L5 | Підтвердити GitHub Actions Checks на `main` | ops (workflow зараз startup_failure репо-wide) |
