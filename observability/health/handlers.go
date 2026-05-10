package health

import (
	"log/slog"
	"net/http"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/health/internal/core"
)

// LiveHandler returns the /health/livez handler. It responds with HTTP
// 200 and a constant {"status":"alive"} body unconditionally while the
// process serves. Liveness exists to catch the case where the HTTP
// responder itself has stopped serving — anything beyond an
// unconditional response would create a kill-restart loop on transient
// downstream blips. See CONTEXT.md "Live" for the discipline.
func LiveHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(core.LiveBody); err != nil {
			// Connection closed mid-write; nothing recoverable here.
			slog.Debug("livez write", slog.String("err", err.Error()))
		}
	})
}

// ReadyHandler returns the /health/readyz handler. It runs every Pinger
// in the supplied map in parallel under the fixed 800 ms internal
// deadline (core.ProbeDeadline) and emits the aggregated response as
// JSON. HTTP status is 200 when every Pinger succeeds, 503 when any
// fails. Content-Type is always application/json so operators can pipe
// the response through jq.
//
// Sibling-isolation invariant: an individual Pinger's failure does NOT
// cancel sibling Pingers within the same call. Every configured check
// is reported in the response, regardless of how many earlier checks
// failed. Only the 800 ms deadline cancels the aggregate.
func ReadyHandler(checks map[string]Pinger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, status := core.Evaluate(r.Context(), checks)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if _, err := w.Write(body); err != nil {
			// Connection closed mid-write; nothing recoverable here.
			slog.Debug("readyz write", slog.String("err", err.Error()))
		}
	})
}
