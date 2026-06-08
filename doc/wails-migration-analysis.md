# Анализ миграции Prizrak-Box: Electron → Wails

> Документ подготовлен для оценки трёх вопросов:
> 1. Возможность миграции десктоп-оболочки с Electron на Wails.
> 2. Отделение бэкенда приложения (`px`) от ядра **moshen** (`v1.19.x-smart`).
> 3. Целесообразность переноса фронтенда/функционала на базу **Nemu-x/SlothClash** (там уже сделана миграция Tauri → Wails).

---

## 0. TL;DR (краткие выводы)

1. **Ядро moshen УЖЕ отделено** — это самостоятельный Go-модуль `github.com/legiz-ru/moshen`, подключённый к `src-go` через `replace`-директиву. Оно физически НЕ лежит внутри `src-go`. Внутри `src-go` лежит **бэкенд `px`**, который *импортирует* moshen как библиотеку.
2. **Бэкенд уже автономен** — `px` и `px-service` собираются как отдельные бинарники и кладутся в Electron через `extraResource`. Фронтенд общается с `px` по **HTTP/REST** (axios + Bearer-secret) почти на 100%. Electron-IPC используется лишь для ~15–20 «системных» вызовов.
3. **Миграция Electron → Wails реалистична и относительно недорога**, потому что переписывать нужно только оболочку (~13 TS-файлов `src-electron/`), а не бэкенд и не фронтенд. Архитектура «GUI + sidecar-бинарник mihomo + привилегированный сервис» — ровно та же, что у SlothClash на Wails.
4. **Перенос на базу SlothClash — НЕ проще, а дороже и рискованнее.** SlothClash — это React + наследие clash-verge-rev с *другой* моделью интеграции ядра (vanilla-mihomo как sidecar + штатный `external-controller`). У Prizrak — Vue 3 + наследие Pandora-Box + **глубокая** интеграция moshen как библиотеки. Перенос означал бы переписывание фронта (React↔Vue) и потерю «умной» интеграции ядра.

**Рекомендация:** идти по пути **A (Electron → Wails, сохраняя `px`/`px-service` как есть)**, а не по пути B (rebase на SlothClash).

---

## 1. Текущая архитектура

```
┌──────────────────────────────────────────────────────────────────┐
│  Electron (src-electron/, ~13 .ts файлов)                          │
│  • Окно/трей/меню, deep-link, автозапуск, single-instance          │
│  • Спавнит и эскалирует px (admin.ts), управляет px-service        │
│  • Express-сервер раздаёт собранный фронт (server.ts)              │
│  • preload.ts → contextBridge → window.px* (IPC-мост)              │
└───────────────┬───────────────────────────────┬──────────────────┘
                │ spawn + IPC                    │ грузит фронт
                ▼                                 ▼
┌───────────────────────────┐      ┌──────────────────────────────────┐
│ px  (src-go/, бинарник)    │◄────►│ Frontend (src/, Vue 3 + Vite)     │
│ • main.go → prizrak.Start  │ HTTP │ • axios → http://127.0.0.1:PORT   │
│ • api/ REST-хендлеры       │ REST │   + Authorization: Bearer SECRET  │
│ • internal/ логика профилей│ +secret│ • Element Plus, Pinia, vue-i18n │
│ • импортирует moshen (lib) │  ▲    │ • тонкий слой window.px* (IPC)    │
└───────────┬────────────────┘  │    └──────────────────────────────────┘
            │ IPC (pipe/socket)  │ callback /pxStore?port&secret
            ▼                    │
┌───────────────────────────┐   │   ┌──────────────────────────────────┐
│ px-service (src-service/)  │   └───│ Ядро moshen (ОТДЕЛЬНЫЙ модуль)    │
│ • system service (admin)   │       │ github.com/legiz-ru/moshen        │
│ • спавнит px для TUN       │       │ v1.19.26-smart-moshen (replace)   │
└───────────────────────────┘       │ форк MetaCubeX/mihomo             │
                                     └──────────────────────────────────┘
```

### 1.1 Фронтенд (`src/`)
- **Стек:** Vue 3.5, Vite 7, Pinia 3 (+ persistedstate), vue-router 4, vue-i18n 11, Element Plus 2.9, ApexCharts.
- **Связь с бэкендом — HTTP/REST.** `src/util/axiosRequest.ts`: `baseURL = http://host:port`, заголовок `Authorization: Bearer <secret>`. Параметры `host/port/secret` приходят из query-строки, которую формирует Electron при загрузке окна.
- Все 8 модулей в `src/api/` (`home`, `profiles`, `proxies`, `connections`, `dns`, `rule`, `mihomo`, `prizrak`) — это чистые HTTP-вызовы. **Этот слой при миграции не меняется.**

