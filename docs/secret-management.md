# Secret management

> **Status: phase 1 implemented, the rest is design.** This document records
> the decisions taken before implementation; the phasing at the end says in
> which order the rest will land. How to operate what exists is in
> [secret-encryption.md](secret-encryption.md).

happyDomain stores credentials on behalf of its users: the API keys of their DNS
providers, the signing secret of their notification webhooks, the options of
some checkers. Today they are stored in clear, in the same records as the rest
of the configuration.

The goal is to let each of these secrets live in one of several places, chosen
per object within limits set by the administrator:

- in clear in the database, which stays the default;
- encrypted in the database, under a key held by the instance;
- encrypted in the database, under a key derived from the user's password, so
  that the administrator cannot read it at rest;
- outside happyDomain altogether, in HashiCorp Vault or a password manager with
  an API, so that happyDomain is no longer the one responsible for it.

The hard part is not encrypting: it is changing where a secret lives or which
key protects it after the fact. Rotating the instance key, changing a password,
recovering an account, moving a provider to Vault: all of them have to be
possible without losing a secret, and without rewriting every record each time.

## Threat model

| Protection | Protects against | Does not protect against |
|---|---|---|
| Instance key | a leaked database dump or backup; a database hosted apart from the application | a compromised application server, which holds the key |
| Password-derived key | the above, plus the administrator and a compromise of the instance at rest | a compromised server *while running*, which sees the password at login |
| External store (Vault) | moves the responsibility to a system built for it, with its own audit log and revocation | the credentials happyDomain uses to reach the store are themselves secrets |

The password-derived mode is what lets a public instance state that its
operator cannot read the credentials of its users.

## What is a secret

- DNS provider credentials: every `happydns.Secret` field in `providers/`.
- Notification channel configuration: the webhook signing secret and header
  values, the UnifiedPush endpoint (a capability URL).
- Checker options whose documentation sets `Secret`.

Not in scope: password hashes (already one-way), the recovery key of an account
(to be hashed rather than encrypted), and the instance configuration (SMTP
password, OIDC client secret), which lives in files or environment variables.

### Background access

Provider credentials are only used during interactive requests. The other two
are not:

- notification channels are used by `Dispatcher.OnExecutionComplete`, run when a
  scheduled check completes;
- checker options are merged by `Engine.runPipeline` during scheduled runs.

