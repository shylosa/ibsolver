# IBSolver — правила для агента

## Порядок роботи

- Працюй одним агентом за замовчуванням. Залучай субагентів за прямим запитом або обґрунтованою потребою, з урахуванням обмежень завдання.
- Читай лише потрібні файли, функції та залежності. Внось мінімальні достатні зміни. Збережи наявні незакомічені правки й уникай стороннього рефакторингу та зайвих абстракцій.
- Перед нетривіальними змінами коротко визнач файли, причину змін і команди перевірки. Для описок і змін коментарів окремий план не потрібен. Обґрунтовуй нові залежності й додавай фреймворки, розгорнуті звіти, плани чи класифікації завдань лише за потреби.
- Використовуй `D:\go\ibsolver-refs` лише для читання потрібних BAS/XLS-джерел. Не змінюй оригінали й не копіюй їх до репозиторію.
- Для чисельних змін перевіряй фізичну коректність, одиниці, точність і відтворюваність. Економія ресурсів не скасовує необхідних перевірок.
- Після успішної перевірки додай короткий запис із датою, описом змін і результатами перевірок у «Журнал змін» у [ROADMAP.md](ROADMAP.md). У підсумку коротко повідом про зміни, перевірки й залишкові обмеження.
- Коміти та `git push` виконуються лише за прямим запитом користувача.

---

## Призначення і архітектура

**IBSolver** — настільний застосунок для Windows 10/11 (Go + Wails v2 + SQLite).
Мета: введення, збереження та чисельний розрахунок фізичних конфігурацій.
Три режими: **preparation** (підготовчі дані, балістика OZVB-3A/Slukhotsky),
**oscillator** (m·x″+c·x′+k·x=0, RK45), **demo** (математичні криві, тестування UI).

```
ibsolver/
├── main.go                     # Точка входу Wails
├── app.go                      # Тонкий Wails bridge (єдина точка Go→JS API)
├── exchange.go                 # JSON-обмін конфігураціями (import/export)
├── external.go                 # Імпорт CSV зовнішніх результатів
├── internal/
│   ├── domain/                 # Типи й валідація — без Wails, без SQLite
│   │   ├── model.go            # State, Configuration, Run, Settings, ChartData, Series
│   │   ├── preparation.go      # PreparationInputs (7 полів: d, q, vd, lc, ns, f1, af)
│   │   ├── numerical.go        # NumericalExample, NumericalSummary, NumericalVersion
│   │   ├── series.go           # SeriesID(), BindSeries()
│   │   └── external.go         # ExternalResult
│   ├── application/            # Координація сценаріїв (mutex, cancel, snapshot)
│   │   ├── service.go          # Calculate, Load, Save*, CancelCalculation, CreateDemo
│   │   ├── exchange.go         # validateExchange, completeConfigurationJSON
│   │   └── external.go         # PreviewExternalImport, SaveExternalImport, DeleteExternal
│   ├── storage/sqlite/         # SQL, міграції v1→v3, транзакції, відновлення
│   │   ├── store.go            # Open, migrate, SaveConfiguration, SaveRun, Load, latestRun
│   │   ├── parameters.go       # Typed envelope format:1 (encode/decode)
│   │   └── external.go         # SaveExternal, ListExternal, DeleteExternal
│   ├── chartdata/              # Генерація часових рядів (6 серій на запуск)
│   │   ├── ballistics.go       # OZVB-3A IVP → P, T, V, L, PSI, G
│   │   ├── oscillator.go       # Адаптер ode.Integrate → 6 серій (x, v, a, Ek, Ep, E)
│   │   └── demo.go             # Математичні демо-криві
│   └── ode/
│       └── rk45.go             # Dormand-Prince 5(4) — без жодних залежностей
├── frontend/
│   ├── src/
│   │   ├── main.js             # Основний модуль UI (Vanilla JS)
│   │   ├── chart.js            # Canvas компонент TimeSeriesChart (zoom/pan/cursor)
│   │   ├── preparation.mjs     # preparationFields, parsePreparation, displayPreparation
│   │   ├── external.mjs        # externalMetadata (метадані CSV-імпорту)
│   │   └── debounce.mjs        # DebouncedTask (autosave 450 мс)
│   └── wailsjs/                # Автогенеровані Go→JS bindings (не редагувати вручну)
├── docs/
│   ├── EDITOR_CONTRACT.md      # Технічний контракт Stage 6A (підготовчі дані)
│   ├── LEGACY_SPEC.md          # Специфікація BAS/XLS-джерел (тільки читання)
│   └── SOURCES.md
├── AGENTS.md                   # Цей файл — правила для агента
└── ROADMAP.md                  # Живий трекер стану
```

---

## Технологічний стек

