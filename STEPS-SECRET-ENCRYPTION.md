# Secret encryption: implementation plan

This plan turns [docs/secret-management.md](docs/secret-management.md) into
ordered, reviewable steps. The design document says *what* and *why*; this file
says *in which order* and *how we know a step is done*. When the two disagree,
fix the design document first.

## Working rules

- **Test first, always.** Every new function starts as a failing test. A step
  lists its tests before its implementation, and they are committed together.
  Existing behavior touched by a step gets a characterization test first, so a
  regression shows up as a red test rather than in production.
- **Simplicity over coverage.** Serve the common case with bulletproof
  reliability. Anything rare, speculative or needing a new option goes to
  [Deferred](#deferred), not into the code. When in doubt, cut.
- **Fail closed.** When the code cannot tell whether something is safe
  (unknown state, unknown safe, missing key), it refuses with an error instead
  of guessing. A 500 is acceptable; a leaked secret is not.
- **No home-made cryptography.** Primitives come from Tink or `x/crypto`
  (argon2id, HKDF). No custom cipher construction, no custom padding.
- **Stable formats are tested as such.** Parsers get Go fuzz tests. Every
  persisted format (sealed value, wrapped keyset, associated data) gets golden
  vectors: fixed inputs whose expected output is committed, so a later change
  that breaks old data fails the build.
- **One step, one or a few commits**, each building and passing tests. A phase
  ends with a security review of its whole diff (`/security-review`) before
  merging. Make atomic commits.
- **Running tests** on the development machine:
  `GOTMPDIR=$PWD/.gocache/tmp go test ./model/... ./internal/secret/...`
  (package by package when disk is short; clean `.gocache/tmp` afterwards).

Each step below has: **Goal**, **Tests first**, **Implementation**, **Done
when**. Checkboxes track progress.

---

## Phase 1: `Secret` type, instance key, rotation

At the end of this phase, an administrator can turn on encryption of every
stored secret under an instance keyset, rotate that keyset, and migrate existing
data, with no visible change for users. This alone answers the audit.

Out of this phase on purpose: any user choice, any external store, any unlock.
There is exactly one policy, instance-wide: `plaintext` (default) or
`instance`.

### Step 1.1: the `happydns.Secret` type

**Goal.** A value type for secret fields that cannot be persisted without going
through sealing, and cannot leak through formatting.

States (internal, not exported):

| State | Comes from | `Reveal()` | `MarshalJSON` |
|---|---|---|---|
| empty | zero value, `""`, `null` | `""` | `""` |
| clear | API input, or an unprefixed value read from storage | the value | **error** `ErrUnsealedSecret` |
| sealed | a `hds:1:` value read from storage | `""` | the sealed token |
| opened | a sealed value after `Open` | the value | the sealed token |
| redacted | `RedactedSecret` from the client, or `Redact()` | `""` | `RedactedSecret` |

The plaintext policy "seals" by moving a clear value to the opened state with
the raw value as token, so existing records keep their exact format.

**Tests first** (`model/secret_test.go`):
- zero value is empty, marshals to `""`, `IsEmpty()` true;
- unmarshal of `"abc"` gives clear; marshal of clear fails with
  `ErrUnsealedSecret`;
- unmarshal of `"hds:1:…"` gives sealed; marshal round-trips the token
  byte for byte; `Reveal()` is empty;
- unmarshal of `RedactedSecret` gives redacted, marshals back to it;
- unmarshal of `null` gives empty; of a number, object or array fails;
- `fmt` with `%v`, `%+v`, `%#v`, `%s`, `%q` never prints the clear value,
  including when the secret is nested in a struct and a pointer;
- a struct holding a clear `Secret` fails `json.Marshal` as a whole (this is
  the fail-closed guarantee the storage relies on);
- the methods used by the sealing layer (set sealed, set opened, read clear
  for sealing) behave per the table; they live in the `happydns` package and
  are documented as reserved for `internal/secret`.

**Implementation.** `model/secret.go`. Unexported fields, value receivers where
possible so copies behave. Clear bytes held in a `[]byte` (no promise of memory
wiping: Go cannot guarantee it, the design document does not claim it).

**Done when** the tests pass and `go vet` is clean.

### Step 1.2: forms and API integration

**Goal.** Forms, redaction and merging recognize `Secret` by type.

**Tests first** (`internal/forms`):
- `GenField` on a `Secret` field reports type `string` and `Secret: true`,
  with or without the `secret` tag;
- `RedactSecrets` turns clear, sealed and opened values into redacted, keeps
  empty as empty;
- `MergeSecrets(existing, incoming)`: redacted incoming takes the stored value
  (whatever its state, without opening it); clear incoming wins; redacted with
  no existing becomes empty;
- existing `string` / `[]byte` behavior unchanged (current tests stay green).

**Implementation.** Extend `redactValue`, `mergeValue` and field generation.
Tell swag that `Secret` is a string (global type override or `swaggertype`).

**Done when** existing forms tests and new ones pass, generated swagger is
unchanged for converted fields.

### Step 1.3: sealed format, context and associated data

**Goal.** The pure, persisted building blocks, with no I/O.

- `SecretContext{Owner Identifier, ObjectType string, ObjectId string, Field string}`.
  `ObjectId` is a string so that checker options (keyed by target, not by one
  identifier) fit. `Field` is the JSON name path (`ApiToken`,
  `auth.token`), stable across Go renames.
- Sealed value: `hds:1:<safe id>:<payload>`, safe id as `Identifier.String()`,
  payload base64url without padding.
- Associated data: each component length-prefixed, in a fixed order:
  `"hds:1"`, safe id, owner, object type, object id, field. Length prefixes
  make `("ab","c")` and `("a","bc")` distinct.

**Tests first** (`internal/secret`):
- format then parse round-trips; parse rejects wrong version, missing parts,
  empty safe id, invalid base64, extra `:` in the safe id;
- `IsSealed` is true only for the exact prefix;
- `FuzzParseSealed`: never panics, and anything it accepts re-formats to the
  same string;
- AAD: every component change changes the output; the `("ab","c")` /
  `("a","bc")` pair differs; golden vector for one fixed context.

**Done when** tests and fuzz seed corpus pass.

### Step 1.4: struct walker

**Goal.** Find every `Secret` in an object and apply a function to it with the
right `SecretContext.Field`.

**Tests first:**
- walks nested structs, embedded structs, pointers to structs, and reports JSON
  field paths;
- ignores unexported fields and fields tagged `json:"-"`;
- a `Secret` inside a slice, or a map other than one of `Secret`s with string
  keys, is reported as an error (fail closed: not supported rather than
  silently skipped); those maps are walked in key order, each value under
  `path["key"]`;
- the callback error aborts the walk and is returned wrapped with the field
  path.

**Implementation.** `internal/secret/walk.go`, type-based (no tag needed).

### Step 1.5: `Manager` with the plaintext policy, wired into providers

**Goal.** Every provider write goes through `Seal`, every dial goes through
`Open`, with the plaintext policy only. Behavior is identical to today, but the
paths exist and are enforced.

`Manager`:
- `SealObject(ctx, sc, obj)`: every clear `Secret` sealed under the current
  policy; sealed/opened ones left as is; redacted ones are an error (they must
  have been merged before);
- `OpenObject(ctx, sc, obj)`: every sealed `Secret` opened; clear ones left
  (legacy plaintext); a secret it cannot open is an error;
- `CheckIncoming(obj)`: refuses a sealed value coming from a client (a client
  never has a reason to send one; accepting it would let a user paste a token
  from elsewhere).

With the plaintext policy, a clear value starting with `hds:1:` is refused at
seal time: it would read back as sealed.

**Tests first:**
- `Manager` unit tests for each rule above;
- provider use case (with an in-memory store and a test provider type holding
  a `Secret`): create, update with a new value, update echoing
  `RedactedSecret`, update changing type; the stored JSON is what it is today;
- `instantiate` and `DefaultProviderValidator` open before instantiating; the
  validator now calls `instantiate` rather than duplicating it (characterize
  the endpoint check first);
- a sealed value in a create/update request is refused with a validation
  error;
- storage fail-closed: writing a provider with a clear `Secret` through
  `CreateProvider` / `UpdateProvider` returns `ErrUnsealedSecret` and writes
  nothing;
- the admin provider API redacts: a legacy plaintext provider read through the
  admin API returns `RedactedSecret`, not an error and not the value;
- backup: the admin backup still copies stored records verbatim (it works on
  raw messages); restore seals clear values under the current policy and keeps
  sealed ones.

**Implementation.** `internal/secret/manager.go`; inject the manager into the
provider `Service`, the admin provider service, and the backup use case.
Audit every `json.Marshal` of a provider body (today: `Provider.ToMessage`,
used by backup) and cover each with a test.

**Done when** the full test suite passes with no provider converted yet (the
test provider type carries the logic).

### Step 1.6: convert the providers

**Goal.** Every secret field in `providers/` becomes a `happydns.Secret`.

**Tests first:**
- a test iterating over every registered provider:
  - every field tagged `secret` is a `Secret`;
  - every field whose name matches
    `(?i)key|token|secret|password|passwd|credential` is a `Secret`, except an
    explicit, commented allowlist (e.g. key *names*, key *ids*);
- for each provider, a stored legacy JSON (unprefixed values) still decodes,
  and `ToDNSControlConfig` / the libdns config receives the clear value after
  `OpenObject`;
- `axfrddns.KeyBlob` (today `[]byte`, stored as base64): the stored base64 text
  becomes the `Secret` value and is decoded where the key is used; a legacy
  record decodes to the same key bytes as before.

**Implementation.** Mechanical: type change plus `.Reveal()` at each use site.
One commit, reviewed with `git diff --word-diff`.

**Done when** the whole repository builds, all tests pass, and a manual run
creates, edits and uses a provider through the web UI with no difference.

### Step 1.7: object identifiers before sealing

**Goal.** The object id is known before sealing, since it is part of the
associated data.

**Tests first:**
- `CreateProvider` with a preset id stores under that id; with an id already
  used it fails; without one it still generates one (other callers);
- the provider use case sets the id before calling `SealObject`.

### Step 1.8: Tink and the instance keyset

**Goal.** Load, generate and rotate the instance keyset; refuse to start in an
inconsistent state.

Configuration:
- `-secret-keyset-file` (`HAPPYDOMAIN_SECRET_KEYSET_FILE`): a cleartext Tink
  JSON keyset (tinkey compatible), AES256-GCM. File permissions are the
  administrator's responsibility; startup warns when it is readable by others.
- `-secret-policy plaintext|instance` (default `plaintext`).

Subcommands (same pattern as `admin-hash`):
- `happydomain secret-keyset generate <file>`: new keyset, one primary key,
  refuses to overwrite;
- `happydomain secret-keyset rotate <file>`: adds a new key and makes it
  primary, keeps the old ones;
- `happydomain secret-keyset info <file>`: key ids, primary, status, never
  key material.

Startup checks:
- `instance` policy without a keyset: refuse to start;
- a keyset configured: a check record (`secret.check`) is decrypted, or
  created under the primary key when absent; failing to decrypt refuses to
  start;
- no keyset configured but instance safes exist in the database: refuse to
  start (they would all be unreadable).

**Tests first:** each subcommand (generate, refuse overwrite, rotate keeps old
keys decrypting, info shows no secret material); each startup case above;
golden vector: a committed test keyset decrypts a committed check record.

### Step 1.9: safes and keyrings

**Goal.** Per-user safes of kind `instance`, holding a DEK keyset wrapped by
the instance keyset.

- `Safe{Id, Owner, Kind, Keyring []WrappedKeyset, CreatedAt}`; phase 1 only
  knows kind `instance` and KEK `instance`.
- Storage: `safe-<id>` primary key, `safe-owner|<owner>|<kind>` index to find
  the default safe of a user. New `SafeStorage` interface, kvtpl
  implementation, instrumented storage regenerated.
- A user's instance safe is created on first seal, under a per-user in-process
  lock so two concurrent requests do not create two.
- Wrapping: `keyset.WriteWithAssociatedData`, associated data = safe id and
  owner, so a wrapped keyset cannot be moved to another safe.

**Tests first:** storage CRUD and index; get-or-create creates once under
concurrency (run with `-race`); wrap then unwrap round-trips; unwrap with
another safe's AAD fails; unwrap after instance key rotation still works with
the old key present; golden vector for a wrapped keyset.

### Step 1.10: the `inline` store and the `instance` policy

**Goal.** `SealObject` under the `instance` policy encrypts each field with the
user's safe DEK; `OpenObject` decrypts any sealed value whatever the current
policy.

**Tests first:**
- seal then open round-trips; the stored token starts with `hds:1:<safe id>:`;
- open fails when the token is copied to another field, another object,
  another owner (AAD);
- open of a token naming an unknown safe fails with a clear error;
- policy `instance` seals clear values; policy back to `plaintext` still opens
  existing sealed values, and seals new ones in clear (reverse migration);
- a clear legacy value is resealed on the next write of its object (lazy
  migration);
- golden vector: a committed safe plus a committed token open to a committed
  value.

No caching of unwrapped keysets in this phase: unwrapping is one local AES-GCM
operation.

### Step 1.11: admin status, reseal, rewrap

**Goal.** Let the operator migrate everything and know when an old key can go.

Admin API (on the existing admin socket):
- `GET /api/secrets/status`: per object type, number of secrets clear, sealed
  per safe kind, unreadable, and objects that cannot be looked at
  (undecodable, some of them named); per instance key id, number of safes
  wrapped with it, plus missing and unreadable keys;
- `POST /api/secrets/rewrap`: rewrap every safe under the primary instance key;
- `POST /api/secrets/reseal`: reseal every object under the current policy.

Both write operations are idempotent (rerunning after an interruption is the
resume mechanism). They tell two kinds of errors apart:

- an error of the *operation* (the listing fails): it stops;
- an error of *one object* (a record that does not decode, an unknown provider
  type, a secret that does not open, a key removed too early): it is reported,
  the object is left as it was, and the others go on. Otherwise one bad record
  would stop every rerun at the same place.

The status report never fails because of one object either: it counts what it
cannot read, since it is needed most when stored data is damaged.

A reseal or rewrap reads an object, then writes it back only if the stored
record is unchanged (`PutIfUnchanged`, native on PostgreSQL and Oracle NoSQL,
a transaction on LevelDB). Whatever another writer did in between wins,
whichever path it took (user API, restore, tidy, deletion): the object is
reported as skipped and taken again on the next run. No lock is needed, and
several happyDomain processes may share the database.

User updates carrying stored secrets forward (providers, notification
channels, checker options) write the same way, conditionally on what they
read: written unconditionally, an update read before a reseal would write the
old sealed values back, possibly tokens of safes about to be dropped. An update
losing the race answers `409 Conflict` (`happydns.ConflictError`); sending the
same request again succeeds, as it reads the object anew. This only covers a
write landing while the update is handled: a client submitting a form read
before someone else's change is not detected, as before this work (an ETag
would be needed, for every object of the application, not only those holding
secrets). An object owned by a
user that no longer exists is skipped too: no safe is ever created for a
deleted user.