### 1.2 Electron-IPC: что именно завязано на оболочку
Тонкий мост в `src-electron/preload.ts` (`window.px*`). Места вызова во фронте (это и есть то, что придётся перенести на Wails-биндинги):

| Группа | Глобал / канал | Файлы-потребители |
|---|---|---|
| TUN-сервис | `window.pxService.{getStatus,install,uninstall,restartBackend,...}` | `components/menu/MyProxy.vue`, `components/setting/MyService.vue` |
| Хранилище | `window.pxStore.{get,set}` | `types/persist.ts` (все Pinia-стора), `main.ts` |
| Каталог конфигов | `electron.invoke('select-directory')`, `pxPreConfigDir`, `pxChangeConfigDir`, `pxConfigDir` | `components/setting/MyConfig.vue` |
| Трей (двусторонне) | `window.pxTray.{on,emit}` | `runtime/index.ts`, `main.ts` |
| Deep-link | `window.pxDeepLink.{onImportProfile,notifyReady}` | `main.ts` |
| Системное | `pxOs`, `pxUsername`, `pxClipboard`, `pxOpen`, `pxShowInFolder`, `pxShowBar`, фон `pxBgCache` | `runtime/index.ts`, `App.vue`, `main.ts` |

Итого: **~15–20 биндингов** + событийная система трея + deep-link + автозапуск + управление сервисом.

### 1.3 Бэкенд `px` (`src-go/`)
- `main.go` → `prizrak.StartCore(addr)`: регистрирует REST-хендлеры **в chi-роутере самого moshen** (`route.Register(...)`, `route.StartByPandoraBox(host,port,secret,cors)`), затем «откликается» Electron-у через `GET <addr>/pxStore?port=PORT&secret=SECRET`.
- Слушает `127.0.0.1:9686` (или свободный порт), аутентификация — 16-символьный secret (хранится в BoltDB).
- Это **отдельный процесс**, который оболочка спавнит: `px -addr=127.0.0.1:XXXX -home=<userData>`. Singleton через `px-server.pid`.
- Эндпоинты: `/version`, `/profile/*`, `/rule/*`, `/mihomo/*`, `/prizrak/*`, `/dns/*`, `/webtest/*`, `/ws` (статистика/логи по WS).

### 1.4 `px-service` (`src-service/`)
- Системный сервис (`github.com/kardianos/service`) с привилегиями; нужен, чтобы включать **TUN без запуска всего приложения от админа**.
- IPC: named pipe `\\.\pipe\prizrak-box-service` (Windows) / unix-socket `/tmp/prizrak-box-service.sock`. Команды: `ping`, `version`, `is_admin`, `start_px`, `stop_px`, `status`.
- **Независим от оболочки** — при миграции переносится «как есть».

### 1.5 Сборка/CI (`forge.config.ts`, `.github/workflows/release.yml`)
- `extraResource` кладёт `px`(`.exe`) и `px-service`(`.exe`) в ресурсы приложения. На macOS бинарники подписываются отдельно (`codesign`).
- CI собирает: backend (`-tags=with_gvisor`, встраивает geo-базы и `Model.bin` в `src-go/internal/em`), `px-service`, затем electron-forge паковка. Android — отдельная сборка ClashMetaForAndroid.
- Артефакты: Win (msi/zip ×amd64/arm64), macOS (dmg/zip), Linux (deb/rpm/AUR), Android (apk).

---

## 2. Ключевой вывод про «отделение бэкенда и ядра moshen»

В постановке задачи сказано: *«сейчас оно общее внутри директории `src-go`»*. На практике картина другая:

```gomod
// src-go/go.mod
require github.com/metacubex/mihomo v1.19.23
...
replace github.com/metacubex/mihomo => github.com/legiz-ru/moshen v1.19.26-smart-moshen
```

- **Ядро moshen — уже самостоятельный репозиторий/модуль** (`github.com/legiz-ru/moshen`, форк `MetaCubeX/mihomo`, тег `v1.19.26-smart-moshen`). Внутри `src-go` его исходников нет — только зависимость через `replace`.
- Внутри `src-go` лежит **прикладной бэкенд `px`**: REST-API (`api/`), логика профилей/подписок (`internal/`), утилиты (`pkg/`). Он *использует* moshen.

То есть «отделение ядра от бэкенда» **на уровне модулей уже выполнено.** Версионируется ядро независимо (свой тег `-smart-moshen`), обновляется бампом `replace`.

