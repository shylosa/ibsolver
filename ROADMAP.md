# IBSolver — Roadmap

## Поточний стан

### Реалізовано (на основі аналізу коду та git-історії)

#### Ядро та інфраструктура
- [x] **Project Bootstrap** — Wails v2 scaffold, Go module `ibsolver`, Vite + Vanilla JS frontend
- [x] **Domain layer** — типи `Configuration`, `Run`, `State`, `Settings`, `ChartData`, `Series`, `Parameters`; методи `Validate()` на всіх ключових типах
- [x] **SQLite storage** — `modernc.org/sqlite` (CGO-free), міграції v1→v3 через `PRAGMA user_version`, транзакційний запис, gzip-стискання серій
- [x] **Application service** — `Service` з `sync.Mutex`, snapshot-ізоляція розрахунку, `context.CancelFunc` для переривання
- [x] **Wails bridge** — тонкий `app.go`, `beforeClose` guard для незбережених чернеток

#### Режими розрахунку
- [x] **ODE solver** — Dormand-Prince RK45 5(4), адаптивний крок, `context.Context` cancellation, типізовані помилки (`ode.Error`)
- [x] **Oscillator mode** — m·x″+c·x′+k·x=0, 6 серій (x, v, a, Ek, Ep, E), аналітична верифікація тестами
- [x] **Demo mode** — 6 математичних кривих для перевірки інтерфейсу
- [x] **Ballistics mode (preparation)** — OZVB-3A/Slukhotsky IVP, 6 серій (P, T, V, L, PSI, G); підготовчі дані (7 полів: d, q, vd, lc, ns, f1, af)

#### UI та UX
- [x] **Canvas chart** — `TimeSeriesChart`, zoom/pan колесом і перетягуванням, подвійний клік — повний діапазон, cursor з вихідними значеннями
- [x] **Легенда** — видимість і масштаб (0,1…10) кожної серії; нормалізація `y / Reference × scale`
- [x] **Autosave** — debounce 450 мс, `draftQueue` для серіалізації, `revision` для відхилення застарілих відповідей
- [x] **Перемикання конфігурацій** — зберігає всі три блоки параметрів (preparation, oscillator, demo) незалежно
- [x] **Регульований розділювач** — ширина панелі результатів 180–480 px, зберігається глобально
- [x] **Темна тема, українська мова** — повністю україномовний UI, темний фон (#131922)

#### Обмін даними
- [x] **JSON import/export** — формат `ibsolver-configuration` v1/v2, перевірка всіх блоків, ліміт 1 MiB
- [x] **CSV export** — всі серії останнього збереженого запуску з одиницями та `float64` точністю
- [x] **CSV import (зовнішні результати)** — UTF-8, кома, 1–6 серій, 2–20 000 точок, до 8 MiB; SHA-256 checksum; незмінна таблиця SQLite v3

#### Тести
- [x] **Go**: `go test ./...` — 6 пакетів (domain, ode, chartdata, application, storage/sqlite, root)
  - Аналітичні розвʼязки з верифікацією похибок (rtol=1e-9: max |err_x| ≈ 3.3e-9 м)
  - Фізичні інваріанти осцилятора (баланс сил, збереження енергії)
  - SQLite міграції v1→v2→v3, відкат невдалого запису, незмінність знімків
- [x] **Frontend**: `npm test` — `src/*.test.mjs` (нормалізація Chart, parserPreparation, debounce)

---

## В роботі / Наступні кроки

### Пріоритет: налаштування якості коду
- [ ] Налаштувати `golangci-lint` (`.golangci.yml`) та додати до CI/verification loop
- [ ] Налаштувати ESLint або Biome для frontend

### Функціональність
- [ ] _(додати задачу)_
- [ ] _(додати задачу)_

### Технічний борг / Рефакторинг
- [ ] _(додати задачу)_

---

## Журнал змін

| Дата | Коміт/Етап | Опис | Тести |
|------|-----------|------|-------|
| — | Stage 1 (45a3a4f) | Project Bootstrap: Wails scaffold, домен, SQLite, demo-режим, Canvas, autosave | go test ✅ |
| — | Stage 2 (03f7efc) | ODE solver RK45, oscillator mode, 6 серій, аналітичні тести | go test ✅ |
| — | Stage 3 (4d8e708) | JSON typed envelope format:1, режим у snapshot, DB schema v2, compatibility tests | go test ✅ |
| — | Stage 4 (14f28d2) | Нормалізація Chart (Reference/Automatic), легенда, масштаб/видимість за ID | go test ✅ |
| — | Stage 5 (5cd87d5) | Регульований розділювач, підготовча структура PreparationInputs | go test ✅ |
| — | Stage 6A (c598e2b) | Редактор підготовчих даних (7 полів), UI для preparation mode | go test ✅ |
| — | Stage 6B (8d20348) | Ballistics solver (OZVB-3A/Slukhotsky), 6 балістичних серій | go test ✅ |
| — | Stage 7 (8b17c57) | Імпорт CSV зовнішніх результатів, DB schema v3, SHA-256, незмінна таблиця | go test ✅ |
| — | IBSolver-Core (a3867f2) | Рефакторинг ядра, exchange v2, external results API | go test ✅ |
| 2026-11-22 | FRONT-01 (a7afc0a) | Фронтенд: main.js, style.css — фінальний UI для всіх трьох режимів | go test ✅, npm test ✅ |
| 2026-11-23 | — | Додано GEMINI.md, ROADMAP.md, .agents/skills/code-review-and-commit/SKILL.md | go test ✅ |
| 2026-11-23 | — | Об’єднано правила агента в AGENTS.md, узгоджено порядок перевірок і оновлено посилання в навичці перевірки коду | go test ./... ✅, npm test ✅ |
| 2026-11-23 | — | Зафіксовано висоту панелі курсора. Часову вісь і показник часу під курсором переведено в мілісекунди | go test ./... ✅, npm test ✅, Wails build ✅, наведення у WebView2 ✅ |
