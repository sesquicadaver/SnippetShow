# Черга виправлень / post-MVP

Канонічна черга після MVP. Поки MVP не завершено — пункти нижче не блокують Phase 0–4, якщо не позначені як OPEN для Autopilot.

## OPEN

_(порожньо для цього патчу — див. також чергу v0.1 у інших PR)_

## DONE (bugfix)

| ID | Тема | Нотатка |
|----|------|---------|
| FIX-SS-20 | Run argv: зайва кома з man-синтаксису | `NormalizeFlagList`; Wiki `commandFromFlag` |

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

## Backlog (після v0.1)

| ID | Тема | Нотатка |
|----|------|---------|
| POST-01 | Міграція шляху БД з `linux-cmd-wiki` на бренд SnippetShow | Лише після явного рішення змінити D1 |
| POST-02 | Опційний Svelte UI | Лише якщо Vanilla розростається (D2) |
| POST-03 | EN i18n словник | Паралельно до `uk.js` |
| POST-04 | openSUSE first-class PM adapter | Зараз best-effort (DIST-01) |
| POST-05 | GUI CI smoke з webkit | Core-тести без GUI лишаються обов’язковими |
| POST-06 | Idle RSS ≤ 80 МБ на webkit2gtk | NFR-01 — виміри в `docs/NFR_NOTES.md` |
| POST-07 | Багатший bash highlight / редактор | UI-04 — без важкої lib у MVP |
