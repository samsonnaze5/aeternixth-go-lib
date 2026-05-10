// Package logger constructs a structured slog.Logger configured for the
// fleet's cloud-native deployment pattern: JSON output, level-filtered,
// no embedded timestamp (the deployment platform — Kubernetes, log
// aggregator — adds the timestamp at ingestion time).
//
// This package coexists with the repository-root logutil package; they
// serve different purposes:
//
//   - logger (this package) — production structured logger meant to be
//     used as the binary's primary slog.Logger.
//   - logutil — debug print helper for inspecting Go values during
//     development (logutil.Payloader). Not for production logs.
//
// Adopters wire logger once at startup:
//
//	log := logger.New(cfg.Log.Level, os.Stderr)
//	log.Info("starting", slog.String("service", "mt5-processor"))
package logger