**Tests first:** status counts on a seeded store; rewrap moves every safe to the
primary key and leaves the data readable; reseal from plaintext to instance and
back; interruption simulated by an erroring store halfway, then rerun completes;
a *permanent* per-object error (corrupt record, unknown type, key removed too
early) reported without stopping the others, on every rerun; concurrent
update, restore and deletion during reseal all win (`-race`); orphans skipped
without creating a safe.

### Step 1.12: lifecycle

**Tests first, then implementation:**
- deleting a user deletes their safes after their objects: their tokens become
  unreadable (crypto-shredding);
- `tidy` removes safes whose owner no longer exists, and only those;
- the admin backup includes safes; restoring on the same instance opens
  everything; restoring without the instance keyset refuses to start
  (step 1.8 check).

### Step 1.13: notification channels

**Goal.** `WebhookConfig.Secret` becomes a `Secret`, sealed with the policy.

- Seal in the notification channel use case: `DecodeConfig`, `SealObject`,
  re-encode. Open in the dispatcher before `Send` and on the admin
  `CheckConfig` path.
- Object type declares background access: fine with `plaintext` and
  `instance`, the only policies of this phase.
- `RedactConfig` / `MergeForUpdate` keep their current API contract
  (`hasSecret`), now backed by `Secret`.