A secret protected by the user's password cannot be opened when the user is not
there. Each object type therefore declares whether it needs background access,
each safe declares when it can be opened (see
[Who can open a safe](#who-can-open-a-safe)), and the policy refuses the
combinations that cannot work when the secret is configured, rather than
failing later in the scheduler. Protecting them with both the instance key and
the password would make no sense: the instance key alone would open them.

## Field-level sealing

Secrets are sealed field by field, never as a whole blob. A future provider
backed by a SQL database will store columns, not a JSON document, and the
non-secret fields of a provider (an endpoint, a zone name) must stay readable.

A sealed field is a string:

```
hds:1:<safe id>:<payload>
```

- A value without the prefix is plaintext. Existing data stays valid as is and
  plaintext remains the default without any migration.
- The prefix is how a sealed value is recognized. A plaintext credential that
  happens to start with `hds:1:` is not a realistic accident; a user crafting
  one only harms their own configuration.
- The value names its safe, so it can always be opened without consulting any
  policy (see [Resolution](#resolution-happens-when-sealing-only)).

## The `happydns.Secret` type

Secret fields change type from `string` to `happydns.Secret`. This touches
roughly a hundred fields across `providers/`, mechanically, and buys guarantees
that a tag cannot give:

- **Fail closed on write.** A `Secret` is either freshly set in clear, sealed,
  or opened. `MarshalJSON` refuses to encode one that never went through
  `Seal`. A code path that forgets to seal (a restore, a future import) fails
  instead of writing plaintext. Sealing into the plaintext safe emits the raw
  value, compatible with existing records.
- **No leak through logs.** `String`, `GoString` and `Format` print
  `••••••••`, so `%v` and `%+v` on a provider are harmless.
- **Explicit access.** The clear value is only read through `Reveal()`, in
  `InstantiateProvider` and the like, where it can be found with grep and
  reviewed.
- **The type marks the secret.** A secret no longer depends on a `secret` tag
  someone might forget. A test still walks every provider and fails on a
  field whose name matches `key|token|secret|password` and is not a `Secret`.
- **SQL ready.** The same type will implement `sql.Scanner` and
  `driver.Valuer`, with the same fail-closed rule.
- **Editing is unchanged.** `forms.RedactSecrets` sends `RedactedSecret` to the
  client; `forms.MergeSecrets` carries the stored sealed value forward when the
  client echoes the placeholder back, without opening it.

Checker options are dynamic maps rather than structs: they are sealed by key,
using `CheckerOptionDocumentation.Secret`. Finding the secret fields
(struct type or schema) is a layer of its own; sealing works on one value and
its context.

## Safes

A **safe** is a named protection owned by a user, and chosen per object: a user
can keep the provider of `domain.corp` in a safe backed by the company Vault and
the provider of `domain.perso` in a safe protected by their password. All the
secret fields of one object go to the same safe.

```go
type Safe struct {
    Id, Owner Identifier
    Name      string          // "Personal", "Corp (Vault)"
    Store     string          // "inline", "vault-kv", ...
    StoreCfg  json.RawMessage // how to reach the store, itself sealed in an instance safe
    Keyring   []WrappedKeyset // one Tink keyset, wrapped by 0..N key encryption keys
}
```

The name avoids "vault", which would be confusing next to HashiCorp Vault.

### Two independent axes

A safe combines two choices that are easy to conflate:

- **Where the data lives**, a `SecretStore`:
  - `inline`: the payload is the ciphertext itself, in the database;
  - `vault-kv`: the payload is a path and version in Vault KV, which encrypts
    and audits;
  - `external`: a reference the user points at in their own password manager,
    read only: happyDomain never deletes nor rotates it. Password managers are
    reached through `exec` helpers (`op read`, `bw get`, `pass show`) rather
    than one SDK per vendor;
  - `file`: the payload is a path on the happyDomain host, the secret is the
    content of the file: Docker and Kubernetes secrets, systemd credentials;
  - `exec`: the payload is an argument given to a helper program, which prints
    the secret: `pass`, `op`, `bw`, `age -d`.
- **What protects the keys**, a `KeyProvider` (key encryption key, KEK):
  - `instance`: a Tink keyset held by the instance, possibly itself encrypted by
    a KMS;
  - `vault-transit`: a Vault Transit key, which never touches happyDomain's disk;
  - `password`: derived from the user's password with argon2id;
  - `recovery`: a random key given to the user once, to recover from a
    forgotten password;
  - `file`: a key read from a file, typically on a USB stick;
  - `exec`: a helper program that unwraps the keyset, such as `age -d` with an
    identity on a YubiKey;
  - `user:<id>`: the key pair of another user the safe is shared with (see
    [Sharing](#sharing)).

Vault appears on both axes in different roles: Transit is a KEK, KV is a store.
Local files and programs do too, see
[Local files and programs](#local-files-and-programs).

```go
type SecretStore interface {
    Scheme() string
    Capabilities() StoreCapabilities // read, write, delete
    // ValidateRef checks a reference typed by the user, for stores working by
    // reference.
    ValidateRef(ctx context.Context, sc SecretContext, ref string) error
    Put(ctx context.Context, sc SecretContext, data []byte) (ref string, err error)
    Get(ctx context.Context, sc SecretContext, ref string) ([]byte, error)
    Delete(ctx context.Context, sc SecretContext, ref string) error
}

type KeyProvider interface {
    ID() string
    // KEK returns one of the errors of "When a secret cannot be opened" when
    // the key cannot be obtained in this context, e.g. a password-protected
    // safe outside of an unlocked session.
    KEK(ctx context.Context, owner Identifier, params json.RawMessage) (tink.AEAD, error)
}
```

A key provider never encrypts data, only keysets.

A store declares what it can do rather than whether it owns the data: `external`
and most `file` stores are read only, an `exec` helper may implement writing and
deleting or not. happyDomain never deletes nor rotates what it cannot write.

### References instead of values

When the store of a safe works by reference (`external`, `file`, an `exec`
helper that cannot write), what the user types in a `Secret` field is the
reference, checked by `ValidateRef`, not the secret. The form keeps a single
field whose label follows the chosen safe.

### The keyring is the pivot

Each safe holds one Tink keyset, its data encryption key (DEK), stored wrapped
by one or more KEKs:

```go
type WrappedKeyset struct {
    KEK    string          // key provider id
    Params json.RawMessage // argon2 salt and costs, Transit key name...
    Blob   []byte
}
```

This is what makes changes cheap: they rewrap one record per safe and never
touch the sealed fields.

| Change | What is rewritten |
|---|---|
| Rotate the instance key | each safe's `instance` entry |
| Change password | the safe's `password` entry, the old password is already required |
| Forgotten password | unlock through `recovery`, rewrap `password` |
| Protect a safe with the password | add `password`, drop `instance` |
| Rotate the DEK | Tink rotation: new primary key, fields resealed lazily or by batch |
| Delete a safe or an account | nothing to rewrite: dropping the keyring makes every sealed value unreadable, backups included |

For the `inline` store, the keyring is mandatory. For external stores it is
optional: encrypting before sending to Vault KV means that even a Vault
administrator cannot read a password-protected safe. The policy decides, not the
store.

Credentials to reach a user's own store (their Vault token, their password
manager API key) are secrets too: they are stored `inline` in one of the user's
safes. The recursion stops there.

## Resolution happens when sealing only

```
Seal: object.SafeId -> user default safe -> instance default kind
      then: allowed by the admin policy, compatible with the object type
Open: the sealed value names its safe; no policy involved
```

- Changing a default or a placement policy never prevents reading an existing
  secret.
- "No policy" means no *placement* policy. Security restrictions (enabled
  stores and key providers, allowed directories and helpers) are checked again
  on every open: a safe created under a looser configuration must stop running
  programs as soon as the administrator forbids them. Disabling a store
  therefore makes the secrets relying on it unreadable; the migration command
  lists them before the configuration changes.
- An object with an explicit `SafeId` keeps it. An object with none inherits:
  when the user changes their default safe, inherited objects are resealed in
  the same request, while the user has everything unlocked.
- The instance default names a *kind* of safe rather than a safe: a user
  inheriting it gets their own safe of that kind, created on first use.
- The administrator lists the stores and key providers users may choose from.

## Associated data

Every encryption binds the ciphertext to where it belongs, as AEAD associated
data:

```
hds:1 | safe id | owner | object type | object id | field path
```

Without it, anyone able to write to the database could copy the sealed API key
of one user into the provider of another and have it decrypted. Object
identifiers are therefore generated before sealing; today the storage assigns
them (`FindIdentifierKey`).

## Where sealing happens

The clear `happydns.Provider` currently travels everywhere: the provider
middleware puts it in the gin context for every `/providers/:pid` route, the
user API redacts it by remembering to call `RedactSecrets`, the admin API and
the backup emit real values, and restoring a backup writes to the storage
without going through the use cases. Sealing in the storage is not an option
either: it has no context to tell whether a session is unlocked, and every
reader of the storage would get plaintext.

- **Seal in write use cases only.** They have the context (user, unlocked
  keyrings) and the `SecretContext`.
- **Open only where happyDomain connects on the user's behalf.** The code
  already has that discipline for the destination check:
  - `provider.Service.instantiate`, which `DefaultProviderValidator` should
    reuse rather than duplicate;
  - the notifier `Send`;
  - the checker options merge.
  Everywhere else (gin context, middleware, API, backup) objects carry sealed
  values only.
- **The storage fails closed** through `Secret.MarshalJSON`.
- **The admin API redacts** secrets like the user API. It cannot simply emit
  what is stored: a legacy plaintext value is a clear `Secret`, which refuses
  to be encoded, and the operator has no use for ciphertext.
- **Backups** copy stored records as they are, sealed values included, along
  with the safes. A backup restores on the same instance, or on another one
  holding the instance keyset. A plaintext export, if needed, is an explicit
  action, for safes that can be opened.
- **Deleting** an object, a safe or an account removes owned external
  references after the database write; orphans left by a failure are collected
  by the existing `tidy` jobs.

## Rotation and migration

- The instance keyset holds several keys, one primary. Rotation (with `tinkey`
  or a happyDomain command) rewraps the safes; old keys stay usable for
  decryption until nothing uses them.
- A batch command reseals or rewraps what is left, can be interrupted and
  rerun, and reports what still depends on an old key, so the operator knows
  when it can be removed.
- Moving from plaintext to encrypted, or back, is the same mechanism: records
  are resealed on write, and the batch command handles the rest.
- A check value encrypted under the instance keyset is verified at startup: a
  wrong or missing key stops the server instead of failing provider by
  provider.

## Unlocking a safe

Some key providers need something only the user has: a password, a key on a
USB stick, a touch on a YubiKey. Asking for it at every provider call is not
an option, so such a safe is unlocked once per session instead. The session
store keeps its records in the database, so the unlocked keyring cannot simply
be kept in the session.

- Unlocking is an explicit action (`POST /safes/:id/unlock`) that calls the key
  provider of the safe: password typed, file read, program run. Logging in is
  the case where password-protected safes are unlocked automatically, the KEK
  being derived from the password with argon2id.
- The unwrapped keyset is wrapped again under `HKDF(session token)` and kept
  with the session.
- Only whoever presents the token can use it: the database alone is not
  enough, nor is the instance key. This requires that the session token is
  never stored, which is the case since sessions are stored under the hash of
  their token.
- Logging out, the session expiring, or locking the safe explicitly drops that
  wrapping. Removing the USB stick prevents the next unlock, not the use of a
  session already unlocked.
- Machine tokens (API keys) work the same way: when creating one from an
  unlocked session, the user can grant it access to some safes. Revoking the
  token deletes that wrapping.
- Users without a password (OIDC) would need a dedicated passphrase or a passkey
  PRF; this is left for later.

## Who can open a safe

Each safe declares when it can be opened, from its store and key providers:

| When | Safes |
|---|---|
| Always | `instance`, `vault-transit`; Vault KV reached with instance credentials |
| In an unlocked session | `password`, `file` and `exec` key providers |
| Best effort | `file` and `exec` stores: it depends on what the user has plugged in or mounted |

Object types needing background access accept safes that can always be opened,
and best effort ones with a warning.

### When a secret cannot be opened

The error says what to do, rather than surfacing as a generic provider failure.
Phase 1 has two:

- `ErrUnopenable`: the value will never open where it is (its safe deleted or
  another owner's, the value moved or malformed): the user enters it again;
- `ErrSafeUnavailable`: its safe exists but cannot be opened with the current
  configuration (no keyset, a keyset lacking its key, a damaged safe): the
  administrator repairs it, the user is told to reach them, and the value is
  neither stored in clear nor taken for lost meanwhile.

They are turned into what the user is told in one place, `secret.UserError`,
called wherever secrets are opened or sealed on a user's behalf. Later phases
add:

- `ErrLocked`: the user has to act: unlock the safe, plug the key in;
- `ErrUnavailable`: a transient failure (Vault down, helper timeout), worth
  retrying;
- `ErrDenied`: the current configuration forbids this safe.

The interface turns them into messages such as "plug your key in and retry".

## Local files and programs

Files and programs on the happyDomain host fit both axes, with different
properties:

| | As a store | As a key provider |
|---|---|---|
| File | the payload is a path, the secret is its content | the file holds a key wrapping the keyset of the safe |
| Program | the payload is an argument, the program prints the secret | the program unwraps the keyset |
| What happyDomain holds | a reference only | the sealed fields, in the database |
| Calls | one per secret, at every use | one per unlock of the safe |
| Removing the USB stick | cuts access immediately | prevents the next unlocks, not an already unlocked session |

For an encrypted USB stick or `age` with a YubiKey, the key provider is almost
always the right choice: as a store, every provider call would require touching
the YubiKey. With `age` as a key provider, wrapping only needs the recipient,
the public key: a safe can be created without the stick at hand.

### Restrictions

Letting a user name a file to read or a program to run hands them the
privileges of the happyDomain process: on a multi-tenant instance, any account
would get a shell. Therefore:

- Both are disabled by default, and a global switch keeps them off on
  multi-tenant instances.
- Programs are named helpers declared by the administrator
  (`--secret-helper pass='/usr/bin/pass show {ref}'`). The user picks a helper
  and only supplies `{ref}`, passed as one argument, never through a shell.
- A free command line is only available behind an explicitly dangerous option,
  meant for a personal instance.
- Files are read from directories listed by the administrator, which may
  contain `{user}` (`/var/lib/happydomain/secrets/{user}/`) so that a user
  cannot point at the files of another. Access goes through `os.OpenRoot`,
  which blocks escaping through `..` or symbolic links. A file readable by
  everyone is refused, as ssh does.
- The administrator can also publish shared safes ("Docker secrets", "pass")
  that users only have to select.

### Helper protocol

Modeled on git credential helpers:

- three operations, `get`, `store` and `erase`; only `get` is required, and the
  capabilities of the store follow what the helper implements;
- the context (owner, acting user, object, field, reference) on standard input,
  as `key=value` lines;
- the secret on standard output, bounded in size;
- a timeout: a helper waiting for a pinentry on a headless server fails cleanly
  instead of hanging;
- a scrubbed environment;
- standard error is never sent back to the user, and only logged at debug
  level: it may contain parts of the secret.

## Sharing

Once domains can be shared (`shared-domains` branch), an owner can grant another
user access to the provider of a domain, and zone operations resolve
credentials through `GetProviderForZone`, for the owner or a grantee. The
secrets stay the owner's: the associated data keeps naming the owner, and the
grantee never receives the clear value, which the API redacts as for anyone.

What changes is who must be able to open the safe:

- Safes that can always be opened need nothing: the authorization is the
  provider grant, and the server opens the safe for the grantee as for the
  owner.
- Safes that need an unlock are only unlocked in the owner's session. For a
  grantee to use them, the keyset of the safe has to be wrapped for the grantee
  too.

This requires a key pair per user: a Tink hybrid (HPKE) key pair whose public
part is stored in clear and whose private part is wrapped by the user's own key
providers (password, recovery), unlocked at login like a password-protected
safe. Sharing a provider then wraps the keyset of its safe under the grantee's
public key, as one more `WrappedKeyset` entry (`user:<id>`). The owner has to
be unlocked when sharing, the grantee does not.

- The unit is the safe, not the object: wrapping a safe for a grantee gives
  them the key of every object in it. Authorization still restricts them to
  what is shared, but cryptographically the safe is open to them. When the
  provider to share lives in a safe holding other objects, the interface
  offers to move it to a dedicated safe first (resealed while the owner is
  unlocked) or to share the whole safe. Per-object keys can come later without
  changing the format.
- Safe wrappings follow provider grants, the way provider grants follow domain
  shares: added when a grant needs them, removed when no remaining grant on a
  provider of that safe does. Removing a wrapping is followed by a rotation of
  the keyset of the safe, so that a copy taken before opens nothing sealed
  after.
- A grant to a user who has no key pair yet (not logged in since, or without a
  password) stays pending: it completes during the owner's next unlocked
  session once the grantee has one.
- Public keys are served by the instance. An administrator substituting one at
  sharing time would receive the wrapping: this is the active compromise the
  threat model already excludes, and a fingerprint shown to both users allows
  checking it.
- External stores are reached with the owner's credentials when a grantee uses
  the provider, so the Vault audit log names happyDomain and the owner. The
  acting user is passed along (helper context, Vault request metadata) so that
  it can be recorded.
- Local stores run with the owner's configuration: the owner's USB stick has to
  be plugged in when the grantee publishes. Such safes are best effort for
  grantees too.

## Tink coverage

| Need | Tink | Notes |
|---|---|---|
| Field encryption with associated data | yes | `tink.AEAD` |
| Keysets with rotation | yes | key id in the ciphertext prefix |
| DEK wrapped by a KEK | yes | `keyset.WriteWithAssociatedData` |
| Vault Transit KEK | yes | `tink-go-hcvault`, supported auth methods to check |
| AWS / GCP KMS KEK | yes | `tink-go-awskms`, `tink-go-gcpkms` |
| Instance keyset management | yes | `tinkey` |
| Password-derived key | no | argon2id from `x/crypto`, imported as an AES-GCM key |
| Several KEKs for one keyset | no | store one wrapping per KEK |
| User key pairs for sharing | yes | hybrid encryption (HPKE) |
| Recovery key | no | 256 random bits, encoded for humans |
| Vault KV, password managers | no | not cryptography: `vault/api`, and `exec` helpers for password managers |

## Phasing

1. `happydns.Secret` on every secret field, safes, instance key provider,
   `inline` store, rotation command. This alone answers the audit.
   Implemented.
   Sharing a provider whose safe can always be opened needs nothing more.
2. Vault KV and Vault Transit, then external references to password managers,
   local files and helper programs as stores.
3. Unlocking safes: password-derived key provider, recovery keys, `file` and
   `exec` key providers.
4. User key pairs, to share providers whose safe needs an unlock.

The abstractions of phase 1 are shaped for the next two, so that adding them
does not change the sealed format nor the code paths.
