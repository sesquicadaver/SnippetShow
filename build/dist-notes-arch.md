# Dist notes — Arch / Manjaro / EndeavourOS

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

## Smoke checklist (P4-17)

1. [ ] Запуск ELF
2. [ ] First-run scan
3. [ ] Пошук (`pacman`-встановлена утиліта, напр. `ls`)
4. [ ] «Оновити індекс»
5. [ ] Сніпет → Copy → Run (`echo ok`)
6. [ ] Script export `.sh` + виконуваність
7. [ ] JSON backup export/import

## Каталог «Всі програми»

- Джерело: `pacman -Sl`
- Порожній/застарілий sync: `sudo pacman -Sy` (D5)

## Відомі обмеження

- AUR-пакети поза `pacman -Sl` не потрапляють у каталог «Всі».
