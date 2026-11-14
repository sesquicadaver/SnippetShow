# Черга виправлень / post-MVP

Канонічна черга після MVP. Рішення v0.1 (D9–D12) — у `docs/DECISIONS.md`.

## OPEN

| ID | Тема | Нотатка |
|----|------|---------|
| FIX-M10 | Оновлення Vite / закриття high advisory frontend tooling | hygiene перед публічним тегом |
| FIX-L4 | `staticcheck` обов’язковий у CI (fail якщо відсутній) | quality gate |
| FIX-M11-split | Розділення моделі package ≠ command | **D10** — імплементація після tooling |

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

## DONE (рішення / dist smoke / NFR)

| ID | Тема | Нотатка |
|----|------|---------|
| DEC-D9 | NFR-01 accepted-deviation для v0.1 | не блокує реліз-тег |
| DEC-D10 | M11 = розділення моделі | OPEN → FIX-M11-split |
| DEC-D11 | POST-01 не до публічного релізу | D1 шлях стабільний |
| DEC-D12 | Dist smoke Ubuntu-first | Arch/Fedora deferred / best-effort |
| FIX-M9-Ubuntu | Dist smoke Ubuntu + checked `build/dist-notes-ubuntu.md` | `scripts/ubuntu-dist-smoke.sh`; NFR факти 2026-11-14 |
| FIX-FTS-3k | Бенч/gate FTS @ ~3000 programs ≤ 50 ms | `TestFTSLatencyAt3000` + `BenchmarkSearchAt3000`; avg ≪ 50 ms; `ci-local.sh` |

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
