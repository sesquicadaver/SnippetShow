# Черга виправлень / post-MVP

Канонічна черга після MVP. Поки MVP не завершено — пункти нижче не блокують Phase 0–4, якщо не позначені як OPEN для Autopilot.

## OPEN (hotfix queue)

| ID | Тема | Нотатка |
|----|------|---------|
| FIX-H1 | ParseOnUpdate + reparse on version change | ТЗ §2.3; увімкнути прапор + фон parse |
| FIX-M1 | Підключити `ParseMany` у index (pool 4–8) | ТЗ §2.2; разом з H1 |
| FIX-H3 | Script Builder: зберігати blank lines / editor fidelity | ТЗ §4.4; L7 shebang strip |
| FIX-H4 | Catalog batch upsert / відкладений FTS | NFR / UX «Підвантажити каталог» |

## DONE (hotfix)

| ID | Тема | Нотатка |
|----|------|---------|
| FIX-H2 | Safety: path/env/command-word bypass | `/usr/bin/sudo`, `/bin/rm -rf`, `env sudo`, … → confirm |

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
