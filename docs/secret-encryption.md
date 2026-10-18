# Encrypting stored credentials

happyDomain stores credentials on behalf of its users:

- the API keys, passwords and TSIG keys of their DNS providers;
- the signing secret and the header values (an `Authorization` token, say) of
  their webhooks;
- the endpoint of their UnifiedPush channels, enough to push to their device;
- the checker options documented as secret, such as the API keys of the
  blacklist sources, whether set by a user or by an administrator for every
  user.

By default they are stored in clear in the database, as they always were.

The `instance` policy encrypts them under a keyset held by the server, outside
the database. A leaked database dump or backup, or a database hosted apart from
happyDomain, no longer gives the credentials away. It does not protect against
a compromised happyDomain server, which holds the keyset.

## Limits of this version

Before enabling it, know that:

- the migrations below (`reseal`, `rewrap`, `drop-safes`) expect a single
  happyDomain process on the database. With several, a reseal may overwrite an
  update made at the same time by another process;
- when a credential no longer decrypts, users get a generic error.

## Generating the keyset

```
happydomain secret-keyset generate /etc/happydomain/secret-keyset.json
```

The keyset is a cleartext [Tink](https://developers.google.com/tink) JSON
keyset, which `tinkey` can also read. The file is created readable by its owner
only; make sure that owner is the user happyDomain runs as. happyDomain warns
at startup when the file can be read by other users. The command never
overwrites an existing file.

`happydomain secret-keyset info <file>` lists the keys of a keyset, never their
material.

The same keyset can be generated with
[`tinkey`](https://developers.google.com/tink/tinkey-overview), holding one
AES256-GCM key:

```
(umask 077 && tinkey create-keyset --key-template AES256_GCM \
  --out-format json --out /etc/happydomain/secret-keyset.json)
```

`tinkey` creates the file with the permissions of the current umask: the
`umask 077` makes it readable by its owner only from the start, and the
parentheses keep it from changing the umask of the shell.
`tinkey list-keyset --in /etc/happydomain/secret-keyset.json` lists its keys,
as `happydomain secret-keyset info` does.

With Docker, mount the file as a secret rather than baking it into an image,
readable by the `happydomain` user of the image (uid 15353) only. A Swarm
secret is mounted with mode `0444` unless told otherwise: set `uid: "15353"`
and `mode: 0400` in the long syntax of `secrets`. A Compose secret read from a
file is bind-mounted and keeps the owner and mode of that file on the host.

## Backing up the keyset

**Losing the keyset loses every credential it protects.** Back it up, and keep
that copy apart from the database backups: a backup holding both is no better
than a database in clear.

Back it up again after each rotation: once the users' keys are re-encrypted
under the new key, a copy of the keyset taken before no longer opens them.

If the keyset is lost anyway, see [If the keyset is lost](#if-the-keyset-is-lost).

## Enabling encryption

```
secret-keyset-file=/etc/happydomain/secret-keyset.json
secret-policy=instance
```

or `HAPPYDOMAIN_SECRET_KEYSET_FILE` and `HAPPYDOMAIN_SECRET_POLICY`.

From then on, every credential saved is encrypted. Those already stored are
encrypted the next time their provider, channel or options are saved; to do it
at once, see [Migrating existing credentials](#migrating-existing-credentials).

Nothing changes for users.

## Keystores

A keystore is where the credentials of a user are sealed. Each user has:

- the keystores of the instance, such as `instance`: happyDomain creates
  them on first use, and only the administrator deletes them;
- the personal keystores they create themselves, under a name, from a kind
  the instance offers (none in this version: see
  [Adding a kind of keystore](#adding-a-kind-of-keystore)).

The settings of a user list their keystores, with what each holds. There they
choose a default keystore, migrate what one holds into another, and delete a
personal keystore once empty: one still holding credentials is refused.

Next to each secret field, a selector chooses the keystore of the object being
saved, shared by all its fields. Through the API, every write of a provider,
a notification channel or checker options takes a `keystore` query parameter.
The credentials of the object go there, those stored elsewhere included.

Without a choice, nothing moves:

- an object in a personal keystore stays there;
- an object in a keystore of the instance follows the policy;
- a new object goes to the default keystore of its owner, else under the
  policy.

Notification channels and checker options are used in the background, when a
scheduled check completes: they only go to a keystore opening without its
owner, such as `instance`. A default keystore that needs its owner gives way
to the policy for them.

The administrator manages the keystores of any user, and of the instance
itself (owner of the options of the admin scope), on the secrets page or
under `/api/secrets/owners/{owner}/keystores`. A reseal leaves the credentials
in a personal keystore where their owner put them.

## Startup checks

happyDomain refuses to start rather than fail credential by credential when:

- the `instance` policy is chosen without a keyset;
- the keyset does not open the check record stored in the database the first
  time a keyset was used with it: this is not the keyset the credentials were
  encrypted with;
- credentials are encrypted in the database but no keyset is configured;
- safes of a kind are stored in the database, but that kind is left out of
  `secret-store` (by default `plaintext` and every kind of safe): the
  credentials they hold would no longer decrypt;
- `secret-policy=plaintext` while `plaintext` is left out of `secret-store`.

To forbid credentials in clear, set `secret-store=instance` (leaving
`plaintext` out) and `secret-policy=instance`. Credentials already in clear
stay readable, and are encrypted as described below.

## Migrating existing credentials

The operations below are on the admin interface, by default the
`happydomain.sock` socket:

```
curl --unix-socket happydomain.sock http://localhost/api/secrets/status
curl --unix-socket happydomain.sock -X POST http://localhost/api/secrets/reseal
```

`/api/secrets/status` counts, for providers, notification channels and checker
options, the credentials:

- `clear`: stored in clear;
- `sealed`: encrypted, and decrypting;
- `unreadable`: encrypted, but no longer decrypting;
- `undecodable`: objects that could not be looked at, such as a provider or a
  channel of a type this version no longer knows. They may hold encrypted
  credentials.

It also lists, under `keys`, the keys of the keyset, with `kek` set to
`instance`, and how many users' keys each one protects.

Which checker options are secret comes from the checkers themselves. For a
checker this version does not load, the options stored encrypted are still
counted and resealed, but those stored in clear cannot be told apart from the
others: they are neither counted nor encrypted.

`/api/secrets/reseal` stores every credential the way the current policy stores
new ones. It reports, for each type of object, how many it looked at, changed
and failed, naming the failures under `errors`. An object that fails is left
as it was, and the others are resealed. It can be interrupted, and running it
again resumes the work.

## Rotating the keyset

1. Add a new key, which becomes the one used from then on. The previous keys
   stay in the keyset and keep decrypting what they encrypted:

   ```
   happydomain secret-keyset rotate /etc/happydomain/secret-keyset.json
   ```

   The file keeps its owner, group and mode. Back it up, as described in
   [Backing up the keyset](#backing-up-the-keyset).

2. Restart happyDomain: until then, it keeps protecting new users' keys with
   the previous key.

3. Re-encrypt the users' keys under the new key. The credentials themselves are
   not rewritten:

   ```
   curl --unix-socket happydomain.sock -X POST http://localhost/api/secrets/rewrap
   ```

   It reports how many users' keys it looked at, changed and failed. A user's
   key that fails, for instance because the key protecting it was already
   removed from the keyset, is left as it was, and the others are
   re-encrypted. It can be interrupted, and running it again resumes the work.

4. Check that the previous key protects nothing any more:

   ```
   curl --unix-socket happydomain.sock http://localhost/api/secrets/status
   happydomain secret-keyset info /etc/happydomain/secret-keyset.json
   ```

   It can then be removed with
   `happydomain secret-keyset delete-key /etc/happydomain/secret-keyset.json <key-id>`,
   once a copy of the keyset file is kept. A key still in use shows up as
   `MISSING` once removed, and what it protects no longer opens: put back the
   copy. Users' keys too damaged to tell which key protects them are counted
   under `UNREADABLE`.

Keep the removed keys as long as you may restore a backup taken before the
rewrap: the users' keys it holds are protected by them.

## Going back to clear

1. Set `secret-policy=plaintext`, keeping the keyset configured, and restart.
   Encrypted credentials still decrypt, and new ones are stored in clear.

2. Store the others in clear too, then check that none is left encrypted:

   ```
   curl --unix-socket happydomain.sock -X POST http://localhost/api/secrets/reseal
   curl --unix-socket happydomain.sock http://localhost/api/secrets/status
   ```

3. Delete the users' keys and the keyset check record:

   ```
   curl --unix-socket happydomain.sock -X POST http://localhost/api/secrets/drop-safes
   ```

   It refuses while a credential is still counted as `sealed` or `unreadable`,
   and while an object is counted as `undecodable`: those would be lost for
   good. Reseal again, have the unreadable credentials entered again, or
   delete the objects concerned, then run it again.

4. Remove `secret-keyset-file` from the configuration, and restart.

Backups taken before still hold encrypted credentials and the users' keys:
keep a copy of the keyset as long as you may restore one of them.

## If the keyset is lost

Only a copy of the keyset brings the credentials back: look for one first,
including a keyset file kept from before a rotation, which still opens what
was not rewrapped since. Neither the database nor the administrative backup
holds the keyset.

Without a copy, the credentials it protected are lost for good, and
happyDomain refuses to start, with or without a new keyset (see [Startup
checks](#startup-checks)). To start again, have happyDomain forget the lost
keyset:

1. Stop happyDomain, and back up the database.

2. Run, with the configuration happyDomain runs with (its configuration file,
   environment and flags):

   ```
   happydomain abandon-secret-keyset
   ```

   It tells how many users' keys it deletes, along with the check record of
   the keyset, and asks to type `abandon` to confirm. It refuses when the
   configured keyset still opens the check record: that keyset is not lost.
   A configured keyset file that is missing, damaged or another keyset is
   taken as lost.

   With Docker, run it in a container of the same image, on the same volumes
   and configuration, while the service is stopped:

   ```
   docker compose run --rm -i happydomain abandon-secret-keyset
   ```

3. Start happyDomain again, either:

   - with `secret-policy=plaintext` and no keyset: credentials entered again
     are stored in clear;
   - or with a new keyset (see [Generating the keyset](#generating-the-keyset))
     and `secret-policy=instance`: credentials entered again are encrypted
     under it. Back it up this time.

4. The credentials sealed under the lost keyset stay in the database, counted
   as `unreadable` by `/api/secrets/status`. Their providers, notification
   channels and checker options no longer work until their owners enter them
   again; saving them with the `••••••••` placeholder is refused. Tell your
   users which ones to fill again, or delete the objects concerned.

Administrative backups taken before still hold the deleted users' keys,
useless without the lost keyset. Restoring one on this instance reports them
as not opening, and does not restore them.

## Backups and deleted accounts

The administrative backup contains the credentials as they are stored:
encrypted under the `instance` policy, along with the users' keys, and in clear
otherwise. It restores on the same instance, or on another one configured with
the same keyset: a user's key that the configured keyset does not open is not
restored, and the restore reports it. It also carries the keystore each user
chose for their new secrets. The data export a user downloads never contains
credentials, nor safes.

Deleting a user, or a user deleting their account, deletes their keys: their
credentials no longer decrypt from the database. Their notification channels
are removed by `tidy`. Administrative backups taken before still hold those
keys, and decrypt them with the keyset for as long as both are kept. A key
left behind, for instance when the database was briefly unavailable, is
deleted by the next `tidy`.

## Adding a kind of keystore

A kind of keystore is a `secret.Store`, in a file of its own under
`internal/secret`, whose factory is listed in `builtinStores`. Besides
`Unlock`, it implements the interfaces of what it can do:

- `Creator`: its keystores are created by their owner, from the parameters
  its `CreateForm` describes. Several of a kind may belong to one owner;
- `Provisioner`: the instance creates its keystore for each owner on first
  use. Only such a store can be the policy;
- `Unattended`: its keystores open without their owner, as objects used in
  the background need. A store that does not implement it is taken as
  needing its owner;
- `StartupChecker`, `Rewrapper`, `KeyReporter`, `KEK`: see their
  documentation in `internal/secret/store.go`.

A store reaching outside happyDomain is only enabled on purpose: list it in
`secret-store`.
