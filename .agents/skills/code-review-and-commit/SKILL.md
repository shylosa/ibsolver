---
name: code-review-and-commit
description: >
  Застосовувати перед кожним комітом або для перевірки поточних змін.
  Скіл перевіряє git diff, шукає потенційні баги та проблеми безпеки,
  запускає тести, оновлює ROADMAP.md і пропонує повідомлення коміту
  у форматі Conventional Commits.
---

# Навичка: code-review-and-commit

Використовувати:
- перед виконанням `git commit`
- коли потрібно перевірити якість поточних змін
- після завершення будь-якого Implementation Plan

---

## Крок 1 — Огляд поточного стану

```powershell
Set-Location d:\go\ibsolver
git status
git diff --stat
```

Зафіксувати: які файли змінено, додано, видалено. Якщо змін немає — повідомити і зупинитися.

---

## Крок 2 — Детальний огляд diff

```powershell
git diff
# або для staged:
git diff --cached
```

Переглянути весь diff і перевірити:

### Потенційні баги
- Нові `float64` без перевірки `math.IsNaN` / `math.IsInf` перед збереженням або використанням у розрахунку
- Відсутній `defer tx.Rollback()` у нових транзакціях SQLite
- Відсутній `defer s.mu.Unlock()` після нового `s.mu.Lock()`
- Нові bridge-методи в `app.go`, що не викликають `a.ready()`
- Нові Go-методи з `error` у сигнатурі, де помилка ігнорується (`_`)

### Проблеми безпеки
- Динамічний SQL без параметризації (конкатенація рядків у SQL-запити)
- `io.ReadAll` без `io.LimitReader` на нових потоках читання (DoS)
- Нові файлові операції без перевірки шляху (`filepath.Clean`, `os.UserConfigDir`)

### Debug-артефакти (видалити перед комітом)
- `fmt.Println`, `console.log`, `console.error` у production-коді Go та JS
- Тимчасові `// TODO: remove` або `// DEBUG` коментарі
- Захардкоджені шляхи або тестові значення у production-коді

### Порушення конвенцій (`AGENTS.md`)
- Рядки помилок у Go: UI-повідомлення → українська, технічні → англійська
- Нові залежності в пакеті `ode` від `domain`/`storage`/`application`
- Нові залежності в `domain` від Wails або SQLite
- Ручні правки у `frontend/wailsjs/` (автогенерований код)

---

## Крок 3 — Запуск тестів

```powershell
# Go тести (всі пакети)
Set-Location d:\go\ibsolver
go test ./...

# Frontend тести
Set-Location d:\go\ibsolver\frontend
npm test
```

- Якщо тести **провалились** — зупинитися, повідомити про конкретну помилку, не продовжувати.
- Якщо тести **пройшли** — зафіксувати результат і перейти до наступного кроку.

---

## Крок 4 — Оновлення `ROADMAP.md`

Відкрити [ROADMAP.md](file:///d:/go/ibsolver/ROADMAP.md) і додати рядок у розділ **"Журнал змін"**:

```markdown
| YYYY-MM-DD | — | Короткий опис змін (що зроблено, що перевірено) | go test ✅, npm test ✅ |
```

Якщо зміни стосуються задачі зі списку "В роботі / Наступні кроки" — позначити її як `[x]`.

---

## Крок 5 — Формування повідомлення коміту

Запропонувати повідомлення у форматі **Conventional Commits**:

```
<type>(<scope>): <короткий опис англійською, imperative mood>

[необов'язкове тіло: що зроблено і навіщо, якщо неочевидно]
```

**Допустимі типи:**

| Тип | Коли використовувати |
|-----|---------------------|
| `feat` | нова функціональність |
| `fix` | виправлення бага |
| `refactor` | рефакторинг без нової функціональності і без виправлення бага |
| `test` | додавання або виправлення тестів |
| `docs` | зміни лише у документації |
| `chore` | налаштування інструментів, залежностей, CI |
| `perf` | покращення продуктивності |

**Scope** — назва пакету або підсистеми: `domain`, `ode`, `storage`, `application`, `chartdata`, `frontend`, `bridge`, `exchange`.

**Приклади:**
```
feat(chartdata): add ballistics solver with 6 time series
fix(storage): add LimitReader to external CSV import (DoS guard)
refactor(domain): split numerical types into separate file
docs: consolidate agent instructions in AGENTS.md
test(ode): add step rejection and cancellation edge cases
```

Показати повідомлення користувачу і **чекати явного підтвердження** перед виконанням `git commit`.

---

## Крок 6 — Коміт (лише після підтвердження)

```powershell
Set-Location d:\go\ibsolver
git add -A
git commit -m "<підтверджене повідомлення>"
```

> **Не виконувати `git push` без явного запиту.**
