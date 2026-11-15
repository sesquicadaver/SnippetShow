# Dist notes — Arch / Manjaro / EndeavourOS

**Політика v0.1 (D12):** Arch smoke **не обов’язковий** для тегу. Статус: **deferred** — після готовності базової системи (зафіксовано 2026-11-14). Best-effort нотатки нижче лишаються валідними.

## Runtime / build deps

```bash
sudo pacman -S base-devel gtk3 webkit2gtk go npm
# clipboard: wl-clipboard або xclip
# термінал: kitty / alacritty / gnome-terminal — або export TERMINAL=…
```

Якщо `pkg-config` бачить `webkit2gtk-4.1`, `build/build.sh` додасть `-tags webkit2_41`.

## Збірка

```bash
./build/build.sh
# артефакт: build/bin/SnippetShow
```

## Smoke checklist (P4-17) — deferred

1. [ ] Запуск ELF — *deferred (D12)*
2. [ ] First-run scan — *deferred (D12)*
3. [ ] Пошук (`pacman`-встановлена утиліта, напр. `ls`) — *deferred (D12)*
4. [ ] «Оновити індекс» — *deferred (D12)*
5. [ ] Сніпет → Copy → Run (`echo ok`) — *deferred (D12)*
6. [ ] Script export `.sh` + виконуваність — *deferred (D12)*
7. [ ] JSON backup export/import — *deferred (D12)*

Повторний прогін: після Ubuntu-стабілізації; трекінг `FIX-M9-Arch` у `docs/ROADMAP_FIXES.md`.

## Каталог «Пакети»

- Джерело: `pacman -Sl` → таблиця `packages` (D10 / SS-29)
- Порожній/застарілий sync: `sudo pacman -Sy` (D5)

## Відомі обмеження

- AUR-пакети поза `pacman -Sl` не потрапляють у каталог «Пакети».
