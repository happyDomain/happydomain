# Checker quotas and scheduling policy

happyDomain's checker subsystem runs scheduled DNS health checks on behalf of
each user. To keep resource usage predictable on shared instances, the
scheduler consults a per-user policy before every job and can throttle or
skip executions based on the user's activity, explicit pause, or daily
budget.

This document describes:

1. The three gates a scheduled check must pass through.
2. How per-day budgets are counted and reset.
3. How an administrator configures system-wide defaults.
4. How per-user overrides work.
5. Operational caveats (restart behaviour, manual triggers, UI indications).

> **Scope:** this document only covers the scheduler's user-level gate. It
> does not cover retention (`--checker-retention-days`, see the janitor) nor
> the per-checker `MinInterval` throttling.

---

## The three gates

Before each scheduled execution, the scheduler evaluates the job against a
**user-level gate**. A job is dropped (and rescheduled for the next tick) if
any of the following apply:

| Gate | Blocks when… | Source of truth |
| --- | --- | --- |
| Scheduling paused | `UserQuota.SchedulingPaused == true` | per-user only |
| Inactivity pause | user has not logged in for N days | per-user, falls back to system default |
| Daily budget | user has executed `MaxChecksPerDay` scheduled checks today | per-user, falls back to system default |

The first two gates ("policy layer") are cached for 5 minutes after a lookup
so the scheduler hot path does not hit storage on every job pop. The cache
is invalidated automatically whenever a user's quota or `LastSeen` changes
(login, admin edit).

The daily budget gate is **not** cached: the counter changes on every
successful execution and must be accurate.

## Daily budget

### How it is counted

- The counter is incremented **once** per successful call to
  `CreateExecution` by the scheduler. It is _not_ decremented if
  `RunExecution` later fails — a check that got as far as being recorded
  counts against the budget. This prevents users from burning capacity via
  engines that repeatedly error.
- **Manual API triggers are counted by default**, and refused with
  `HTTP 429 Too Many Requests` once the user is over budget. This applies
  to "Run now" in the UI and to `POST
  /api/domains/.../checkers/.../executions`. The behaviour is controlled
  by `--checker-count-manual-triggers` (default `true`); set it to
  `false` to restore the legacy bypass — in that mode, manual triggers
  are neither checked nor incremented.

### When it resets

- The counter resets at **00:00 UTC** every day. This is intentionally
  independent of the user's local timezone so the behaviour is consistent
  across deployments. A user in UTC+8 will see their counter flip
  mid-afternoon local time.
- The counter is kept **in memory only**. A process restart resets it to
  zero for every user. In a rolling-restart environment, this effectively
  grants a partial top-up; operators should size their defaults with this
  in mind.

### Interval-aware throttling

When the budget is at least **80 % consumed**, the scheduler starts skipping
jobs whose configured interval is shorter than **4 hours**. Jobs with a
longer interval continue to run until the hard limit is reached.

The goal is to prevent rare-but-important checks (for example, a daily
DNSSEC sanity check) from being starved by frequent low-value pings (for
example, a 1-minute probe). Put bluntly: if you are going to run out of
budget anyway, spend the last 20 % on the checks you would most miss.

Constants (not currently configurable at runtime):

- `throttleFillRatio` = 0.8
- `throttleShortIntervalCutoff` = 4 h

Both live in `internal/usecase/checker/user_gate.go`.

### UI signalling

Planned (not-yet-run) executions returned by the "upcoming checks" API are
marked with status `ExecutionRateLimited` whenever the target user is over
budget. This lets the frontend show the user that scheduled work is on hold
until tomorrow, distinct from a merely-pending job.

Only _synthetic_ planned entries carry this status; it is never persisted
on a real execution record.

## System-wide configuration

| CLI flag | Default | Meaning |
| --- | --- | --- |
| `--checker-inactivity-pause-days` | `90` | Stop scheduling for users inactive for this many days. `0` disables the inactivity gate. |
| `--checker-max-checks-per-day` | `0` | Cap on scheduled executions per user per day. `0` means unlimited. Counter resets at 00:00 UTC. |
| `--checker-count-manual-triggers` | `true` | When `true`, manual triggers count against `MaxChecksPerDay` and are refused with HTTP 429 once exhausted. When `false`, manual triggers bypass the quota entirely. No effect when `MaxChecksPerDay` is `0` (unlimited). |

Example:

```sh
happyDomain \
  --checker-max-checks-per-day=2000 \
  --checker-inactivity-pause-days=60 \
  --checker-count-manual-triggers=true
```

Values set via the CLI are read at startup. Changing them requires a
restart of the happyDomain process.

## Per-user overrides

Administrators can override the system defaults for individual users via
the admin API (`UserQuota`):

```json
{
  "max_checks_per_day": 500,
  "inactivity_pause_days": 14,
  "scheduling_paused": false
}
```

Semantics:

- `max_checks_per_day`: `0` means "use the system default", any positive
  value is an override, and a **negative** value disables the daily cap
  for that user (explicit unlimited, independent of the system default).
  Changes take effect immediately — on the next scheduler tick, the
  user's budget cache is recomputed against the new limit while the
  accumulated usage counter for the current day is preserved.
- `inactivity_pause_days`: `0` means "use the system default", any
  positive value is an override, and a **negative** value disables
  inactivity pausing for that user.
- `scheduling_paused`: hard per-user override. Takes effect on the next
  scheduler tick (within the cache TTL of 5 minutes; an admin edit
  invalidates the cache immediately).

After editing a user's quota via the admin API, the gate cache is
invalidated automatically. You should not need to restart the scheduler.

## Operational notes

- **Failed executions still count.** If the checker engine is misbehaving
  for an unrelated reason, users will see their budget drained by failed
  runs. Watch the scheduler logs (`Scheduler: checker ... failed: ...`)
  and the `ExecutionFailed` status counts in the admin metrics.
- **Process restarts clear the counters.** For deployments with a hard
  budget, avoid frequent restarts during peak hours.
- **Manual triggers count by default.** With
  `--checker-count-manual-triggers=true` (the default), pressing
  "Run now" consumes one unit of the daily budget and returns HTTP 429
  once the budget is exhausted. Set the flag to `false` to restore the
  legacy bypass; that option is useful for self-hosted instances where
  the quota is only meant to protect against runaway scheduled load.
- **HTTP 429 response body.** Rejected manual triggers return
  `{"errmsg": "daily check quota exhausted; try again after 00:00 UTC"}`
  alongside the 429 status. Clients should surface this to the user —
  the `ExecutionRateLimited` status (value `4`) used for planned
  executions in `?include_planned=1` can be reused for consistent
  iconography.
- **Invalidate is cooperative.** The gate cache is invalidated on user
  update and on admin quota edit. If you modify user data through a
  backdoor (direct database write, migration, …) the cache will stay
  stale until its 5-minute TTL expires.

## Diagnostics

There is currently no dedicated endpoint to introspect per-user usage.
Pragmatic options, in order of effort:

1. Read the scheduler logs: every gated job emits a line at debug level
   (commented out by default in `scheduler.go`; enable locally when
   investigating).
2. Query planned executions via the user-facing API with
   `?include_planned=1` and look for `ExecutionRateLimited` entries —
   this confirms a user is currently over budget.
3. Restart the process to force a clean slate (destructive to all
   counters; use only as a last resort).
