package healthgorm_test

import (
	"errors"
	"fmt"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/health/healthgorm"
)

func ExampleNewPinger_nilDB() {
	_, err := healthgorm.NewPinger(nil)
	fmt.Println(errors.Is(err, healthgorm.ErrNilDB))
	// Output: true
}
