# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Behavioral Guidelines

These guidelines reduce common LLM coding mistakes. They bias toward caution over speed — for trivial tasks, use judgment.

### Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

**Before sending: pass 3 filters.** If any fails, revise *before* the user pushes back — not after.

- **Over-engineering** — went beyond what was asked → cut what wasn't requested
- **AI smell** — looks generated (commentary on every line, abstractions used once, error handling for cases that can't happen) → rewrite
- **Generic** — vague catch-all (one error handler swallows everything, function tries to do many things) → make specific to the actual case

### Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

**If the first attempt is structurally wrong, rewrite — don't patch.** Refining a bad approach line by line costs more than starting over. Signal: you find yourself fixing the same architectural choice in 3+ places, or each "fix" introduces a new problem. When in doubt, propose the rewrite to the user before silently iterating.

---

## Project Overview

Go utility library (`github.com/samsonnaze5/aeternixth-go-lib`) — a collection of 16 reusable packages for backend applications. Requires Go 1.25+.

## Common Commands

```bash
go build ./...        # Build all packages
go test ./...         # Run all tests
go test ./gmail/...   # Run tests for a single package
task format           # Format code (gofmt + goimports via Taskfile)
```

## Architecture

Flat package structure — each directory is an independent utility package with no cross-package dependencies (except `response` depends on `errors`, `fiber` depends on `middleware`/`errors`/`validator`, and `middleware` depends on `errors`/`jwt`).

**Critical: directory names differ from package names for several packages:**

| Directory | Package Name | Import Path |
|-----------|-------------|-------------|
| `aws/` | `thirdpartyaws` | `.../aws` |
| `jwt/` | `jwtutil` | `.../jwt` |
| `defaults/` | `defaultutil` | `.../defaults` |
| `password/` | `passwordutil` | `.../password` |
| `fiber/` | `fiberutil` | `.../fiber` |

All other packages match their directory names.

## Key Design Patterns

- **Generics** used in `pagination.Response[T]`, `jwtutil.JWTService[T jwt.Claims]`, `fiberutil.GetRequestBody[T]`, and `fiberutil.GetQueryParams[T]`
- **Dependency Inversion** in `gmail`: callers depend on `EmailSender` interface, not concrete `GmailSender`
- **Immutable value objects** in `gmail`: `Message` uses unexported fields + getters to enforce construction through `NewMessage()` validation
- **Sentinel errors** throughout (errors.Is()-compatible): `gmail.ErrEmptyRecipient`, `jwtutil.ErrExpiredToken`, etc.
- **`response` + `errors` pairing**: `response` package imports `errors` as `apperrors` and wraps `AppError` into Fiber HTTP responses
- **Middleware chain**: `middleware` package provides JWT auth, error handling, panic recovery, and 404 handling for Fiber apps; `fiber` (fiberutil) provides typed helpers to extract params, body, query, and user info from `fiber.Ctx`

## Code Style

- All exported functions and types have godoc comments with examples
- `errors` package uses string constants for error codes (`ErrCodeNotFound = "NOT_FOUND"`) mapped to HTTP status codes
- `null` package follows a consistent pattern: `ToNull{Type}(pointer) → sql.Null{Type}` and `ToNull{Type}Pointer(sqlNull) → *type`
- `decimal` package is an alias to `github.com/shopspring/decimal.Decimal` offering zero-allocation string parsing and exact math.