| Компонент | Версія | Примітка |
|-----------|--------|----------|
| Go | 1.27.1 | `go.mod` |
| Wails | v2.16.0 | desktop shell, Go↔JS bridge |
| `modernc.org/sqlite` | v1.60.1 | CGO-free SQLite |
| Vite | ^7.0.0 | frontend bundler |
| Node.js test runner | built-in | `node --test src/*.test.mjs` |
| WebView2 | системний | потрібен на машині кінцевого користувача |

Немає: ORM, UI-фреймворків, зовнішніх CSS-бібліотек, будь-яких Go web-фреймворків у production-коді.

---

## Конвенції коду

Код, ідентифікатори й коментарі пишемо англійською. Інтерфейс, повідомлення про помилки для UI та спілкування з користувачем ведемо українською. Технічні повідомлення пишемо англійською.

### Go

- **Обробка помилок:** `fmt.Errorf("...: %w", err)` для wrapping; повертати `error` у caller, не логувати всередині.
- **Пакет `ode`**: без залежностей від `domain`, `storage`, `application`. Не додавати.
- **Пакет `domain`**: без Wails і SQLite. Валідація через методи `Validate() error` на типах.
- **Bridge (`app.go`)**: тонкий — лише `ready()` guard і делегування до `service`. Парсинг і валідація не переходять у bridge.
- **Мʼютекс у `Service`**: `mu sync.Mutex` захищає store і cancel. Розблоковувати перед довгими операціями (розрахунком).
- **Числа**: лише скінченні `float64`; перевіряти `math.IsNaN` / `math.IsInf` перед збереженням.
- **Іменування**: `PascalCase` для exported, `camelCase` для unexported, пакети lowercase без підкреслень.

### JavaScript (Frontend)

- **Vanilla JS, ES modules** (`.mjs` для окремих модулів, `.js` для основних).
- **Немає фреймворків**: без React, Vue, jQuery.
- **Autosave**: `DebouncedTask` з 450 мс; черга `draftQueue` для серіалізації запитів.
- **DOM**: хелпер `const $ = id => document.getElementById(id)`.
- **Парсинг чисел**: власний парсер у `preparation.mjs` — не `parseFloat` з прийняттям префікса; підтримка коми/крапки як роздільника.
- **Canvas**: `TimeSeriesChart` у `chart.js` — не модифікувати без потреби змінювати візуалізацію.
- Оновлюй автогенеровані файли в `frontend/wailsjs/` через Wails.

### SQL / SQLite

- Виконуй транзакції через `db.Begin()`, `defer tx.Rollback()` і явний `tx.Commit()`.
- **Schema version**: `PRAGMA user_version` (зараз 3). Нова таблиця/колонка → новий `version < N` блок у `migrate()` та інкремент версії.
- **Дані серій**: gzip JSON BLOB у `run_series`. Ліміт читання — 20 MiB.
- **Незмінні зовнішні результати**: тригер `external_results_immutable` в схемі.
- **DB path**: виключно portable — `ibsolver.db` у каталозі виконуваного файлу
  (`os.Executable()` + `filepath.Dir()`, функція `DBPathNextTo` у `storage/sqlite`).
  НЕ використовувати `%APPDATA%`, `%LOCALAPPDATA%`, домашній каталог чи робочий
  каталог процесу; fallback-шляхів і створення каталогів извне не існує.

---

## Перевірка змін

Починай з адресних тестів для змінених сценаріїв. Використовуй попередні результати для частин, яких зміни не зачепили. Після кожного блоку нетривіальних змін виконай `go test ./...` і `npm test` у наведеному порядку. Виправ помилки, спричинені змінами, і повтори перевірку. Збірку Wails та інтеграційні перевірки додавай, коли цього потребує ризик змін.

```powershell
# 1. Go тести (всі пакети)
Set-Location d:\go\ibsolver
go test ./...

# 2. Frontend тести
Set-Location d:\go\ibsolver\frontend
npm test

# 3. Перевірка збірки (опціонально при змінах у Go або frontend)
Set-Location d:\go\ibsolver
wails build -trimpath
```

Налаштування лінтерів для Go та frontend відстежується в [ROADMAP.md](ROADMAP.md).

Команда для dev-режиму з автоматичним оновленням.

```powershell
Set-Location d:\go\ibsolver
wails dev
```

Для встановлення frontend-залежностей за lockfile використовуй `npm ci`. За наявності всіх пакунків у локальному кеші можна виконати `npm ci --offline`.

```powershell
Set-Location d:\go\ibsolver\frontend
npm ci
```

Якщо sandbox обмежує запис у стандартні кеші, виконай ці команди з кореня репозиторію.

```powershell
$env:GOCACHE = "$PWD\.cache\go-build"
$env:npm_config_cache = "$PWD\.cache\npm"
```