**Tests first:** create/update/redact/merge round-trips; the dispatcher sends
with the clear secret (HMAC checked in the test); stored config never contains
the clear value under `instance`.

Webhook headers (every value, since any may carry a token) and UnifiedPush
endpoints (capability URLs) are sealed too, once `Walk` goes into maps of
secrets with string keys. A destination held in a secret is checked opened.

### Step 1.14: checker options

**Goal.** Option values whose documentation sets `Secret` are sealed.

- First, characterize: today these values are returned to the client
  unredacted. Decide (and test) redaction in the same step, with the
  `RedactedSecret` echo semantics of providers.
- Seal by key when options are saved; open in
  `BuildMergedCheckerOptionsWithAutoFill`.
- Only string values can be secret: a non-string value for a secret option is
  refused.
- `ObjectId` is the options target key (checker, user, domain, service).

**Tests first:** save seals; merge opens; redaction and echo-back; non-string
refused; the scheduler path (`Engine.runPipeline`) receives clear values.

### Step 1.15: documentation and release

- Administrator documentation: generating the keyset, enabling the policy,
  migrating, rotating, backing up the keyset (losing it loses every secret),
  single-process constraint for migrations.
- Changelog entry, including the admin API no longer showing credentials.
- `/security-review` on the phase diff; fix, then merge.