### Что реально «сцеплено» (и это нормально)
`px` интегрирует moshen **как библиотеку, на уровне Go-API**, а не как внешний бинарник:

| Точка интеграции | Где | Степень связности |
|---|---|---|
| `route.Register` / `route.StartByPandoraBox` | `prizrak/core.go` | Prizrak-эндпоинты живут **внутри HTTP-сервера ядра** |
| `config.ParseRawConfig`, сборка `RawConfig`, мердж профилей, prizrak-оверрайды (имя TUN «Prizrak», DNS-hijack) | `internal/meta.go` | **Высокая** — прямая работа со структурами ядра |
| `executor.Patch` / `executor.Shutdown` (hot-reload) | `internal/meta.go`, `main.go` | **Высокая** |
| `adapter.ParseProxy`, `convert.ConvertsV2Ray` (валидация/парсинг подписок) | `internal/resolve.go` | Средняя |
| `mmdb`/geo, `log`, `tunnel`, `statistic` | разное | Низкая–средняя |

**Вывод:** дальнейшее «отделение» бэкенда от ядра *глубже текущего* (т.е. чтобы `px` не зависел от внутренних API moshen) потребовало бы переписать конвейер сборки `RawConfig`, валидацию прокси и hot-reload поверх внешнего HTTP-контроллера ядра — это **6–10 недель** и фактический отказ от «умной» интеграции. **Делать это в рамках миграции на Wails не нужно и вредно.** Текущая модель «`px` = кастомный mihomo + prizrak-API» — это сильная сторона проекта (наследие Pandora-Box), её стоит сохранить.

---

## 3. Вариант A — Electron → Wails (рекомендуемый)

**Идея:** заменить только оболочку. `px`, `px-service`, фронтенд Vue, ядро moshen — без изменений по сути. Wails-приложение (Go) спавнит `px` так же, как сейчас это делает `admin.ts`.

### 3.1 Почему дёшево
- Фронтенд общается с `px` по HTTP — **в Wails это работает без изменений** (тот же `axios` на `127.0.0.1:port` с secret). Wails отдаёт фронт сам (assetserver), Express (`server.ts`) больше не нужен.
- `px`/`px-service` остаются отдельными бинарниками — меняется лишь способ их доставки (Wails `embed`/ресурсы вместо `extraResource`) и код спавна (Go `os/exec` вместо Node `spawn`).
- Это та же архитектура, что у **SlothClash на Wails** (подтверждено: mihomo-sidecar + привилегированный сервис + динамический порт + secret-аутентификация).

### 3.2 Что придётся переписать (Go вместо TS)
| Модуль Electron | Объём | Чем заменить в Wails |
|---|---|---|
| `admin.ts` (спавн+эскалация `px`) | ~300 строк | Go `os/exec`; эскалация: Win `runas`/манифест, mac `osascript`, Linux `pkexec`/`polkit` |
| `service.ts` (IPC к px-service) | ~400 строк | Go-клиент named pipe/unix-socket (логика уже есть в `src-service/ipc`) |
| `tray.ts` (трей + меню + live-обновления) | ~495 строк | Wails v2 — нативного меню трея нет «из коробки»: `getlantern/systray` или Wails v3 (где трей нативный) |
| `main.ts` (lifecycle, окно, deep-link, single-instance, UA-spoofing) | ~485 строк | Wails `options.App`, события, протокол-хендлер; single-instance — `wails-single-instance`/файл-лок |
| `launch.ts` (автозапуск) | ~200 строк | `emersion/go-autostart` или ручной `.desktop`/реестр/LaunchAgent (часть логики уже на Go в `px-service`) |
| `preload.ts` (`window.px*`) | ~104 строки | Wails-биндинги Go-структуры + `runtime.EventsOn/Emit` |
| `store.ts` (electron-store) | ~30 строк | JSON-файл/BoltDB на Go |
| `server.ts` (Express) | ~180 строк | не нужен (assetserver Wails); загрузку кастомного фона — на Go-хендлер |
| `log.ts`, `shortcut.ts`, `selector.ts`, `change.ts` | мелочь | `log/slog`, глобальные шорткаты (по необходимости), `runtime.OpenDirectoryDialog`, перенос каталога на Go |

### 3.3 Фронтенд: точечные правки
Создать тонкий адаптер взамен `window.px*`. Удобно сохранить **те же имена/сигнатуры**, чтобы остальной код не трогать:
- `runtime/index.ts`, `types/persist.ts`, `main.ts`, `MyProxy.vue`, `MyService.vue`, `MyConfig.vue`, `App.vue` — заменить вызовы `window.px*`/`electron.invoke` на Wails-биндинги (`window.go.main.App.*`) и `runtime.Events*`.
- Часть можно оставить на «вебных» аналогах: `pxOpen` → `BrowserOpenURL`, `pxClipboard` → Clipboard API/Wails, `pxOs` → биндинг или `navigator`.

