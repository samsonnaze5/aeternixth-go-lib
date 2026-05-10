package healthkafka_test

import (
	"errors"
	"fmt"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/health/healthkafka"
)

func ExampleNewMetadataPinger_emptyBrokers() {
	_, err := healthkafka.NewMetadataPinger(nil, []string{"t1"}, nil)
	fmt.Println(errors.Is(err, healthkafka.ErrEmptyBrokers))
	// Output: true
}