---

## Phase 2: choice of safe, external stores

At the end of this phase, the administrator can publish Vault, file and helper
safes, and users choose per provider among what the administrator allows.

Scope decisions for simplicity:
- **Safes are published by the administrator** (configured in the instance
  configuration). Users *select* them; they do not configure their own Vault or
  helpers yet.
- **Password managers go through exec helpers** (`op read`, `bw get`,
  `pass show`) rather than vendor SDKs: one mechanism, no new dependencies.

### Step 2.1: error classes and "who can open"

- `ErrLocked`, `ErrUnavailable`, `ErrDenied`, carried through `OpenObject` and
  mapped by the API to distinct error codes and messages.
- Each safe kind declares when it can be opened (always / best effort; "in an
  unlocked session" arrives in phase 3). Each object type declares whether it
  needs background access; the policy refuses a never-in-background safe for
  such an object at seal time, and accepts best effort with a warning.

**Tests first:** each error class surfaces with its API code; the compatibility
matrix object type × safe kind.

### Step 2.2: choosing the safe

- `SafeId` on provider meta (nil = inherit). Resolution when sealing: object,
  then user default, then instance default kind.
- Admin configuration lists the published safes and which kinds users may pick.
- API: list selectable safes; set a provider's safe (reseals its fields in the
  request); set the user default (reseals inheriting objects in the request).
