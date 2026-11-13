# Dist notes — Fedora / RHEL-подібні

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

## Smoke checklist (P4-18)

1. [ ] Запуск ELF
2. [ ] First-run scan (rpm/dnf metadata)
3. [ ] Пошук відомої утиліти
4. [ ] «Оновити індекс»
5. [ ] Сніпет → Copy → Run (`echo ok`)
6. [ ] Script export `.sh` + `chmod +x`
7. [ ] JSON backup export/import

## Каталог «Всі програми»

- Джерело: `dnf list`
- Порожній кеш: `sudo dnf makecache` (D5)

## openSUSE (P4-19 / best-effort)

- Не окремий обов’язковий smoke-таргет.
- Може частково працювати через rpm-подібні шляхи; при невідомому PM — graceful degrade зі статусом unsupported.
- Не очікувати повного каталогу без адаптера zypper (post-MVP).
