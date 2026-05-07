package itestkit

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/segmentio/kafka-go"
)

// createAllKafkaTopics walks every Kafka instance and creates the configured
// topics. Existing topics are tolerated (the spec asks for "create" semantics,
// so already-present topics are not an error). The returned error is wrapped
// with the instance and topic name for easy debugging.
func createAllKafkaTopics(ctx context.Context, opts StackOptions, res map[string]*KafkaResource) error {
	for name, cfg := range opts.Kafka {
		if !cfg.CreateTopics || len(cfg.Topics) == 0 {
			continue
		}
		r, ok := res[name]
		if !ok {
			continue
		}
		if err := CreateKafkaTopics(ctx, r.Brokers, cfg.Topics); err != nil {
			return fmt.Errorf("create kafka[%s]: %w", name, err)
		}
	}
	return nil
}

// CreateKafkaTopics creates the listed topics on the given Kafka cluster.
// brokers must contain at least one host:port endpoint reachable from this
// process. Topics that already exist are silently skipped.
//
// Partitions and ReplicationFactor default to 1 when zero. Topic.Config is
// passed through verbatim as Kafka topic-level configuration.
func CreateKafkaTopics(ctx context.Context, brokers []string, topics []KafkaTopic) error {
	if len(brokers) == 0 {
		return errors.New("no brokers provided")
	}
	if len(topics) == 0 {
		return nil
	}

	conn, err := kafka.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("dial kafka %s: %w", brokers[0], err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("kafka controller: %w", err)
	}

	controllerAddr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
	ctrlConn, err := kafka.DialContext(ctx, "tcp", controllerAddr)
	if err != nil {
		return fmt.Errorf("dial kafka controller %s: %w", controllerAddr, err)
	}
	defer ctrlConn.Close()

	specs := make([]kafka.TopicConfig, 0, len(topics))
	for _, t := range topics {
		partitions := t.Partitions
		if partitions <= 0 {
			partitions = 1
		}
		replicas := t.ReplicationFactor
		if replicas <= 0 {
			replicas = 1
		}
		entries := make([]kafka.ConfigEntry, 0, len(t.Config))
		for k, v := range t.Config {
			entries = append(entries, kafka.ConfigEntry{ConfigName: k, ConfigValue: v})
		}
		specs = append(specs, kafka.TopicConfig{
			Topic:             t.Name,
			NumPartitions:     partitions,
			ReplicationFactor: replicas,
			ConfigEntries:     entries,
		})
	}

	if err := ctrlConn.CreateTopics(specs...); err != nil {
		// kafka-go returns aggregate errors keyed by topic. We surface the
		// first non-"already exists" error to the caller.
		return fmt.Errorf("create topics: %w", err)
	}
	return nil
}
