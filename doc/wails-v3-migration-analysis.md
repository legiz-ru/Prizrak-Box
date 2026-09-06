# Вариант A на Wails v3: детальный анализ миграции Prizrak-Box

> Дополнение к `doc/wails-migration-analysis.md`. Здесь вариант A (миграция только оболочки, с сохранением `px`/`px-service`/moshen/фронта Vue) прорабатывается **сразу под Wails v3**, чтобы использовать нативный трей и другие нововведения v3.
>
> Статус Wails v3 на момент анализа (июнь 2026): **alpha** (последняя `v3.0.0-alpha.92`, 15.05.2026). API «reasonably stable», есть приложения в проде, команда дорабатывает документацию и тулинг перед релизом. Это **главный риск** варианта (см. §5).

---

## 0. Зачем именно v3 (а не v2)

В анализе v2 главным техническим риском был **трей**: в Wails v2 нет полноценного нативного системного трея с богатым меню (режимы/профили/прокси-группы/дашборды, чекбоксы, радио, сабменю, live-обновления) — пришлось бы тащить сторонний `getlantern/systray` и городить обвязку. **Wails v3 закрывает ровно эту боль нативно** и попутно даёт ещё несколько вещей, которые прямо ложатся на текущую архитектуру Prizrak.

Ключевые нововведения v3, релевантные миграции:

| Нововведение v3 | Что даёт Prizrak |
|---|---|
| **Нативный System Tray** (Checkbox/Radio/Submenu/Text/Separator, `Update()`, привязка окна, light/dark + template-иконки, `Show()/Hide()`, headless-трей с live-tooltip) | Полная замена `tray.ts` без сторонних либ — снимает главный риск v2 |
| **Services pattern** (standalone-структуры, DI, lifecycle-хуки `ServiceStartup`/`ServiceShutdown`, типизированные TS-биндинги, возможность регистрировать сырые HTTP-хендлеры) | Замена `preload.ts`/IPC и `admin.ts`/`service.ts` чистыми Go-сервисами |
| **Single Instance Lock** + колбэк `OnSecondInstanceLaunch(SecondInstanceData)` с argv второго инстанса | Замена `requestSingleInstanceLock` + проксирование deep-link-аргументов |
| **Custom URL Protocols** (`prizrak-box://`): на macOS авто-генерация Info.plist, на Windows — через NSIS-инсталлятор; событие `ApplicationLaunchedWithUrl`/`OpenedWithURL` | Замена deep-link-логики `main.ts` |
| **Multiple Windows** (нативные доп. окна) | Опционально: отдельное окно настроек/онбординга |
| **Типизированная событийная система** | Замена `pxTray.on/emit` между Go и фронтом |
| **Taskfile + `wails3` CLI** (вместо `wails.json`-only) | Новый билд-пайплайн |
| Улучшения runtime: clipboard, dialogs, notifications, screens, лучшее управление окном | Закрывает `pxClipboard`, `selector.ts`, диалоги сервиса и т.п. |

