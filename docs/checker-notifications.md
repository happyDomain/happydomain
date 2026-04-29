# Checker notifications

Notifications turn checker results into user-visible alerts. The system watches
every completed `Execution`, decides whether the status transition deserves an
alert under the user's preferences, and fans the message out to one or more
delivery channels (email, webhook, UnifiedPush). Audit records and per-issue
acknowledgements are kept so the user can see what was sent, and silence noisy
incidents until they recover.

## Goals

- **Transition driven, not poll driven.** Only state changes notify. Identical
  back to back results stay silent. This is what keeps a flapping checker from
  paging on every run.
- **Per user, per scope policy.** A user can configure preferences globally,
  per domain, or per service, and the most specific rule wins (service over
  domain over global). Channels are owned by the user, and may be allow listed
  per preference.
- **Opt in by default.** A user with no configured preference still receives
  `warn` and above alerts on all enabled channels, so onboarding does not
  require authoring rules. A configured preference can lower or raise that
  bar, suppress recovery notices, or set quiet hours.
- **Acknowledgement closes the loop.** A user can acknowledge an active issue
  and stop further alerts at the same severity until the incident recovers or
  escalates.
- **Auditable delivery.** Every send, and every failure (including back
  pressure drops), is recorded so users can confirm an alert really left the
  building.
- **Decoupled from the checker engine.** Notification dispatch hangs off a
  callback the engine fires after each execution, so a slow channel cannot
  wedge a checker run.

## Architecture

The checker engine fires a registered callback after each `ExecutionDone`. The
callback enters the notification subsystem, which loads the user's state and
preferences, runs a pure policy function over the status transition, persists
the new state, and on a positive decision enqueues sends onto a bounded worker
pool. Workers resolve the matching sender from a type indexed registry, and
write a record of the result into the audit log.

State, preferences, channels, and records each have a dedicated storage
interface. A per `(checker, target, user)` mutex serialises the read modify
write done by the dispatcher and by manual acknowledgement actions, so an ack
cannot be silently overwritten by a concurrent checker run. The HTTP layer
(`internal/api/route/notification.go`) exposes CRUD on channels, preferences,
history, and the acknowledge and clear endpoints scoped to a checker. The
Svelte UI under `web/src/routes/me/notifications/` drives all of this.

Key types:

- `Dispatcher` (`internal/usecase/notification/dispatcher.go`): the seam
  between checker and notification, glues every collaborator together and
  owns no I/O of its own.
- `Resolver`: picks the most specific preference for a target, and the
  channels (filtered by the preference allow list) that should carry the
  alert.
- `policy.decide`: pure function returning skip, advance, or notify, plus
  recovery, escalation, and clear ack flags. Unit tested.
- `StateLocker`: in process per key mutex shared by the dispatcher and the
  ack service.
- `Pool`: bounded queue (256) and fixed worker pool (4) with a 15s send
  timeout, records every result and writes an audit row on saturation rather
  than dropping silently.
- `Registry` and `ChannelSender` (`internal/notification/sender.go`): typed
  registry of transports, with `TypedSender[C]` and `Adapt` providing
  decode, validate, redact, merge, and send test for free.
- `AckService`: acknowledge, clear, get, and list state, behind the same
  state lock.
- Models (`model/notification.go`): `NotificationChannel`,
  `NotificationPreference`, `NotificationState`, `NotificationRecord`.

### Decision flow inside `policy.decide`

1. `oldStatus == newStatus` returns skip.
2. Compute `isRecovery` (`newStatus < warn` while `oldStatus >= warn`) and
   `isEscalation` (`newStatus > oldStatus && newStatus >= warn`). Either one
   sets `ClearAck`, since the incident is over or has worsened.
3. No preference, or `pref.Enabled=false`, returns advance (record the
   transition, do not send).
4. Non recovery below `pref.MinStatus` returns advance.
5. Recovery while `!pref.NotifyRecovery` returns advance.
6. Active acknowledgement, with `!ClearAck`, and not a recovery, returns
   advance. The user already knows.
7. Inside `pref.QuietStart..QuietEnd` (UTC, wraps midnight if start is
   greater than end) returns advance.
8. Otherwise, notify.

Advance updates `LastStatus` only. Notify also stamps `LastNotifiedAt`, and
this stamp is written **before** enqueuing, so a fast re run sees the new
status and skips, even if the worker pool has not yet drained.

## Existing channel implementations

All senders live in `internal/notification/`. They share `safe_http.go`
(outbound URL validation against private and loopback ranges, redirect
bounds), and where relevant `httpjson.go` for JSON POSTs. Each registers a
constant of type `happydns.NotificationChannelType`.

| Type          | Constant                | Config (JSON)                              | Notes                                                                             |
|---------------|-------------------------|---------------------------------------------|-----------------------------------------------------------------------------------|
| `email`       | `ChannelTypeEmail`      | `{ "address"?: string }`                    | Falls back to the user's account email. Renders via the existing `Mailer`. Base URL captured at construction. |
| `webhook`     | `ChannelTypeWebhook`    | `{ "url": string, "secret"?: string, ... }` | POSTs JSON. Optional HMAC SHA256 signature header derived from `secret`.  |
| `unifiedpush` | `ChannelTypeUnifiedPush`| `{ "endpoint": string }`                    | POST to the distributor provided endpoint. URL is validated against the safe HTTP allow list. |

Each sender is implemented as a `TypedSender[C]` and exposed through `Adapt`,
so JSON decode, `Validate`, redact, merge, and `SendTest` are uniform across
transports. Senders that carry secrets implement `ConfigRedactor[C]`, so the
client never sees the secret, and `ConfigMerger[C]`, so an empty value on
update preserves the stored secret rather than wiping it.
