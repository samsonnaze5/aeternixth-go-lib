package otelgorm_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	otelapi "go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/otel/otelgorm"
)

// execDriver is a database/sql driver whose connections accept any Exec
// and nothing else — enough to drive GORM's Raw callback chain without a
// real database.
type execDriver struct{}

func (execDriver) Open(string) (driver.Conn, error) { return execConn{}, nil }

type execConn struct{}

func (execConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not supported") }
func (execConn) Close() error                        { return nil }
func (execConn) Begin() (driver.Tx, error)           { return nil, errors.New("not supported") }
func (execConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}

func init() { sql.Register("otelgorm-exec", execDriver{}) }

func newTracedDB(t *testing.T) (*gorm.DB, *tracetest.InMemoryExporter) {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	prev := otelapi.GetTracerProvider()
	otelapi.SetTracerProvider(tp)
	t.Cleanup(func() {
		otelapi.SetTracerProvider(prev)
		_ = tp.Shutdown(context.Background())
	})

	sqlDB, err := sql.Open("otelgorm-exec", "")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{ConnPool: sqlDB})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	if err := otelgorm.Register(db); err != nil {
		t.Fatalf("Register: %v", err)
	}
	return db, exp
}

func TestRegister_NilDB(t *testing.T) {
	if err := otelgorm.Register(nil); !errors.Is(err, otelgorm.ErrNilDB) {
		t.Errorf("err: want ErrNilDB, got %v", err)
	}
}

func TestRegister_QueryWithoutParentSpan_EmitsNothing(t *testing.T) {
	db, exp := newTracedDB(t)

	if err := db.WithContext(context.Background()).Exec("UPDATE t SET a = 1").Error; err != nil {
		t.Fatalf("Exec: %v", err)
	}

	if got := exp.GetSpans(); len(got) != 0 {
		t.Errorf("spans: want none for a query outside any span, got %d (%s)", len(got), got[0].Name)
	}
}

func TestRegister_QueryUnderParentSpan_EmitsChildWithoutBindValues(t *testing.T) {
	db, exp := newTracedDB(t)

	ctx, parent := otelapi.Tracer("test").Start(context.Background(), "handler")
	if err := db.WithContext(ctx).Exec("UPDATE t SET secret = ?", "s3cr3t-value").Error; err != nil {
		t.Fatalf("Exec: %v", err)
	}
	parent.End()

	spans := exp.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("spans: want db child + parent, got %d", len(spans))
	}
	child := spans[0]
	if child.Parent.SpanID() != parent.SpanContext().SpanID() {
		t.Errorf("db span parent: want %s, got %s", parent.SpanContext().SpanID(), child.Parent.SpanID())
	}
	for _, kv := range child.Attributes {
		if strings.Contains(kv.Value.Emit(), "s3cr3t-value") {
			t.Errorf("attribute %s leaks bind value: %q", kv.Key, kv.Value.Emit())
		}
	}
}
