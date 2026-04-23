# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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
