package itestkit

import (
	"context"
	"fmt"

	"github.com/samsonnaze5/aeternixth-go-lib/itestkit/internal/stringutil"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
	tcnetwork "github.com/testcontainers/testcontainers-go/network"
)

const defaultKafkaImage = "confluentinc/confluent-local:7.7.0"

// startKafka launches one Kafka (KRaft mode) container per entry in
// instances. Topic creation is handled later by createKafkaTopics so a
// failed topic does not orphan an otherwise-healthy broker.
func startKafka(
	ctx context.Context,
	opts StackOptions,
	net *testcontainers.DockerNetwork,
) (map[string]*KafkaResource, error) {
	if len(opts.Kafka) == 0 {
		return nil, nil
	}
	out := make(map[string]*KafkaResource, len(opts.Kafka))
	for name, cfg := range opts.Kafka {
		res, err := startOneKafka(ctx, name, cfg, opts, net)
		if err != nil {
			terminateKafka(out, opts)
			return nil, fmt.Errorf("start kafka[%s]: %w", name, err)
		}
		out[name] = res
	}
	return out, nil
}

func startOneKafka(
	ctx context.Context,
	name string,
	cfg KafkaOptions,
	stackOpts StackOptions,
	net *testcontainers.DockerNetwork,
) (*KafkaResource, error) {
	image := cfg.Image
	if image == "" {
		image = defaultKafkaImage
	}
	clusterID := cfg.ClusterID
	if clusterID == "" {
		clusterID = fmt.Sprintf(
			"%s-%s-cluster",
			stringutil.NormalizeIdentifier(stackOpts.ProjectName),
			stringutil.NormalizeIdentifier(name),
		)
	}

	containerOpts := []testcontainers.ContainerCustomizer{
		kafka.WithClusterID(clusterID),
	}
	if net != nil {
		containerOpts = append(containerOpts, tcnetwork.WithNetwork([]string{"kafka-" + name}, net))
	}
	if len(cfg.ExtraEnv) > 0 {
		containerOpts = append(containerOpts, testcontainers.WithEnv(cfg.ExtraEnv))
	}

	container, err := kafka.Run(ctx, image, containerOpts...)
	if err != nil {
		return nil, err
	}

	brokers, err := container.Brokers(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, fmt.Errorf("brokers: %w", err)
	}

	return &KafkaResource{
		Name:      name,
		Brokers:   brokers,
		Container: container,
	}, nil
}

func terminateKafka(res map[string]*KafkaResource, opts StackOptions) {
	if opts.Debug.KeepContainersOnFailure {
		return
	}
	for _, r := range res {
		if c, ok := r.Container.(testcontainers.Container); ok && c != nil {
			_ = testcontainers.TerminateContainer(c)
		}
	}
}
