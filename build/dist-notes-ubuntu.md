# Dist notes — Ubuntu / Debian-сімейство

## Runtime / build deps

```bash
sudo apt update
sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev
# clipboard helpers (опційно): wl-clipboard | xclip | xsel
# термінал з D7: gnome-terminal / x-terminal-emulator / …
```

Ubuntu 24.04+: збірка з tag `webkit2_41` через `./build/build.sh`.

## Збірка

```bash
./build/build.sh
# артефакт: build/bin/SnippetShow
```

## Smoke checklist (P4-16)

1. [ ] Запуск ELF / `wails dev`
2. [ ] First-run scan (`EnsureIndexed`)
3. [ ] Пошук відомої утиліти (`ls` / `ffmpeg`)
4. [ ] «Оновити індекс»
5. [ ] Сніпет → Copy → Run (`echo ok`)
6. [ ] Script Builder: + Script → reorder → Export `.sh` (перевірити `chmod +x`)
7. [ ] JSON Export / Import (повторний Import ідемпотентний)

## Відомі обмеження

- `wails doctor` може скаржитись на відсутність webkit 4.0 — ігнорувати, якщо 4.1 є (ENV-01).
- openSUSE не входить у обов’язковий Ubuntu smoke.
