// Package healthkafka adapts a Kafka cluster (broker list + topic list)
// to the
// [github.com/samsonnaze5/aeternixth-go-lib/observability/health.Pinger]
// contract via the Metadata RPC.
//
// The pinger dials the first reachable broker, runs Metadata, and
// asserts that every supplied topic exists. Per ADR-0001 in this
// repository, only the Metadata variant ships — Dial-only is rejected
// because it cannot detect a topic deleted out from under the service.
//
// Construct via [NewMetadataPinger]; non-empty broker and topic lists
// are required. See ADR-0002 for Kubernetes probe spec guidance for
// services using this pinger.
package healthkafka
