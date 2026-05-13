# itestkit — คู่มือใช้งานสำหรับโปรเจกต์ปลายทาง

> เอกสารนี้เขียนสำหรับ Go developer (รวม intern) ที่จะเขียน integration test ในโปรเจกต์ที่ import `github.com/samsonnaze5/aeternixth-go-lib/itestkit`.
>
> **เป้าหมาย:** ทุก project ทำตาม pattern เดียวกัน — อ่าน test ที่ repo ไหนก็คุ้นเคยเหมือนกันหมด.

ศัพท์หลัก (Stack / Instance / Reset) อยู่ใน [`CONTEXT.md`](../CONTEXT.md).
การตัดสินใจเรื่อง migration mirror อยู่ใน [ADR-0003](../docs/adr/0003-itestkit-test-migration-mirror.md).

---

## 1. itestkit ทำอะไรให้คุณ

ทำให้:
- เปิด container จริงของ Postgres / ClickHouse / Redis / Kafka / MockServer / WireMock / LocalStack ผ่าน Testcontainers Go
- รัน migrations + seeds (เฉพาะ Postgres / ClickHouse)
- สร้าง Kafka topics
- โหลด WireMock mappings, MockServer expectations, LocalStack init scripts
- คืน DSN / host / port / brokers / endpoint
- ผูก cleanup ให้ `t.Cleanup` อัตโนมัติ

ไม่ทำให้:
- ไม่เขียน schema/migration ให้
- ไม่ตัดสินใจ business logic ของ test
- ไม่จัดการ connection pool ของ app (เขียนใน `helpers.go` ของ project ปลายทาง)
- ไม่เข้าใจ goose / golang-migrate annotations — รับเฉพาะ plain SQL

---

## 2. Prerequisites — ของที่ต้องเตรียมก่อนเริ่ม

