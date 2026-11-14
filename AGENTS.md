# Lean Development Workflow

- Work with one agent by default. Do not spawn subagents without an explicit request or demonstrated need; honor task-specific restrictions.
- Read only relevant files, functions and dependencies. Avoid whole-project scans without a reason.
- Make the smallest sufficient change. Avoid unrelated refactoring, speculative interfaces and unnecessary abstractions.
- Test according to risk: targeted tests first; full suites, native builds and integration checks when necessary. Reuse established evidence unless something changed.
- Treat `D:\go\ibsolver-refs` as read-only. Read only specific relevant sources; never copy originals into Git.
- Future numerical work requires physical correctness, explicit units, numerical accuracy and reproducibility. Resource savings never justify skipping necessary checks.
- Justify new dependencies. Do not add frameworks, large reports, plans or task-size classifications without a need.
- Code, identifiers and comments use English. UI and user communication use Ukrainian.
- Keep domain independent of Wails and SQLite; keep the Wails bridge thin. No solver or historical formula migration in Stage 1.
- After work, briefly report changes, verification and remaining limitations.
- Do not commit or push without a direct request. Preserve existing work and uncommitted changes.
