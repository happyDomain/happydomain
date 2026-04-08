# happyDomain Metrics

happyDomain exposes Prometheus metrics at `GET /metrics` on the **admin
socket only** (Unix socket or loopback). The admin socket is not
authenticated; do not expose it to untrusted networks. The public HTTP API
does **not** serve `/metrics`.

All metric names are prefixed with `happydomain_`.

## Exported metrics

| Metric | Type | Labels | Cardinality bound | Description |
|---|---|---|---|---|
| `happydomain_http_requests_total` | counter | `method`, `path`, `status` | HTTP methods × Gin route templates × HTTP status codes (low hundreds) | Total HTTP requests served. `path` is the Gin route template (e.g. `/api/domains/:domain`), never the raw URL, to keep cardinality bounded. |
| `happydomain_http_request_duration_seconds` | histogram | `method`, `path` | same as above | HTTP request latency, default Prometheus buckets. |
| `happydomain_http_requests_in_flight` | gauge | – | 1 | HTTP requests currently being served. |
| `happydomain_scheduler_queue_depth` | gauge (func) | – | 1 | Sampled at scrape time via `RegisterSchedulerQueueDepth`. Reports 0 when no scheduler is registered. |
| `happydomain_scheduler_active_workers` | gauge | – | 1 | Workers currently executing a check. |
| `happydomain_scheduler_checks_total` | counter | `checker`, `status` | #checker types × {`success`, `error`} | Total scheduler check executions. Checker IDs are system-defined, never user input. |
| `happydomain_scheduler_check_duration_seconds` | histogram | `checker` | #checker types | Check execution latency. |
| `happydomain_provider_api_calls_total` | counter | `provider`, `operation`, `status` | #providers × #ops × {`success`, `error`} | DNS provider API calls. `provider` is the dnscontrol provider name (bounded set). |
| `happydomain_provider_api_duration_seconds` | histogram | `provider`, `operation` | same | DNS provider API latency. |
| `happydomain_storage_operations_total` | counter | `operation`, `entity`, `status` | ~6 ops × ~5 entities × {`success`, `error`} | Storage operations. |
| `happydomain_storage_operation_duration_seconds` | histogram | `operation`, `entity` | same | Storage operation latency. |
| `happydomain_storage_stats_errors_total` | counter | `entity` | #entities | Errors encountered while collecting storage stats during a scrape. Alert on a non-zero rate — silent storage failures otherwise produce gaps in the gauges below. |
| `happydomain_registered_users` | gauge | – | 1 | Registered user accounts (sampled live at scrape time). |
| `happydomain_domains` | gauge | – | 1 | Domains managed across all users. |
| `happydomain_zones` | gauge | – | 1 | Zone snapshots stored. |
| `happydomain_providers` | gauge | – | 1 | Provider configurations across all users. |
| `happydomain_build_info` | gauge | `version`, `revision`, `dirty`, `build_date` | 1 per build | Always 1; metadata is in the labels. |

## Cardinality rules

- **Never** add a label whose value comes from user input: domain name, user
  ID, zone ID, provider URL, raw HTTP path, etc.
- New labels MUST have a documented finite bound in the table above before
  the metric is merged.
- Histograms inherit the cardinality of their labels — be especially careful.

## Security

`/metrics` exposes business intelligence (entity counts, provider mix,
latency profiles) and operational shape (queue depth, worker counts). It is
intentionally only mounted on the admin socket (`internal/app/admin.go`).
Bind that socket to a Unix path or `127.0.0.1` only — exposing it on a
network interface will leak this information to anyone who can reach it.

## Implementation notes

- The HTTP middleware uses `c.FullPath()` (Gin route template) to populate
  the `path` label. See `internal/metrics/http.go`.
- The scheduler queue depth gauge is a `GaugeFunc` that calls back into the
  scheduler at scrape time, installed via
  `metrics.RegisterSchedulerQueueDepth`. The scheduler unregisters its
  accessor in `Stop()` so stopped schedulers do not leak their queue.
- The storage stats collector runs each `Count*` query in its own goroutine
  with a `recover()` guard, so a panicking backend cannot crash the scrape.
  Failures increment `happydomain_storage_stats_errors_total{entity=…}`.
- `happydomain_build_info` is set once at startup from `cmd/happyDomain/main.go`
  using `versioninfo.Revision`, `LastCommit`, and `DirtyBuild`.