### 3.4 Риски варианта A
- **Трей в Wails v2 слабый.** Самый чувствительный момент: у Prizrak богатое контекстное меню трея (режимы, профили, прокси-группы, дашборды, эмодзи-иконки). Нужно либо `systray`, либо ждать/брать **Wails v3** (нативный трей-API). Это главный технический риск.
- **Deep-link** (`prizrak-box://`) — регистрация схемы на 3 ОС + single-instance проксирование URL. Решаемо, но требует платформенного кода.
- **macOS подписание/нотаризация** — перенастройка пайплайна (Wails использует свои упаковщики; gon/notarytool).
- **Размер/UX** — плюс Wails: бинарник меньше и легче по RAM (нет Chromium у Electron; используется системный WebView2/WebKit/WebKitGTK). Минус: различия рендеринга между WebView2/WebKitGTK (тестировать UI на всех ОС).

### 3.5 Оценка трудозатрат (вариант A)
| Блок | Оценка |
|---|---|
| Каркас Wails + сборка фронта + assetserver | 0.5 нед |
| Спавн/эскалация `px` + интеграция `px-service` (Go-клиент уже есть) | 1–1.5 нед |
| Трей + меню (live-обновления, эмодзи) | 1.5–2.5 нед (главный риск) |
| Окно/lifecycle/single-instance/deep-link/UA-spoofing | 1–1.5 нед |
| Автозапуск (3 ОС) + хранилище + каталог конфигов | 1 нед |
| Адаптация фронта (замена `window.px*`) | 0.5–1 нед |
| CI/CD: 3 ОС × 2 арх, подпись/нотаризация, инсталляторы | 1.5–2 нед |
| Тестирование на Win/mac/Linux | 1–1.5 нед |
| **Итого** | **≈ 8–12 недель** одного разработчика |

> Android (ClashMetaForAndroid) миграция не затрагивает.

---

## 4. Вариант B — перенос фронта/функционала на базу SlothClash

### 4.1 Что такое SlothClash (по факту репозитория)
- **Wails v2**, фронт — **React + TypeScript** (pnpm), форк **clash-verge-rev** (наследие Tauri, переписан на Wails).
- Ядро — **vanilla `MetaCubeX/mihomo` как sidecar-бинарник** (`core_manager.go`: `exec.CommandContext(ctx, bin, "-d", dataDir)`), управление через **штатный external-controller** mihomo (`PUT /configs?force=true`, `GET /version`), динамический порт + secret, авто-рестарт, привилегированный сервис, схема `slothclash://`.

### 4.2 Принципиальная несовместимость моделей
| Аспект | Prizrak-Box | SlothClash |
|---|---|---|
| Фронтенд | **Vue 3** + Element Plus | **React** + наследие clash-verge |
| Ядро | **moshen** (форк) как **Go-библиотека** внутри `px` | **vanilla mihomo** как **внешний бинарник** |
| Интеграция конфигов | сборка `RawConfig` в процессе + `executor.Patch` | запись `config.yaml` + `PUT /configs` |
| Prizrak-API | кастомные `/profile`, `/rule`, `/prizrak/*` **внутри ядра** | штатный API mihomo |
| Наследие | Pandora-Box | clash-verge-rev |

### 4.3 Почему это дороже, а не проще
1. **React ↔ Vue.** Весь фронт Prizrak (views, components, Pinia-стора, Element Plus, i18n) пришлось бы **переписать на React** или выкинуть фронт SlothClash и поставить Vue (тогда от SlothClash остаётся только Go-слой — а его проще написать самим под текущую модель).
2. **Потеря «умной» интеграции ядра.** SlothClash гоняет ваниль-mihomo как sidecar. Чтобы сохранить moshen-`smart` и кастомную логику профилей/подписок/мерджа (`internal/meta.go`, `resolve.go`), пришлось бы либо тащить `px` внутрь SlothClash (тогда зачем SlothClash), либо переписывать всю prizrak-логику поверх внешнего контроллера (те самые 6–10 недель из §2).
3. **Двойная мердж-боль.** Сведение двух разных Go-архитектур (core-manager SlothClash vs prizrak-`px`) + разные схемы deep-link, сервисов, хранилищ.