Источники: [What's New in Wails v3](https://v3.wails.io/whats-new/), [System Tray Menus](https://v3.wails.io/features/menus/systray/), [Multiple Windows](https://v3alpha.wails.io/features/windows/multiple/), [Single Instance](https://v3alpha.wails.io/guides/single-instance/), [Custom URL Protocols](https://v3alpha.wails.io/guides/distribution/custom-protocols/), [Application API](https://v3alpha.wails.io/reference/application/), [Releases](https://github.com/wailsapp/wails/releases).

---

## 1. Целевая архитектура на Wails v3

```
┌──────────────────────────────────────────────────────────────────────┐
│  Wails v3 App (Go)                                                     │
│  • application.New(Options{ Services:[...], SingleInstance:..., ... })  │
│  • assetserver раздаёт собранный Vue-фронт (Express больше не нужен)    │
│  • Нативный SystemTray + Menu (live Update)                            │
│  • Services (DI + lifecycle): спавнят/контролируют px, говорят с svc   │
│  • Events: Go ⇄ фронт; ApplicationLaunchedWithUrl → deep-link          │
│  • Custom protocol prizrak-box:// ; SingleInstance + OnSecondInstance  │
└───────────────┬───────────────────────────────┬──────────────────────┘
                │ os/exec (спавн)                │ грузит фронт (WebView2/WebKit)
                ▼                                 ▼
┌───────────────────────────┐      ┌──────────────────────────────────┐
│ px  (src-go/) — без правок │◄────►│ Frontend (src/, Vue 3) — почти    │
│ REST + moshen(lib)         │ HTTP │ без правок: axios по HTTP+secret  │
└───────────┬────────────────┘ REST│ тонкий адаптер вместо window.px*  │
            │ pipe/socket        +s └──────────────────────────────────┘
            ▼
┌───────────────────────────┐
│ px-service (src-service/)  │  ← без правок; Go-клиент IPC переносится из service.ts
└───────────────────────────┘
```

Неизменными остаются: **`px`, `px-service`, ядро moshen, весь HTTP-API, фронтенд Vue по сути**. Меняется только оболочка — теперь на Go под Wails v3.

---

## 2. Маппинг модулей Electron → механизмы Wails v3

| Модуль Electron | Объём | Реализация в Wails v3 |
|---|---|---|
| `tray.ts` (трей+меню+live) | ~495 стр | **Нативный SystemTray**: `app.NewSystemTray()`, `Menu` с `AddCheckbox`/`AddRadio`/`AddSubmenu`, динамика через изменение `MenuItem` + `menu.Update()`. Режим/прокси/TUN → Checkbox; прокси в группе → Radio; профили/группы/дашборды → Submenu. Эмодзи-флаги — в тексте пунктов меню (нативные label поддерживают Unicode). Template-иконка для macOS, light/dark для остальных |
| `admin.ts` (спавн+эскалация px) | ~300 стр | **Service** `CoreService` с `os/exec`. Эскалация: Win — манифест/`runas`, macOS — `osascript ... with administrator privileges`, Linux — `pkexec`/polkit. Паттерн поллинга `GET /version` до готовности — взять из SlothClash `core_manager.go` |
| `service.ts` (IPC к px-service) | ~400 стр | **Service** `TunService`: Go-клиент named pipe/unix-socket (протокол уже определён в `src-service/ipc`). Методы биндятся во фронт типизированно |
| `preload.ts` (`window.px*`) | ~104 стр | **Сервисы v3 авто-биндятся** в типизированный TS (`bindings/`). Имена методов подобрать под текущие `window.px*`, чтобы фронт почти не трогать |
| `main.ts` lifecycle/окно | ~485 стр | `application.New(...)`, `WebviewWindowOptions`; восстановление bounds — из своего стораджа; UA-spoofing — на стороне `px`/Go HTTP-клиента, а не WebView |
| `main.ts` deep-link | — | **Custom protocol** `prizrak-box://` + событие `ApplicationLaunchedWithUrl`; на «втором инстансе» URL приходит в `OnSecondInstanceLaunch(SecondInstanceData{Args})` |
| `main.ts` single-instance | — | **SingleInstanceLock** в опциях приложения + `OnSecondInstanceLaunch` (показать окно + обработать deep-link argv) |
| `launch.ts` (автозапуск) | ~200 стр | macOS — LaunchAgent plist, Windows — реестр `Run`/Task Scheduler, Linux — `~/.config/autostart/*.desktop` (логика `.desktop` уже есть в текущем коде; часть — на Go в `px-service`). Готовых helper'ов в v3 нет — пишем сами или `emersion/go-autostart` |
| `server.ts` (Express) | ~180 стр | **Не нужен**: фронт отдаёт assetserver Wails. Загрузку кастомного фона — сырым HTTP-хендлером, **зарегистрированным сервисом v3** (v3 это умеет), либо методом сервиса, пишущим файл |
| `store.ts` (electron-store) | ~30 стр | **Service** `StoreService` поверх JSON/BoltDB. Методы `Get/Set` под Pinia-persist |
| `selector.ts` | ~10 стр | `application.OpenFileDialog().CanChooseDirectories(true)` |
| `change.ts` (смена каталога) | ~30 стр | Метод сервиса (перенос каталога + рестарт через `application.Relaunch`/перезапуск `px`) |
| `log.ts` | ~30 стр | `log/slog` на Go |
| `shortcut.ts` (глоб. шорткаты) | ~30 стр | По необходимости — сторонняя либа (в v3 нет встроенного глобального хоткея); сейчас используется только «showOrHide» |

---

## 3. Как ложится самый «больной» кусок — меню трея

Текущее меню трея Prizrak (`tray.ts`) строит:
- **Режимы** Rule/Global/Direct — взаимоисключающие → `Radio`/`Checkbox` с ручной синхронизацией;
- **System Proxy**, **TUN** — переключатели → `Checkbox`;
- **Профили** — сабменю со списком и эмодзи-иконками → `Submenu` + `Radio`/`Checkbox`;
- **Прокси-группы** — сабменю на группу, внутри радио по выбранной ноде → `Submenu` + `Radio`;
- **Дашборды** — сабменю-ссылки → `Submenu` + `Text`-пункты с `OnClick` → `BrowserOpenURL`;
- **live-обновления** по событиям из фронта (`px_mode`, `px_proxy`, `px_tun`, `px_profiles`, `px_proxyGroups`, `px_dashboards`).

В v3 это переносится 1:1:
- состояние присылается из фронта **типизированными событиями** (вместо `pxTray.emit`);
- Go-обработчик пересобирает/правит `MenuItem` и зовёт `menu.Update()` — документация явно требует звать `Update()` после изменения свойств пунктов;
- эмодзи-флаги остаются как Unicode в `label` пунктов (не нужно рисовать иконки через canvas, как в Electron);
- иконку трея ставим template-режимом на macOS (адаптация под тему) и обычной на Win/Linux; есть `SystemTray.Show()/Hide()`.

**Вывод:** на v3 трей перестаёт быть риском и становится почти прямым портом логики.

---

## 4. Сервисы v3 как замена IPC-моста

Текущий `preload.ts` экспонирует `window.px*`. В v3 это заменяется набором Go-сервисов, которые **автоматически биндятся** во фронт с типами. Предлагаемое разбиение:

- `CoreService` — спавн/стоп/рестарт `px`, проброс `host/port/secret` во фронт, поллинг готовности. Lifecycle-хук `ServiceStartup` поднимает `px` при старте, `ServiceShutdown` — гасит.
- `TunService` — `getStatus/install/uninstall/isRunning/restartBackend/showInstallDialog` поверх IPC к `px-service`.
- `StoreService` — `get/set` (Pinia persist) + `bgCache read/write/clear`.
- `SystemService` — `os/username/clipboard/openExternal/showInFolder/openConfigDir/preConfigDir/changeConfigDir/selectDirectory`.
- `TrayBridge` — приём состояния от фронта (события) и отдача кликов трея во фронт.

На фронте — тонкий адаптер `runtime/wails.ts`, реализующий те же сигнатуры, что нынешние `window.px*`, поверх сгенерированных биндингов и `Events`. Тогда `MyProxy.vue`, `MyService.vue`, `MyConfig.vue`, `persist.ts`, `runtime/index.ts`, `App.vue`, `main.ts` правятся точечно.

---

## 5. Риски, специфичные для v3

1. **Alpha-статус (главный).** API стабилизируется, но до финального релиза возможны breaking changes между alpha. Митигация: **зафиксировать конкретную версию** (`v3.0.0-alpha.92` или новее на момент старта), вести зависимость через `go.mod`, не прыгать по alpha без необходимости, заложить буфер на стабилизацию.
2. **Баг single-instance ↔ deep-link на macOS** ([issue #5089](https://github.com/wailsapp/wails/issues/5089)): SingleInstanceLock конфликтует с `OpenedWithURL`. Для Prizrak важно (схема `prizrak-box://` + single-instance). Митигация: проверить статус бага на момент старта; при необходимости — обработать URL из `OnSecondInstanceLaunch(argv)` вместо отдельного события, либо временный воркэраунд.
3. **Зрелость упаковки/подписи** в v3-тулинге (NSIS/Windows, dmg+нотаризация macOS, deb/rpm/AUR Linux) — менее обкатана, чем electron-forge. Пайплайн `release.yml` переписывается под `wails3 package`/Taskfile; нотаризацию macOS и подпись `px`/`px-service` сохраняем.
4. **Различия WebView** (WebView2 на Win, WebKit на macOS, WebKitGTK на Linux): нет Chromium как в Electron — нужно протестировать UI Element Plus/ApexCharts на всех трёх движках (особенно WebKitGTK на Linux).
5. **Нет встроенных** глобальных хоткеев и autolaunch-хелперов — пишем сами/сторонними либами (объём небольшой).
6. **WebKitGTK-зависимости на Linux** — другой набор системных пакетов в сборке/инсталляторах, чем у Electron.

---

## 6. Обновлённые оценки трудозатрат (вариант A на v3)

| Блок | v2-оценка | **v3-оценка** | Комментарий |
|---|---|---|---|
| Каркас + assetserver + фронт | 0.5 нед | 0.5 нед | + освоение Taskfile/`wails3` |
| Спавн/эскалация `px` + `px-service` | 1–1.5 нед | 1–1.5 нед | без изменений |
| **Трей + меню** | 1.5–2.5 нед | **1–1.5 нед** | нативно в v3 → дешевле |
| Окно/lifecycle/single-instance/deep-link | 1–1.5 нед | 1–1.5 нед | + риск #5089 на macOS |
| Автозапуск/хранилище/каталог | 1 нед | 1 нед | без изменений |
| Адаптация фронта (`window.px*` → биндинги) | 0.5–1 нед | 0.5–1 нед | типизированные биндинги помогают |
| CI/CD (3 ОС × 2 арх, подпись, инсталляторы) | 1.5–2 нед | **2–2.5 нед** | v3-тулинг менее обкатан |
| Тестирование на Win/mac/Linux | 1–1.5 нед | 1.5–2 нед | + риск alpha + WebKitGTK |
| **Буфер на нестабильность alpha** | — | **+1–2 нед** | специфика v3 |
| **Итого** | 8–12 нед | **≈ 9–13 недель** | трей дешевле, но alpha добавляет стабилизацию |

Чистый объём «трейной» работы на v3 заметно меньше, но alpha-статус и менее зрелый билд-тулинг съедают часть выигрыша. Итог сопоставим с v2 по срокам, но **результат качественнее** (нативный трей, мульти-окна, типизированные биндинги, чище кодовая база).

---

## 7. Рекомендуемый план (вариант A, Wails v3)

**Фаза 0 — PoC и проверка рисков (1–2 нед).**
- `wails3 init` (Vue-шаблон/ручная интеграция Vite), собрать текущий фронт в assets.
- `CoreService` спавнит `px`, фронт по HTTP+secret рендерится в WebView.
- **Сразу проверить два риска v3:** (1) трей с чекбоксами/сабменю/`Update()`; (2) deep-link + single-instance на macOS (issue #5089).
- Зафиксировать версию alpha в `go.mod`.

**Фаза 1 — системный слой (2–3 нед).** `CoreService` (эскалация), `TunService` (IPC к `px-service`), single-instance + `OnSecondInstanceLaunch`, кастомный протокол, автозапуск (3 ОС).

**Фаза 2 — трей и события (1.5–2 нед).** Полное меню трея (режимы/прокси/TUN/профили/группы/дашборды) + двусторонние события Go⇄фронт + live `Update()`.

**Фаза 3 — фронт-адаптер (0.5–1 нед).** `runtime/wails.ts` с сигнатурами текущих `window.px*`; точечные правки потребителей.

**Фаза 4 — CI/CD (2–2.5 нед).** `wails3 package`/Taskfile под Win/mac/Linux ×amd64/arm64; подпись/нотаризация; msi(NSIS)/dmg/deb/rpm/AUR; доставка `px`/`px-service` как ресурсов; перенос `release.yml`.

**Фаза 5 — стабилизация (1.5–2 нед).** Тесты UI на WebView2/WebKit/WebKitGTK, TUN, прокси, deep-link, автозапуск; обкатка alpha-нюансов.

> Android (ClashMetaForAndroid) миграция не затрагивает.

---

## 8. Решение по v2 vs v3 — вывод

- **v3 — предпочтителен** именно из-за нативного трея (главный риск v2 исчезает) и сервис-паттерна/типизированных биндингов (чище замена IPC).
- **Цена** — alpha-статус: фиксируем версию, закладываем буфер на стабилизацию, заранее проверяем баг #5089 (macOS deep-link ↔ single-instance) и зрелость упаковки.
- **Итог:** идти на **вариант A + Wails v3**, начав с PoC (Фаза 0), который за 1–2 недели подтвердит, что трей и deep-link на v3 работают для наших сценариев; после этого продолжать по плану. Если в PoC alpha окажется нестабильной под наши кейсы — откат на v3-релиз (когда выйдет) или временно на v2 с `systray` без переписывания остального.
