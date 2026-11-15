# Dist notes — Fedora / RHEL-подібні

**Політика v0.1 (D12):** Fedora smoke **не обов’язковий** для тегу. Статус: **deferred** — після готовності базової системи (зафіксовано 2026-11-14). Best-effort нотатки нижче лишаються валідними.

## Runtime / build deps

```bash
sudo dnf install gtk3-devel webkit2gtk4.1-devel golang npm gcc pkgconf-pkg-config
# або webkit2gtk4.0-devel / webkit2gtk3-devel залежно від релізу
# clipboard: wl-clipboard / xclip
```

## Збірка

```bash
./build/build.sh
# артефакт: build/bin/SnippetShow
```

## Smoke checklist (P4-18) — deferred

1. [ ] Запуск ELF — *deferred (D12)*
2. [ ] First-run scan (rpm/dnf metadata) — *deferred (D12)*
3. [ ] Пошук відомої утиліти — *deferred (D12)*
4. [ ] «Оновити індекс» — *deferred (D12)*
5. [ ] Сніпет → Copy → Run (`echo ok`) — *deferred (D12)*
6. [ ] Script export `.sh` + `chmod +x` — *deferred (D12)*
7. [ ] JSON backup export/import — *deferred (D12)*

Повторний прогін: після Ubuntu-стабілізації; трекінг `FIX-M9-Fedora` у `docs/ROADMAP_FIXES.md`.

## Каталог «Пакети»

- Джерело: `dnf list` → таблиця `packages` (D10 / SS-29)
- Порожній кеш: `sudo dnf makecache` (D5)

## openSUSE (P4-19 / best-effort)

- Не окремий обов’язковий smoke-таргет.
- Може частково працювати через rpm-подібні шляхи; при невідомому PM — graceful degrade зі статусом unsupported.
- Не очікувати повного каталогу без адаптера zypper (post-MVP).