| ของ                                                                          | ตรวจยังไง                 |
|------------------------------------------------------------------------------|---------------------------|
| Docker runtime (Docker Desktop / Colima / OrbStack / Podman / Docker Engine) | `docker ps` ไม่ error     |
| Go 1.25+                                                                     | `go version`              |
| Task ([taskfile.dev](https://taskfile.dev))                                  | `task --version`          |
| RAM อย่างน้อย 4 GB ว่าง                                                      | สำหรับ stack ที่มี Kafka  |
| Network ออก internet ได้                                                     | pull image ครั้งแรก ~1 GB |
| `migrations/` (ถ้าใช้ Postgres/ClickHouse)                                   | มีไฟล์ migration อยู่แล้ว |

> CI ที่เป็น GitHub Actions / GitLab CI กับ Docker-in-Docker ทำงานได้เลย ไม่ต้องตั้ง env var เพิ่ม

---

## 3. Folder layout มาตรฐาน (ต้องเป็นแบบนี้ทุก project)

```
<repo>/
├── migrations/                                   # prod migrations (goose / migrate / ...)
│   ├── 0001_create_users.sql
│   └── 0002_add_user_status_idx.sql
├── tests/
│   └── integration/                              # package integration
│       ├── bootstrap.go                          # TestMain — เปิด/ปิด Stack
│       ├── helpers.go                            # PostgresPool / RedisClient / KafkaWriter / ...
│       ├── reset.go                              # Reset(t)
│       ├── *_test.go                             # test จริง
│       └── testdata/
│           ├── postgres/
│           │   └── main/                         # <instance>/
│           │       ├── migrations/               # plain SQL mirror (generated)
│           │       │   └── 0001_create_users.sql
│           │       └── seeds/
│           │           └── 0001_seed_admin.sql
│           ├── clickhouse/
│           │   └── events/
│           │       ├── migrations/
│           │       └── seeds/
│           ├── wiremock/                         # ถ้าใช้ WireMock
│           │   └── exchange_rate/
│           │       ├── mappings/
│           │       └── __files/
│           └── localstack/                       # ถ้าใช้ AWS
│               └── aws/
│                   └── init/
│                       └── 001_create_bucket.sh
└── Taskfile.yml
```

### กฎของ path
- `<service>/<instance>/<purpose>/` — เสมอ (แม้ instance เดียวก็ต้องมีโฟลเดอร์ instance)
- migration / seed: `0001_snake_case.sql` (4-digit zero-padded + snake_case)
- LocalStack init: `001_snake_case.sh` (3-digit zero-padded)
- instance name: regex `^[a-z][a-z0-9_-]*$` (lowercase, ขึ้นด้วยตัวอักษร)

### Default instance name ถ้าไม่มีเหตุผลอื่น
| Service | Default name |
|---|---|
| Postgres | `main` |
| ClickHouse | `events` |
| Redis | `cache` |
| Kafka | `main` |
| MockServer / WireMock | ชื่อ external service เช่น `exchange_rate` |
| LocalStack | `aws` |

ถ้า project มีหลาย instance หรือมี **DDD bounded context** ชัด ๆ → ตั้งตามชื่อนั้น เช่น `orders`, `ledger`, `users-readmodel`.

---

## 4. Step 1 — เพิ่ม dependency

```bash
go get github.com/samsonnaze5/aeternixth-go-lib
```

driver ที่ helpers ต้องใช้ (ตัวอย่าง stack ครบชุด):

```bash
go get github.com/jackc/pgx/v5/pgxpool
go get github.com/redis/go-redis/v9
go get github.com/segmentio/kafka-go
go get github.com/ClickHouse/clickhouse-go/v2
```

---

## 5. Step 2 — Mirror migrations (เฉพาะ project ที่ใช้ goose)

> ข้าม step นี้ ถ้า project เขียน migration เป็น plain SQL อยู่แล้ว — ใช้ `migrations/` ตรง ๆ ได้เลย (กฎ "ห้ามชี้ที่ migrations/ โดยตรง" ใช้กับ goose project; plain SQL ถือว่า mirror = ตัวเองอยู่แล้ว ให้ใส่ symlink หรือเทียบเท่า)

itestkit รัน `.sql` ตรง ๆ ผ่าน `db.Exec`. ถ้าไฟล์มี `-- +goose Up` กับ `-- +goose Down` ในไฟล์เดียวกัน → จะรันทั้งคู่ → migration ถูกย้อนทันที.

วิธีแก้: สร้าง plain-SQL mirror ใต้ `tests/integration/testdata/postgres/<instance>/migrations/` ที่ตัด annotations ออก.

### 5.1 Task target สำหรับ sync

```yaml
# Taskfile.yml
version: "3"

tasks:
  itest:sync-migrations:
    desc: Generate plain-SQL test mirror from prod goose migrations
    cmds:
      - |
        DST=tests/integration/testdata/postgres/main/migrations
        rm -rf "$DST"
        mkdir -p "$DST"
        for f in migrations/*.sql; do
          base=$(basename "$f")
          awk '
            /-- \+goose Down/ { exit }
            !/-- \+goose Up/  { print }
          ' "$f" > "$DST/$base"
        done
        echo "✓ wrote $(ls "$DST" | wc -l | xargs) files into $DST"
```

> หลายๆ instance หรือ ClickHouse → copy block นี้ ปรับ `DST` กับ source path ตามไป.

### 5.2 Workflow

```text
dev เขียน migration goose ใหม่ใน migrations/0042_xxx.sql
  ↓
task itest:sync-migrations        ← mirror อัปเดต
  ↓
git add migrations/ tests/integration/testdata/
git commit
```

### 5.3 CI check (กันลืม sync)

```yaml
# .github/workflows/test.yml
- name: Ensure migration mirror is in sync
  run: |
    task itest:sync-migrations
    git diff --exit-code tests/integration/testdata/postgres/
```

ถ้า dev ลืม sync → CI fail พร้อม diff ของไฟล์ที่ขาด

> **ข้อจำกัด:** ถ้าไฟล์ goose ใช้ `-- +goose StatementBegin` / `-- +goose StatementEnd` (block สำหรับ stored procedures), awk แบบง่าย ๆ จัดการไม่ได้ — ต้องเขียน mirror ไฟล์นั้นมือ. กรณีหายาก แต่ดักไว้.

---

## 6. Step 3 — `bootstrap.go`

```go
// tests/integration/bootstrap.go
package integration

import (
    "context"
    "log"
    "os"
    "testing"

    "github.com/samsonnaze5/aeternixth-go-lib/itestkit"
)

// Stack ที่ทุก *_test.go ในโฟลเดอร์นี้แชร์กัน.
var Stack *itestkit.Stack

func TestMain(m *testing.M) {
    ctx := context.Background()
    var err error
    Stack, err = itestkit.StartStack(ctx, nil, itestkit.StackOptions{
        ProjectName: "client-portal-api", // ใช้ชื่อ microservice / repo จริง
        Postgres: map[string]itestkit.PostgresOptions{
            "main": {
                MigrationPaths:  []string{"testdata/postgres/main/migrations"},
                SeedPaths:       []string{"testdata/postgres/main/seeds"},
                ApplyMigrations: true,
                ApplySeeds:      true,
                StrictPath:      true,
            },
        },
        Redis: map[string]itestkit.RedisOptions{
            "cache": {FlushBeforeTest: true},
        },
        Kafka: map[string]itestkit.KafkaOptions{
            "main": {
                CreateTopics: true,
                Topics: []itestkit.KafkaTopic{
                    {Name: "user.created", Partitions: 3},
                },
            },
        },
    })
    if err != nil {
        log.Fatalf("itestkit: %v", err)
    }
    code := m.Run()
    _ = Stack.Cleanup(ctx)
    os.Exit(code)
}
```

หมายเหตุ:
- ส่ง `nil` ใน argument `t` ของ `StartStack` เพราะ `TestMain` ยังไม่มี `*testing.T`. cleanup จะถูกเรียกเองตอนจบ `m.Run()` ด้วย `Stack.Cleanup(ctx)`.
- `ProjectName` ปรากฏใน Docker network name, default DB name, Kafka cluster ID → ใส่ชื่อ microservice/repo จะ debug ง่าย.
- `StrictPath: true` ทำให้ path ที่สะกดผิดถูกจับทันที (เปิดเสมอ).

---

## 7. Step 4 — `helpers.go`

```go
// tests/integration/helpers.go
package integration

import (
    "context"
    "testing"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/redis/go-redis/v9"
    "github.com/segmentio/kafka-go"
)

func PostgresPool(t *testing.T, name string) *pgxpool.Pool {
    t.Helper()
    pool, err := pgxpool.New(context.Background(), Stack.Config.PostgresDSN(name))
    if err != nil {
        t.Fatalf("postgres pool[%s]: %v", name, err)
    }
    t.Cleanup(pool.Close)
    return pool
}

func RedisClient(t *testing.T, name string) *redis.Client {
    t.Helper()
    opts, err := redis.ParseURL(Stack.Config.RedisURL(name))
    if err != nil {
        t.Fatalf("redis url[%s]: %v", name, err)
    }
    c := redis.NewClient(opts)
    t.Cleanup(func() { _ = c.Close() })
    return c
}

func KafkaWriter(t *testing.T, name, topic string) *kafka.Writer {
    t.Helper()
    w := &kafka.Writer{
        Addr:     kafka.TCP(Stack.Config.KafkaBrokerList(name)...),
        Topic:    topic,
        Balancer: &kafka.LeastBytes{},
    }
    t.Cleanup(func() { _ = w.Close() })
    return w
}

func KafkaReader(t *testing.T, name, topic string) *kafka.Reader {
    t.Helper()
    r := kafka.NewReader(kafka.ReaderConfig{
        Brokers: Stack.Config.KafkaBrokerList(name),
        Topic:   topic,
        GroupID: t.Name(), // unique group ต่อ test → message ไม่ปนกัน
    })
    t.Cleanup(func() { _ = r.Close() })
    return r
}
```

กฎของ helper:
- รับ `*testing.T` และ instance name เสมอ — ไม่มีรูป shortcut
- ใช้ `t.Helper()` เพื่อให้ test fail บอกบรรทัดของ test จริง ไม่ใช่บรรทัดของ helper
- ผูก close ผ่าน `t.Cleanup` — ไม่ต้อง defer ใน test
- Kafka reader → `GroupID: t.Name()` เป็นกฎ; offset ของ test ไม่ปนกัน

---

## 8. Step 5 — `reset.go`

```go
// tests/integration/reset.go
package integration

import (
    "context"
    "database/sql"
    "testing"

    _ "github.com/jackc/pgx/v5/stdlib"

    "github.com/samsonnaze5/aeternixth-go-lib/itestkit"
)

var migrationTrackerTables = map[string]struct{}{
    "schema_migrations":        {},
    "goose_db_version":         {},
    "flyway_schema_history":    {},
    "dbmate_schema_migrations": {},
}

// Reset เคลียร์สถานะของ Stack ทั้งหมดให้พร้อมสำหรับ test ตัวถัดไป.
// เรียกที่ "บรรทัดแรก" ของทุก test.
func Reset(t *testing.T) {
    t.Helper()
    ctx := context.Background()

    for name, r := range Stack.Postgres {
        tables := discoverPostgresTables(t, r.DSN, name)
        if len(tables) == 0 {
            continue
        }
        if err := itestkit.TruncatePostgres(ctx, r.DSN, tables...); err != nil {
            t.Fatalf("reset postgres[%s]: %v", name, err)
        }
    }

    for name, r := range Stack.Redis {
        if err := itestkit.RedisFlushAll(ctx, r.URL); err != nil {
            t.Fatalf("reset redis[%s]: %v", name, err)
        }
    }

    // MockServer / WireMock — ถ้ามี ให้เรียก reset endpoint แล้ว re-apply expectations
    // (เขียนใน helpers.go เฉพาะ project ที่ใช้)
}

func discoverPostgresTables(t *testing.T, dsn, name string) []string {
    t.Helper()
    db, err := sql.Open("pgx", dsn)
    if err != nil {
        t.Fatalf("open postgres[%s]: %v", name, err)
    }
    defer db.Close()

    rows, err := db.QueryContext(context.Background(),
        `SELECT tablename FROM pg_tables WHERE schemaname = 'public'`)
    if err != nil {
        t.Fatalf("discover tables[%s]: %v", name, err)
    }
    defer rows.Close()

    var out []string
    for rows.Next() {
        var tbl string
        if err := rows.Scan(&tbl); err != nil {
            t.Fatalf("scan: %v", err)
        }
        if _, skip := migrationTrackerTables[tbl]; skip {
            continue
        }
        out = append(out, tbl)
    }
    return out
}
```

กฎของ Reset:
- เรียก **บรรทัดแรก** ของทุก test (`func TestXxx(t *testing.T) { Reset(t); ... }`)
- TRUNCATE auto-discover ทุก table ใน `public` schema ยกเว้น migration trackers — ไม่ต้อง maintain list
- Kafka **ไม่ถูก reset** — test ใช้ `GroupID: t.Name()` แทนเพื่อกัน offset ปน

---

## 9. Step 6 — เขียน test ตัวแรก

```go
// tests/integration/user_test.go
package integration

import (
    "context"
    "testing"
)

func TestCreateUser(t *testing.T) {
    Reset(t)
    ctx := context.Background()
    pool := PostgresPool(t, "main")

    _, err := pool.Exec(ctx,
        `INSERT INTO users (id, email) VALUES ($1, $2)`,
        "u-001", "alice@example.com")
    if err != nil {
        t.Fatalf("insert: %v", err)
    }

    var got string
    err = pool.QueryRow(ctx,
        `SELECT email FROM users WHERE id = $1`, "u-001").Scan(&got)
    if err != nil {
        t.Fatalf("query: %v", err)
    }
    if got != "alice@example.com" {
        t.Fatalf("got %q, want alice@example.com", got)
    }
}
```

---

## 10. Step 7 — Task target สำหรับรัน test

```yaml
# Taskfile.yml — เพิ่มเข้าไป
tasks:
  test:integration:
    desc: Run integration tests
    cmds:
      - go test -count=1 -timeout=10m ./tests/integration/...
```

รัน:
```bash
task test:integration
# หรือสั่งตรง
go test -count=1 -timeout=10m ./tests/integration/...
```

- `-count=1` ปิด test caching — integration test ต้องรันใหม่จริงทุกครั้ง
- `-timeout=10m` ครอบคลุม container boot + ทุก test (ปรับตาม project)

---

## 11. CI integration (ตัวอย่าง GitHub Actions)

```yaml
# .github/workflows/integration.yml
name: integration
on: [pull_request]
jobs:
  integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - uses: arduino/setup-task@v2
      - name: Sync migration mirror & verify
        run: |
          task itest:sync-migrations
          git diff --exit-code tests/integration/testdata/postgres/
      - name: Run integration tests
        run: task test:integration
```

---

## 12. Debug — เมื่อ test fail

เปิด debug flags ชั่วคราว (เฉพาะ local):

```go
Debug: itestkit.DebugOptions{
    KeepContainersOnFailure: true,  // ไม่ kill container ตอน start fail
    PrintConnectionInfo:     true,  // log DSN / host / port หลัง start
    PrintContainerLogs:      true,  // stream log ของ container
},
```

แล้วใช้:
```bash
go test -count=1 -timeout=10m -v ./tests/integration/...
docker ps                           # หา container ที่ค้าง
docker logs <id>                    # อ่าน log
docker exec -it <id> psql -U test -d <db>  # ต่อตรงเข้า Postgres ส่อง state
```

> Debug flags **ห้ามเปิดใน CI / main branch** — เปิดเฉพาะตอน debug

---

## 13. กฎที่ห้ามแหก (Rules)

1. **ห้าม `t.Parallel()`** ใน `tests/integration/*_test.go` — Stack แชร์, Reset ไม่ทนต่อ race
2. **ห้ามชี้ `MigrationPaths` ไปที่ `migrations/`** โดยตรง (สำหรับ goose project) — ต้องผ่าน mirror ใน `testdata/`
3. **ห้ามเปิด connection เอง** (`sql.Open`, `pgxpool.New`, `redis.NewClient`) ใน test — ต้องเรียก helper (`PostgresPool(t, …)` ฯลฯ)
4. **ห้ามข้าม `Reset(t)`** ที่ต้นทุก test
5. **ห้ามพึ่งข้อมูล** ที่ test อื่นทิ้งไว้ — test ต้อง self-contained
6. **ห้าม mock DB / Redis / Kafka** ใน integration test — ถ้าจะ mock ให้ไปอยู่ unit test
7. **ห้าม hard-code** DSN / port / host / broker — ใช้ `Stack.Config.*` เสมอ
8. **ห้ามแก้ไฟล์ใน `testdata/postgres/<instance>/migrations/`** โดยตรง — เป็น generated artifact จาก `task itest:sync-migrations`
9. **ห้ามใช้ instance name** ที่ไม่ตรง regex `^[a-z][a-z0-9_-]*$`
10. **ห้ามเปิด `Debug` flag** ใน CI / main branch

---

## 14. Best practices

- **ProjectName ใช้ชื่อ microservice / repo จริง** เช่น `"client-portal-api"`. ช่วยตอน `docker ps` มี stack หลายตัวจะแยกได้
- **Seed = ข้อมูลที่ทุก test ใช้ร่วมกัน** (เช่น lookup table, role definitions). ข้อมูลเฉพาะ test → insert ใน test body, ไม่ต้องใส่ seed
- **MockServer = default** สำหรับ HTTP stub. WireMock เฉพาะกรณี: ทีมมี JSON mapping catalog อยู่แล้ว / mock ซับซ้อน / share ข้าม project
- **Kafka topic ประกาศใน `StackOptions`** ตอน boot — ห้ามสร้าง topic runtime ใน test
- **Kafka consumer group = `t.Name()`** — กฎเดียวกันทุก project
- **เปิด `PrintConnectionInfo: true`** ตอน dev local → คัด DSN ไปเปิด DB tool ส่อง state
- **`StrictPath: true` เสมอ** ใน option ที่รับ path — ลืมไฟล์ → fail ทันที ไม่ผ่านแบบเงียบ ๆ
- **เขียน assertion เป็น behavior ไม่ใช่ side effect** — query state จาก DB แล้ว assert; อย่า hook function ภายในของ app
- **ใส่ `Reset(t)` ก่อน `PostgresPool(t, …)`** ใน test body — order matters (Reset เปิด/ปิด connection ของตัวเอง)

---

## 15. Reference

- API หลัก: [`options.go`](options.go) (`StackOptions`), [`result.go`](result.go) (`Stack`, `AppTestConfig`)
- ตัวอย่าง compile-only: [`examples_test.go`](examples_test.go)
- Helper สำเร็จรูป: [`cleanup.go`](cleanup.go) (`TruncatePostgres`, `DropPostgresPublicSchema`, `RedisFlushAll`, …)
- ศัพท์ Stack / Instance / Reset: [`CONTEXT.md`](../CONTEXT.md)
- การตัดสินใจ migration mirror: [ADR-0003](../docs/adr/0003-itestkit-test-migration-mirror.md)