- UI: safe selector on the provider form and in user settings; the secret
  field label follows the store (value or reference).
- Security restrictions (enabled kinds) checked again on every open.

**Tests first:** resolution order; explicit choice not moved by a default
change; inheriting objects resealed on default change; forbidden kind refused
at seal *and* at open (`ErrDenied`); policy change never breaks reading of a
still-allowed safe.

### Step 2.3: Vault KV store

- Instance-configured Vault (address, auth by AppRole or token, mount, path
  prefix). Path per secret:
  `<prefix>/<owner>/<object type>/<object id>/<field>`.
- `Put` / `Get` / `Delete`; acting user passed as request metadata for Vault's
  audit log.
- Delete after the database write; orphans collected by a `tidy` pass listing
  the prefix.

**Tests first:** against a fake Vault HTTP server: put/get/delete, auth
failure is `ErrUnavailable` or `ErrDenied` as appropriate, timeout is
`ErrUnavailable`, orphan collection. One opt-in integration test against a real
`vault server -dev` (skipped unless an env var is set).

### Step 2.4: Vault Transit as the instance KEK

- `tink-go-hcvault` as an alternative to the keyset file for wrapping safe
  keysets; the check record and rewrap work the same.

**Tests first:** same suite as step 1.9/1.11 with the Transit KEK behind a fake
server; opt-in real Vault test.

### Step 2.5: file store

- Directories listed by the administrator, with `{user}` substitution; access
  through `os.OpenRoot`; files readable by others refused; references validated
  by `ValidateRef`.
- Read only.

