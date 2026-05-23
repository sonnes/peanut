# Peanut

Peanut is a simple, flexible, and efficient task runner. See ./README.md for details.

## TDD (strict)

Write tests BEFORE implementation. Run failing test, write minimum code to pass, confirm green. No exceptions.

- Use `github.com/stretchr/testify` (`require` for fatal, `assert` for non-fatal). Table-driven tests preferred.

## Go Style

Write vertical, readable code. Favor more lines over longer lines:

- Keep packages small and focused — no circular dependencies
- One argument per line for long function calls (trailing comma on last arg)
- Intermediate variables over deeply nested expressions
- Named booleans for long conditionals
- Early returns to keep logic flat
- Vertical struct literals (one field per line)

## Documentation

- GoDoc on all public functions/types. Use `[pkg.Type]` annotations. Use `doc.go` for package-level docs.
- **Concept docs** (`docs/concepts/`): one file per concept with YAML frontmatter (`title`, `description`, `package`). Create for new abstractions, update when APIs/behavior change. Include Go examples matching actual API. Update `docs/concepts/README.md` index.
