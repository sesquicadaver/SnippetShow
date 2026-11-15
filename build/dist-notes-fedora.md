# Dist notes — Fedora / RHEL-подібні

**Політика v0.1 (D12):** Fedora smoke **не обов’язковий** для тегу. Статус: **deferred** — після готовності базової системи (зафіксовано 2026-11-14). Best-effort нотатки нижче лишаються валідними.

## Для збірки

```bash
sudo dnf install golang npm gcc pkgconf-pkg-config \
  gtk3-devel webkit2gtk4.1-devel
# або webkit2gtk4.0-devel / webkit2gtk3-devel на старіших релізах
# перевірте: go version ≥ 1.26
go install github.com/wailsapp/wails/v2/cmd/wails@latest
export PATH="$PATH:$(go env GOPATH)/bin"
# clipboard: wl-clipboard / xclip
```

## Для роботи

Запуск уже зібраного ELF — runtime shared libs (без `*-devel` / golang / npm / gcc):

```bash
sudo dnf install gtk3 webkit2gtk4.1
# на старіших: webkit2gtk4.0 / webkit2gtk3
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