**Tests first:** reading inside the root; `..` and symlink escapes refused;
another user's directory unreachable through `{user}`; world-readable file
refused; missing file is `ErrUnavailable`; `ValidateRef` rejects absolute paths
and traversal.

### Step 2.6: exec helpers

- Named helpers declared by the administrator: path plus argument template with
  `{ref}` as a single argument; no shell.
- Protocol: `get` only in this phase; context as `key=value` lines on stdin;
  secret on stdout (size limit); timeout; scrubbed environment; stderr only
  logged at debug level.

**Tests first** (with small test helper programs built in the test): the
argument is passed verbatim even with spaces, quotes, `;`, `$()`; timeout kills
the process and returns `ErrUnavailable`; oversized output refused; non-zero
exit is an error without stderr in the message; environment contains only what
is allowed.

---

## Phase 3: unlocking safes

At the end of this phase, a user can protect a safe with their password, keep a
recovery key, and the administrator cannot read it at rest.

### Step 3.1: session unlock storage

- Unlocked keysets wrapped under `HKDF(session token, "hds unlock")` and stored
  alongside the session, keyed by session hash and safe id; deleted with the
  session (logout, expiry, close-all).

**Tests first:** wrap/unwrap with the token; unreadable with the database alone
(token absent); deleted with the session; expired session cannot unlock.

### Step 3.2: password KEK and recovery key

- argon2id with parameters stored in `WrappedKeyset.Params`.
- Creating a password safe requires the password; a recovery key (256 random
  bits, human-readable encoding) is generated and shown once.
- Login unlocks password safes automatically.
- Password change rewraps the `password` entry (old password already
  required).
- Forgotten password: reset proceeds as today; password safes stay locked
  until the user enters the recovery key, which rewraps `password`.
- Admin password reset: same as forgotten password.

**Tests first:** each flow above, including the "forgot password without
recovery key" case: secrets unreadable, clear message, nothing else broken.
Golden vector for argon2id parameters and a wrapped keyset.

### Step 3.3: lock/unlock API and UI

- `POST /safes/:id/unlock`, `POST /safes/:id/lock`; UI prompt on `ErrLocked`.

### Step 3.4: machine tokens

- When creating a machine token from an unlocked session, grant it some safes
  (wrap under the machine token). Revoking the token deletes the wrapping.

### Step 3.5: `file` and `exec` KEKs

- Same unlock flow, the KEK coming from a file or a helper (e.g. `age -d`).

---

## Phase 4: sharing unlock-requiring safes

Starts after `shared-domains` is merged. Sharing providers whose safe can always
be opened already works from phase 1.

### Step 4.1: user key pairs

- Tink HPKE key pair per user, public part in clear, private part wrapped by
  the user's password (and recovery) KEK; created at login when the password is
  available.

### Step 4.2: wrappings follow provider grants

- Granting a provider in an unlock-requiring safe wraps the safe keyset for the
  grantee (`user:<id>`); requires the owner to be unlocked.
- Grants to users without a key pair stay pending until the owner's next
  unlocked session.
- Revoking removes the wrapping, rotates the safe DEK and reseals its objects.
- UI suggests moving the provider to a dedicated safe when its safe holds other
  objects; shows key fingerprints.

**Tests first:** grant, use by grantee, revoke, post-revoke copy of the old
wrapping opens nothing sealed after rotation; pending grant completion.

---

## Deferred

Recorded so they are not forgotten, and not implemented until someone needs
them:

- Users configuring their own Vault, password manager or helpers (phase 2 only
  lets them select administrator-published safes).
- Free command lines for helpers (dangerous option for personal instances).
- `store` / `erase` operations for exec helpers.
- KMS-encrypted instance keyset (AWS / GCP KMS).
- OIDC users in phase 3 (dedicated passphrase or passkey PRF).
- Per-object keys (sharing one provider without its whole safe).
- Per-field safe choice.
- Caching unwrapped keysets or external store reads.
- Caching, per type, whether a struct can reach a `Secret` (`Walk` and the
  copy of `OpenCopy` look at the values each time).
- Objects where a struct holding secrets is reached twice (shared pointers,
  cycles): refused today, since one secret under two paths cannot match two
  associated data.
