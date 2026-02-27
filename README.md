# aeternixth-go-lib

A collection of reusable Go utility packages for building backend applications. Designed to reduce boilerplate and provide consistent patterns across projects.

## Installation

```bash
go get github.com/samsonnaze5/aeternixth-go-lib
```

Requires **Go 1.25+**

## Packages

### `null` — SQL Null Type Converters

Convert between Go pointer types (`*string`, `*int64`, etc.) and `database/sql` Null types (`sql.NullString`, `sql.NullInt64`, etc.) for nullable database columns.

```go
import "github.com/samsonnaze5/aeternixth-go-lib/null"

// Pointer -> SQL Null
name := "Alice"
nullName := null.ToNullString(&name) // sql.NullString{String: "Alice", Valid: true}
nullName = null.ToNullString(nil)    // sql.NullString{Valid: false}

// SQL Null -> Pointer
ptr := null.ToNullStringPointer(nullName) // nil
```

**Supported types:** `String`, `Int16`, `Int32`, `Int64`, `Float64`, `Bool`, `Time`, `UUID`, `Date` (string ↔ time.Time)

---

### `errors` — Standardized Application Errors

Structured error type with error codes, messages, and HTTP status codes for consistent API error responses.

```go
import "github.com/samsonnaze5/aeternixth-go-lib/errors"

err := errors.NewNotFound("user not found")
err = errors.NewBadRequest("invalid email format")
err = errors.NewValidationError(map[string]string{
    "email": "is required",
    "age":   "must be at least 18",
})
```

**Built-in error constructors:** `NewBadRequest`, `NewUnauthorized`, `NewForbidden`, `NewNotFound`, `NewConflict`, `NewInternalServerError`, `NewValidationError`

---

### `response` — Fiber HTTP Response Helpers

Standardized JSON response functions for the [Fiber](https://gofiber.io/) web framework.

```go
import "github.com/samsonnaze5/aeternixth-go-lib/response"

// Success responses
response.Success(c, user)          // 200 OK
response.Created(c, newUser)       // 201 Created
response.NoContent(c)              // 204 No Content

// Error responses
response.BadRequest(c, "invalid input")
response.NotFound(c, "user not found")
response.Unauthorized(c, "token expired")
response.ValidationError(c, fieldErrors)
```

---

### `jwt` — Generic JWT Service

Type-safe JWT token generation and validation using Go generics with HMAC-SHA256 signing.

```go
import "github.com/samsonnaze5/aeternixth-go-lib/jwt"

type MyClaims struct {
    UserID string `json:"user_id"`
    jwt.RegisteredClaims
}

svc := jwtutil.NewJWTService("secret-key", func() *MyClaims {
    return &MyClaims{}
})

token, err := svc.GenerateToken(&MyClaims{
    UserID: "abc",
    RegisteredClaims: jwt.RegisteredClaims{
        ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
    },
})

claims, err := svc.ValidateToken(token)
fmt.Println(claims.UserID) // "abc"
```

---

### `password` — Bcrypt Password Hashing

Secure password hashing and verification using bcrypt.

```go
import "github.com/samsonnaze5/aeternixth-go-lib/password"

hash, err := passwordutil.HashPassword("my-password")
err = passwordutil.VerifyPassword(hash, "my-password") // nil (match)
err = passwordutil.VerifyPassword(hash, "wrong")       // error (mismatch)
```

---

### `validator` — Struct Validation

Thin wrapper around [go-playground/validator](https://github.com/go-playground/validator) with human-readable error formatting.

```go
import "github.com/samsonnaze5/aeternixth-go-lib/validator"

type Request struct {
    Email string `validate:"required,email"`
    Name  string `validate:"required,min=2"`
}

err := validator.Validate(req)
if err != nil {
    msg := validator.FormatValidationError(err)
    // "Email must be a valid email address; Name must be at least 2 characters"
}
```

---

### `pagination` — Paginated Responses

Generic pagination utilities for building paginated API responses.

```go
import "github.com/samsonnaze5/aeternixth-go-lib/pagination"

offset := pagination.CalculateOffset(page, limit) // SQL OFFSET
resp := pagination.NewResponse(users, page, limit, totalCount)
// resp.TotalPages, resp.PageIndex, resp.PageSize, resp.TotalItems
```

---

### `ratelimit` — In-Memory Rate Limiter

Key-based rate limiting with a fixed cooldown window, safe for concurrent use.

```go
import "github.com/samsonnaze5/aeternixth-go-lib/ratelimit"

limiter := ratelimit.NewLimiter(60 * time.Second)

allowed, retryAfter := limiter.Allow("user:123")
if !allowed {
    fmt.Printf("Rate limited. Retry after %v\n", retryAfter)
}

limiter.Reset("user:123") // clear cooldown
```

---

### `gmail` — Gmail API Email Sender

Send emails through the Gmail REST API (v1) using OAuth2 credentials. Follows the Dependency Inversion Principle with an `EmailSender` interface.

```go
import "github.com/samsonnaze5/aeternixth-go-lib/gmail"

sender, err := gmail.NewGmailSender(gmail.Config{
    ClientID:     os.Getenv("GMAIL_CLIENT_ID"),
    ClientSecret: os.Getenv("GMAIL_CLIENT_SECRET"),
    RefreshToken: os.Getenv("GMAIL_REFRESH_TOKEN"),
    SenderName:   "My App",
    SenderEmail:  "noreply@myapp.com",
})

msg, err := gmail.NewMessage("user@example.com", "Welcome!", "<h1>Hello</h1>")
err = sender.Send(context.Background(), msg)
```

---

### `aws` — AWS S3 File Upload

Upload base64-encoded files to AWS S3 with automatic MIME type detection.

```go
import "github.com/samsonnaze5/aeternixth-go-lib/aws"

url, err := thirdpartyaws.Upload(
    accessKey, secretKey,
    "my-bucket", "uploads/images",
    "data:image/jpeg;base64,/9j/4AAQ...",
)
// url: "https://my-bucket.s3.amazonaws.com/uploads/images/uuid.jpeg"
```

**Supported formats:** JPEG, PNG, GIF, BMP, WEBP, TIFF, PDF, TXT, ZIP, MP4

---

### `timeutil` — UTC Time Helpers

Convenience functions for working with time in UTC.

```go
import "github.com/samsonnaze5/aeternixth-go-lib/timeutil"

now := timeutil.Now()              // time.Now().UTC()
expires := timeutil.NowPlusHour(24) // 24 hours from now in UTC
```

---

### `defaults` — Default Value Helpers

Safe pointer dereferencing with fallback defaults.

```go
import "github.com/samsonnaze5/aeternixth-go-lib/defaults"

size := defaultutil.DefaultInt(pageSize, 20)       // 20 if pageSize is nil
sort := defaultutil.DefaultString(sortBy, "created_at") // "created_at" if sortBy is nil
```

---

### `logutil` — Debug Logging

Quick JSON serialization and printing for debugging payloads.

```go
import "github.com/samsonnaze5/aeternixth-go-lib/logutil"

p := &logutil.Payloader{}
p.Print(myStruct) // Output: PAYLOAD {"field":"value"}
```

## License

MIT
