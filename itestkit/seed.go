package itestkit

// Seed execution shares the same plumbing as migration execution. The seed
// runners live in migration.go (applyPostgresSeeds, applyClickHouseSeeds)
// to avoid duplicating the file-collection and execution logic. This file
// is kept to match the spec's documented layout.
