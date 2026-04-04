# Checker scheduling and execution

happyDomain's checker subsystem runs small health checks against DNS zones,
domains, and services. Each check can be triggered in two ways (on a
recurring schedule, or manually from the UI/API), and both paths share the
same execution pipeline (observation → rule evaluation → aggregated status).

This document describes:

1. The two ways a check can be triggered (scheduled vs manual).
2. How the execution pipeline turns an observation into a status.
3. How options are composed across scopes and how auto-fill variables work.
4. Where to find the per-user throttling rules.

> **See also:** [checker-quotas.md](./checker-quotas.md) for the per-user
> scheduling gate (pause, inactivity, daily budget) that sits in front of
> every scheduled and manual trigger.

---

## Key concepts

A few terms recur throughout this document and the checker APIs:

- **Checker.** The unit of monitoring logic: a named program (built-in,
  plugin, or external HTTP service) that declares a set of observations
  it can collect and a set of rules that evaluate them. A checker is
  identified by a stable `checkerId` and described by a
  `CheckerDefinition` (options, intervals, availability, rules).
- **Observation.** The raw data produced by a single collect step: for
  example, the RTT samples of a ping probe or the response of a DNS
  query. Each observation is typed, identified by an
  `ObservationKey`, serialised to JSON, and stored in an
  `ObservationSnapshot`. A checker may expose several observation
  keys; rules request the ones they need.
- **Rule.** A named predicate that turns observations into a
  `CheckState` (`StatusOK`, `StatusWarn`, `StatusCrit`, ...). A single
  checker typically ships several rules that look at the same
  observation from different angles (e.g. "packet loss" and "latency"
  on the same ping data). Users can enable or disable rules
  individually per target.
- **Target.** What a check runs against: a user, a domain, a
  (zone, subdomain) pair, or a service inside a zone. The
  `CheckerAvailability` of a checker decides which target levels it
  may attach to.
- **CheckPlan.** The per-target configuration that ties a checker to a
  specific target. It carries the user's (or admin's) overrides:
  interval, and a per-rule `enabled` map. A plan is optional: if none
  exists, the checker runs with its declared defaults.
- **Execution.** A single run of the pipeline for a given
  (checker, target). It is created either by the scheduler (when a
  plan or default schedule is due) or by a manual trigger, and
  progresses through the lifecycle statuses `Pending → Running →
  Done | Failed`.
- **CheckEvaluation.** The result attached to a finished execution: a
  list of rule states plus a reference to the `ObservationSnapshot`
  they were computed from. This is what the UI renders as green / amber
  / red badges.

In short: a **CheckPlan** pairs a checker with a target; each run
produces one **Execution**, which collects one or more **Observations**
into a snapshot and feeds them to the enabled **Rules**; the rules'
states are bundled into a **CheckEvaluation**, which becomes the
execution's final result.

## How a check runs

### 1. Scheduled runs (automatic)

