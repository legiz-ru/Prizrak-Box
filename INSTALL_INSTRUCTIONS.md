# ВАЖНО: Как правильно запустить установщик

## Проблема: Установщик запускается без диалогов

Если установщик запросил права администратора и сразу установился без показа диалогов, это может быть по следующим причинам:

### Причина 1: Запущен неправильный файл

После сборки (`npm run make`) в папке `out/make/wix/x64/` создается несколько файлов:

```
out/make/wix/x64/
├── Prizrak-Box-<version>.msi          ← Правильный файл!
├── Prizrak-Box-<version>-setup.exe    ← Может запускать MSI в тихом режиме
└── ...
```

**РЕШЕНИЕ:** Запускайте `.msi` файл напрямую, НЕ `.exe` файл!

```cmd
# Правильно - показывает UI
Prizrak-Box-1.0.x-alpha8.msi

# Неправильно - может запускать в тихом режиме
Prizrak-Box-1.0.x-alpha8-setup.exe
```

### Причина 2: Upgrade существующей установки

Если у вас уже установлена предыдущая версия Prizrak-Box, MSI может выполнять upgrade без UI.

**РЕШЕНИЕ:** Удалите предыдущую версию через "Установка и удаление программ", затем запустите новый установщик.

### Причина 3: Запуск через командную строку с параметрами

Если установщик был запущен с параметрами `/quiet`, `/qn` или `/passive`, UI не показывается.

**РЕШЕНИЕ:** Запускайте MSI без параметров или с параметром `/qf` (полный UI):

```cmd
# Полный UI (все диалоги)
msiexec /i Prizrak-Box-1.0.x-alpha8.msi

# Или просто двойной клик по .msi файлу
```

### Причина 4: Кэшированный установщик

Windows может кэшировать установщик и запускать старую версию.

**РЕШЕНИЕ:** Очистите кэш установщика:

```cmd
# Удалите кэш MSI
rd /s /q %WINDIR%\Installer\$PatchCache$

# Или просто перезагрузите компьютер
```

## Правильный способ запуска установщика

### Метод 1: Двойной клик (рекомендуется)

1. Откройте папку `out/make/wix/x64/`
2. Найдите файл `Prizrak-Box-<version>.msi`
3. Дважды кликните по нему
4. Должны появиться диалоги:
   - Welcome
   - License Agreement (GPL3)
   - Custom Setup (Feature Tree)
   - Installation Directory
   - Ready to Install

### Метод 2: Через командную строку

```cmd
# Полный UI
msiexec /i "C:\path\to\Prizrak-Box-1.0.x-alpha8.msi"

# С логированием (для отладки)
msiexec /i "C:\path\to\Prizrak-Box-1.0.x-alpha8.msi" /l*v install.log

# С выбором языка (русский)
msiexec /i "C:\path\to\Prizrak-Box-1.0.x-alpha8.msi" TRANSFORMS=:1049
```

## Ожидаемая последовательность диалогов

Если все работает правильно, вы должны увидеть:

1. ✅ **Welcome Dialog** - "Welcome to the Prizrak-Box (Machine - MSI) Setup Wizard"
2. ✅ **License Agreement** - GPL3 лицензия с чекбоксом "I accept the terms in the License Agreement"
3. ✅ **Custom Setup** - Дерево функций:
   ```
   [✓] Prizrak-Box (1.0.x)
       [✓] Main Application
       [✓] TUN Service Mode
   ```
4. ✅ **Installation Folder** - Выбор папки установки
5. ✅ **Ready to Install** - Подтверждение настроек
6. ✅ **Installing** - Прогресс установки
7. ✅ **Completed** - Завершение

## Отладка проблемы

Если UI все еще не показывается, создайте лог установки:

```cmd
msiexec /i Prizrak-Box-1.0.x-alpha8.msi /l*vx install.log
```

Откройте `install.log` и найдите:
- `UILevel` - должно быть `5` (Full UI) или `3` (Reduced UI)
- Если `UILevel = 2` (Basic UI) или `UILevel = 0` (Silent), значит что-то переопределяет UI

## Параметры запуска MSI

```cmd
# Полный UI (все диалоги) - по умолчанию
msiexec /i installer.msi

# Уровни UI
/qf   - Full UI
/qr   - Reduced UI (только прогресс)
/qb   - Basic UI (только прогресс с кнопкой Cancel)
/qn   - No UI (тихая установка)
/passive - Прогресс без взаимодействия

# С логированием
/l*v log.txt   - Verbose лог
/l*vx log.txt  - Extra verbose лог
```

## Проверка после установки

После установки проверьте:

```cmd
# Проверить установленные программы
wmic product where "name like '%Prizrak%'" get name,version

# Проверить службу TUN (если была выбрана)
sc query PrizrakBoxService
```

## Контакты для помощи

Если проблема не решается:
1. Создайте issue на GitHub с приложенным `install.log`
2. Укажите версию Windows и способ запуска установщика
3. Приложите скриншоты (если были)