### 4.4 Что из SlothClash полезно (как референс, не как база)
SlothClash — **отличный образец того, как сделать вариант A**. Стоит позаимствовать **паттерны кода** (не кодовую базу):
- `core_manager.go` — спавн ядра, выбор свободного порта, поллинг `GET /version` до готовности, авто-рестарт при падении.
- `core_pipe_transport_*.go` — IPC к привилегированному сервису (у Prizrak уже есть аналог в `src-service/ipc`).
- `app_lifecycle_{darwin,windows}.go` — платформенные lifecycle-хуки в Wails.
- `app_update.go` — автообновление.
- Регистрация `slothclash://` — как делать deep-link в Wails.

### 4.5 Оценка трудозатрат (вариант B)
| Блок | Оценка |
|---|---|
| Перенос фронта Vue→React **или** замена фронта SlothClash на Vue | 6–10 нед |
| Сведение Go-архитектур, перенос prizrak-логики на модель SlothClash | 6–10 нед |
| Перенос кастомных фич (РФ-функционал, мердж профилей, smart-ядро) | 3–5 нед |
| CI/CD, упаковка, подпись, тесты | 2–3 нед |
| **Итого** | **≈ 17–28 недель**, с высоким риском регрессий |

---

## 5. Сводное сравнение вариантов

| Критерий | A. Electron→Wails (свой шелл) | B. База SlothClash |
|---|---|---|
| Фронтенд | Vue остаётся (правки точечные) | переписать (React) или выкинуть фронт SC |
| Ядро moshen-smart | сохраняется как есть | теряется/переписывается |
| Prizrak-логика (`internal/`) | без изменений | портировать на другую модель |
| `px-service` | как есть | сводить с core-manager SC |
| Объём работ | **8–12 недель** | 17–28 недель |
| Риск | средний (главное — трей) | высокий |
| Выигрыш (RAM/размер) | да | да |
| Сохранение идентичности проекта | полное | низкое |

---

## 6. Рекомендация и поэтапный план (вариант A)

**Мигрировать оболочку на Wails, сохранив `px` + `px-service` + moshen + фронт Vue. SlothClash использовать как референс реализации, а не как базу.**

**Фаза 0 — PoC (1–2 нед).** Каркас Wails v2/v3; собрать фронт Vite в assets; запустить spawn `px` и проверить, что фронт через HTTP+secret работает в WebView. Сразу проверить трей (главный риск) — `systray`/Wails v3.

**Фаза 1 — системный слой (2–3 нед).** Перенести `admin.ts` (спавн+эскалация), интеграцию `px-service`, single-instance, deep-link, автозапуск.

**Фаза 2 — биндинги и трей (2–3 нед).** Go-структура с методами под `window.px*`; `runtime.Events*` для двусторонних трей-событий; меню трея с live-обновлением (режимы/профили/группы/дашборды).

**Фаза 3 — фронт-адаптер (0.5–1 нед).** Заменить `window.px*`/`electron.invoke` на Wails-биндинги, сохранив имена/сигнатуры.

**Фаза 4 — CI/CD и упаковка (2–3 нед).** Wails-сборка под Win/mac/Linux ×amd64/arm64; подпись/нотаризация; msi/dmg/deb/rpm/AUR; доставка `px`/`px-service` как ресурсов.

**Фаза 5 — стабилизация (1–2 нед).** Тесты UI на WebView2/WebKit/WebKitGTK, TUN, прокси, deep-link, автозапуск на всех ОС.

### Параллельно (необязательно, но полезно)
- Зафиксировать контракт `px` HTTP-API (мини-OpenAPI) — тогда оболочка/ядро становятся ещё слабее связаны, и любые будущие миграции тривиальны.
- Решение по Wails **v2 vs v3**: v3 даёт нативный трей и мульти-окна (снимает главный риск), но на момент анализа стоит проверить его стабильность.

---

## 7. Ответы на исходные вопросы

1. **Миграция Electron→Wails возможна** и оправдана: переписывается только оболочка (~13 TS-файлов), фронт и бэкенд переиспользуются. Оценка 8–12 недель.
2. **Отделять бэкенд от ядра дополнительно не требуется** — moshen уже отдельный Go-модуль (`replace` → `legiz-ru/moshen`). Глубокая связность `px`↔moshen (через `executor`/`RawConfig`) — это осознанная и полезная архитектура (наследие Pandora-Box), её следует сохранить.
3. **Переносить на базу SlothClash не проще, а дороже** (React-фронт + другая модель интеграции ядра, 17–28 недель, потеря smart-ядра). SlothClash стоит использовать как **референс** для реализации варианта A.