The scheduler maintains a priority queue of upcoming jobs and, on every
tick, pops the jobs whose `NextRun` is due. A job that passes the
[user-level gate](./checker-quotas.md#the-three-gates) is handed to the
engine, executed, and re-enqueued for its next run.

**Interval resolution.** The effective interval for a given (checker,
target) pair is chosen in this order:

1. `CheckPlan.Interval`: the per-target override stored in DB, if the
   user (or admin) set one.
2. `CheckerDefinition.Interval.Default`: the checker's own default,
   declared at registration time.
3. `24h`: the hardcoded fallback used when the checker did not declare
   an interval spec.

The result is then **clamped** to `[Interval.Min, Interval.Max]` if the
checker declared bounds, so a user cannot set `Interval = 10s` on a
checker whose minimum is `1m`.

**Spread and jitter.** To avoid thundering herds when many checks share
the same interval, the scheduler adds:

- a **deterministic offset** per (checkerID, target) hashed into the
  interval window, and
- a ~5 % **deterministic jitter** per cycle.

Two users who configure the same 5-minute probe on the same day will
therefore run it on different sub-minute offsets.

**Applicability.** A checker is scheduled for a target only if its
`CheckerAvailability` matches: `ApplyToDomain` enrolls every domain,
`ApplyToZone` every published zone, and `ApplyToService` every service
of a type listed in `LimitToServices` (or all of them, if the list is
empty).

### 2. Manual runs ("Run now")

Users (and admins) can trigger an execution on demand:

```
POST /api/domains/{domain}/checkers/{checkerId}/executions
POST /api/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/executions
```

The request body is optional:

```json
{
  "options":      { "...": "..." },
  "enabledRules": { "ruleName": true, "otherRule": false }
}
```

- `options` are **run-time overrides** that take effect for this single
  execution. They are validated against the checker's `RunOpts`.
- `enabledRules` temporarily selects a subset of the rules for this run;
  omit it to evaluate every rule configured for the target.

Behaviour:

- By default, the endpoint returns **HTTP 202** with the newly created
  `Execution` and runs the pipeline asynchronously. Poll
  `GET .../executions/{id}` (or its aliases) for completion.
- With `?sync=true`, it blocks and returns **HTTP 200** with the
  resulting `CheckEvaluation`.
- Manual triggers are subject to the per-user daily budget and may be
  refused with **HTTP 429**. See
  [checker-quotas.md](./checker-quotas.md#daily-budget).

The handler is `TriggerCheck` in the checker API controller.

## The execution pipeline

Whether the trigger is scheduled or manual, a single execution goes
through the same steps:

1. **Resolve options.** Merge admin → user → domain → service → run-time
   overrides into a single `CheckerOptions`, then apply
   [auto-fill](#auto-fill-variables).
2. **Collect observations.** Each `ObservationProvider` referenced by
   the enabled rules is invoked (with caching: two rules sharing the
   same observation key only collect once). Providers return a typed
   struct that is JSON-serialised into an `ObservationSnapshot`.
3. **Evaluate rules.** Each enabled `CheckRule` receives the snapshot
   (via an `ObservationGetter`) plus the resolved options, and returns a
   `CheckState` with one of the statuses below.
4. **Aggregate** the individual rule states into the execution's final
   result (worst-status wins, by default) and persist both the snapshot
   and the evaluation.
5. **Update the `Execution`** with its terminal `ExecutionStatus` and a
   link to the evaluation.

For the anatomy of a checker (data types, providers, rules, metrics),
see the companion project
[checker-dummy](https://git.happydns.org/checker-dummy), which is the
reference walkthrough.

### Check statuses

Returned by each rule and aggregated at the execution level:

| Constant         | Meaning                                         |
| ---------------- | ----------------------------------------------- |
| `StatusOK`       | Healthy.                                        |
| `StatusInfo`     | Informational, not a problem.                   |
| `StatusWarn`     | Soft threshold crossed.                         |
| `StatusCrit`     | Hard threshold crossed.                         |
| `StatusError`    | The rule itself could not evaluate (e.g. bad data). |
| `StatusUnknown`  | Not enough information to decide.               |

### Execution lifecycle statuses

Attached to the `Execution` record (not to individual rule states):

| Status                 | Value | Meaning                                               |
| ---------------------- | ----- | ----------------------------------------------------- |
| `ExecutionPending`     | `0`   | Created, not yet running.                             |
| `ExecutionRunning`     | `1`   | Pipeline is executing.                                |
| `ExecutionDone`        | `2`   | Pipeline completed; see the linked `CheckEvaluation`. |
| `ExecutionFailed`      | `3`   | Pipeline errored before producing an evaluation.      |
| `ExecutionRateLimited` | `4`   | Synthetic, only on planned (not-yet-run) entries returned by `?include_planned=1`; never persisted. See [checker-quotas.md](./checker-quotas.md#ui-signalling). |

## Configuring a check

Every checker exposes a set of typed options grouped by **scope**. The
scope determines who sets the option and for how many targets it
applies at once:

| Scope          | Who sets it        | Applies to                 |
| -------------- | ------------------ | -------------------------- |
| `AdminOpts`    | instance admin     | every user on the instance |
| `UserOpts`     | end user           | all their own targets      |
| `DomainOpts`   | end user / auto    | a single domain            |
| `ServiceOpts`  | end user / auto    | a single service           |
| `RunOpts`      | trigger caller     | a single execution         |

At execution time the scopes are merged in order of increasing
specificity (`admin → user → domain → service → run`), so a per-service
value wins over a per-domain one, which wins over a per-user default,
and so on. Admin-provided values can be locked with the `NoOverride`
attribute to prevent lower scopes from changing them.

The interval itself is **not** an option; it lives on the `CheckPlan`
record for that (checker, target) pair, as described above.

### Auto-fill variables

Some options don't need to be typed in manually: they can be resolved
from the surrounding context of the execution. Each such option is
declared with an `AutoFill` attribute, and the engine populates it just
before the collect step. Supported variables:

| Constant              | Resolves to                                          |
| --------------------- | ---------------------------------------------------- |
| `AutoFillDomainName`  | The target domain's FQDN.                            |
| `AutoFillSubdomain`   | The subdomain under which the target service lives.  |
| `AutoFillZone`        | The published zone the check is running against.     |
| `AutoFillServiceType` | The service's type identifier.                       |
| `AutoFillService`     | The full service payload.                            |

Auto-fill runs **last** and overrides any value that may have been set
at a lower scope: the goal is to keep these fields authoritative. Rule
code can therefore rely on them being present and correct at
`Evaluate()` time without re-deriving them from the observation.

## Operational notes

- **Same pipeline for both triggers.** A manual run and a scheduled run
  produce an `Execution` of the same shape; the only distinction is the
  `TriggerInfo.Type` (`TriggerManual` vs `TriggerSchedule`) stored on
  the execution.
- **User gate first.** The per-user pause / inactivity / daily-budget
  gate is evaluated before the pipeline starts; a gated scheduled job
  is dropped and the counter is not incremented. The rules are in
  [checker-quotas.md](./checker-quotas.md).
- **Rules can be disabled per target.** `CheckPlan.Enabled` is a rule
  name → boolean map. A missing entry means "enabled", an empty map
  means "all enabled". A plan where every rule is explicitly `false`
  disables the checker entirely for that target
  (`CheckPlan.IsFullyDisabled`).
- **Options validation.** Options are validated both when a plan is
  stored and when a manual trigger arrives (with `RunOpts` included for
  the latter). Invalid options yield HTTP 400 at trigger time;
  `TriggerCheck` does not silently drop bad input.
