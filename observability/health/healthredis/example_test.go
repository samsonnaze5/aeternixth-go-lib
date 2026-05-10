package healthredis_test

import (
	"errors"
	"fmt"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/health/healthredis"
)

func ExampleNewPinger_nilClient() {
	_, err := healthredis.NewPinger(nil)
	fmt.Println(errors.Is(err, healthredis.ErrNilClient))
	// Output: true
}
